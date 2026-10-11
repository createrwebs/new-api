package service

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

type RouteFailureClass string

const (
	// ClassSafePreExecutionFailure: Upstream connect/dial/DNS/TLS error occurred before request transmission.
	// Safe to fallback without duplicate execution or billing amplification.
	ClassSafePreExecutionFailure RouteFailureClass = "SAFE_PRE_EXECUTION_FAILURE"

	// ClassExplicitProviderFailure: Explicit upstream HTTP 429, 502, 503, 504.
	// Safe to fallback according to routing policy.
	ClassExplicitProviderFailure RouteFailureClass = "EXPLICIT_PROVIDER_FAILURE"

	// ClassAmbiguousExecutionFailure: Request body was transmitted, but connection was reset, timed out, or returned EOF.
	// Upstream may have accepted and executed the request. Fallback is strictly forbidden by default to prevent Denial-of-Wallet.
	ClassAmbiguousExecutionFailure RouteFailureClass = "AMBIGUOUS_EXECUTION_FAILURE"

	// ClassFallbackEligible: General alias for backward-compatible fallback eligibility checks.
	ClassFallbackEligible RouteFailureClass = "FALLBACK_ELIGIBLE"

	// ClassProviderAuthFailure: Upstream provider credential failure (401/403); channel should be auto-banned and failed over.
	ClassProviderAuthFailure RouteFailureClass = "PROVIDER_AUTH_FAILURE"

	// ClassTerminalClient: Client request issue (400, 404, 422); retrying would produce identical error.
	ClassTerminalClient RouteFailureClass = "TERMINAL_CLIENT"

	// ClassTerminalLocalAuth: Local Tora auth failure; client is unauthorized.
	ClassTerminalLocalAuth RouteFailureClass = "TERMINAL_LOCAL_AUTH"

	// ClassTerminalQuota: Local Tora quota exhausted; cannot proceed.
	ClassTerminalQuota RouteFailureClass = "TERMINAL_QUOTA"

	// ClassTerminalCommitted: Response stream has already been committed to downstream client; failover is forbidden.
	ClassTerminalCommitted RouteFailureClass = "TERMINAL_COMMITTED"

	// ClassClientCancelled: Downstream client aborted the request.
	ClassClientCancelled RouteFailureClass = "CLIENT_CANCELLED"
)

type RouteDecision struct {
	Class         RouteFailureClass
	AllowFallback bool
	Reason        string
	StatusCode    int
}

// ClassifyRouteError evaluates an error from an upstream or local attempt and determines whether fallback is permissible.
// responseCommitted indicates whether headers or stream bytes have already been sent to downstream client.
func ClassifyRouteError(ctx context.Context, err *types.NewAPIError, responseCommitted bool) RouteDecision {
	if err == nil {
		return RouteDecision{
			Class:         ClassFallbackEligible,
			AllowFallback: false,
			Reason:        "no_error",
			StatusCode:    http.StatusOK,
		}
	}

	// 1. Check if client cancelled the request context
	if ctx != nil && errors.Is(ctx.Err(), context.Canceled) {
		return RouteDecision{
			Class:         ClassClientCancelled,
			AllowFallback: false,
			Reason:        "client_cancelled",
			StatusCode:    499, // Client Closed Request
		}
	}

	// 2. If response has already been committed downstream, fallback is strictly forbidden
	if responseCommitted {
		return RouteDecision{
			Class:         ClassTerminalCommitted,
			AllowFallback: false,
			Reason:        "response_already_committed",
			StatusCode:    err.StatusCode,
		}
	}

	// 3. Local Tora quota or auth errors
	if types.IsSkipRetryError(err) {
		switch err.StatusCode {
		case http.StatusUnauthorized:
			return RouteDecision{Class: ClassTerminalLocalAuth, AllowFallback: false, Reason: "local_auth_unauthorized", StatusCode: 401}
		case http.StatusForbidden:
			return RouteDecision{Class: ClassTerminalLocalAuth, AllowFallback: false, Reason: "local_auth_forbidden", StatusCode: 403}
		case http.StatusPaymentRequired:
			return RouteDecision{Class: ClassTerminalQuota, AllowFallback: false, Reason: "local_quota_exhausted", StatusCode: 402}
		}
	}

	// 4. Provider credential failure (upstream 401/403)
	// Upstream provider rejected the channel key. Auto-ban the channel and allow fallback to another channel/route.
	if err.StatusCode == http.StatusUnauthorized || (err.StatusCode == http.StatusForbidden && !types.IsSkipRetryError(err)) {
		return RouteDecision{
			Class:         ClassProviderAuthFailure,
			AllowFallback: true,
			Reason:        "provider_credential_invalid",
			StatusCode:    err.StatusCode,
		}
	}

	// 5. Upstream Rate Limit (429) or Server Overload / Gateway Errors (502, 503, 504)
	if err.StatusCode == http.StatusTooManyRequests {
		return RouteDecision{
			Class:         ClassFallbackEligible,
			AllowFallback: true,
			Reason:        "upstream_rate_limited",
			StatusCode:    429,
		}
	}
	if err.StatusCode == http.StatusBadGateway ||
		err.StatusCode == http.StatusServiceUnavailable ||
		err.StatusCode == http.StatusGatewayTimeout {
		return RouteDecision{
			Class:         ClassFallbackEligible,
			AllowFallback: true,
			Reason:        "upstream_server_error",
			StatusCode:    err.StatusCode,
		}
	}

	// 6. Network connection failure / pre-response network error / timeout / ErrorCodeDoRequestFailed
	if err.GetErrorCode() == types.ErrorCodeDoRequestFailed || err.StatusCode == 0 || err.StatusCode < 100 || err.StatusCode >= 600 || (err.StatusCode == http.StatusInternalServerError && err.Err != nil) {
		underlyingErr := err.Err
		if underlyingErr == nil && err.Error() != "" {
			underlyingErr = errors.New(err.Error())
		}
		isSafePre, isAmbiguous, netReason := classifyNetworkFailure(underlyingErr)
		if isAmbiguous {
			// AMBIGUOUS_EXECUTION_FAILURE: Request body was transmitted; remote may be executing.
			// Fallback is strictly forbidden by default to prevent Denial-of-Wallet.
			return RouteDecision{
				Class:         ClassAmbiguousExecutionFailure,
				AllowFallback: false,
				Reason:        netReason,
				StatusCode:    http.StatusBadGateway,
			}
		}
		if isSafePre {
			// SAFE_PRE_EXECUTION_FAILURE: Connect/dial/DNS/TLS error before request transmission.
			return RouteDecision{
				Class:         ClassSafePreExecutionFailure,
				AllowFallback: true,
				Reason:        netReason,
				StatusCode:    http.StatusBadGateway,
			}
		}
		// Default conservative non-fallback
		return RouteDecision{
			Class:         ClassAmbiguousExecutionFailure,
			AllowFallback: false,
			Reason:        "ambiguous_upstream_drop",
			StatusCode:    http.StatusBadGateway,
		}
	}

	// 7. Client-side terminal errors (400 Bad Request, 404 Model Not Found, 422 Unprocessable, 413, 415)
	// These must never trigger route fallback because the request itself cannot be serviced by any provider.
	if err.StatusCode == http.StatusBadRequest ||
		err.StatusCode == http.StatusNotFound ||
		err.StatusCode == http.StatusUnprocessableEntity ||
		err.StatusCode == http.StatusRequestEntityTooLarge ||
		err.StatusCode == http.StatusUnsupportedMediaType {
		return RouteDecision{
			Class:         ClassTerminalClient,
			AllowFallback: false,
			Reason:        "client_terminal_error",
			StatusCode:    err.StatusCode,
		}
	}

	// 8. Check operation_setting dynamic retry rules
	if operation_setting.ShouldRetryByStatusCode(err.StatusCode) {
		return RouteDecision{
			Class:         ClassFallbackEligible,
			AllowFallback: true,
			Reason:        "dynamic_retry_status_matched",
			StatusCode:    err.StatusCode,
		}
	}

	// 9. Other client-side errors (4xx)
	if err.StatusCode >= 400 && err.StatusCode < 500 {
		return RouteDecision{
			Class:         ClassTerminalClient,
			AllowFallback: false,
			Reason:        "client_error",
			StatusCode:    err.StatusCode,
		}
	}

	// Default fallback: non-retryable 5xx or unclassified
	return RouteDecision{
		Class:         ClassTerminalClient,
		AllowFallback: false,
		Reason:        "unclassified_error",
		StatusCode:    err.StatusCode,
	}
}

func classifyNetworkFailure(err error) (isSafePre bool, isAmbiguous bool, reason string) {
	if err == nil {
		return false, true, "unknown_network_error"
	}
	errStr := strings.ToLower(err.Error())

	// 1. Check for client cancellation
	if errors.Is(err, context.Canceled) || strings.Contains(errStr, "context canceled") {
		return false, false, "client_cancelled"
	}

	// 2. DNS resolution failure (Pre-execution)
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) || strings.Contains(errStr, "no such host") || strings.Contains(errStr, "lookup ") {
		return true, false, "dns_resolution_failure"
	}

	// 3. TCP Connect refused (Pre-execution)
	if errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(errStr, "connection refused") {
		return true, false, "tcp_connect_refused"
	}

	// 4. TLS Handshake failure before request transmission (Pre-execution)
	var recordHeaderErr tls.RecordHeaderError
	if errors.As(err, &recordHeaderErr) ||
		strings.Contains(errStr, "tls:") ||
		strings.Contains(errStr, "certificate") ||
		strings.Contains(errStr, "handshake failure") {
		return true, false, "tls_handshake_failure"
	}

	// 5. Dial-specific OpErrors (connection attempt failed before write)
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return true, false, "dial_failure"
		}
		if opErr.Op == "read" {
			// Read error after connection was established and request was sent!
			return false, true, "read_timeout_or_reset"
		}
		if opErr.Op == "write" {
			// Write error during transmission: ambiguous whether remote read partial body
			return false, true, "write_connection_broken"
		}
	}

	// 6. Timeout while awaiting headers or reading response body (Ambiguous Execution)
	if strings.Contains(errStr, "awaiting headers") ||
		strings.Contains(errStr, "timeout awaiting response") ||
		strings.Contains(errStr, "client.timeout") ||
		errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(errStr, "deadline exceeded") {
		return false, true, "response_timeout_after_transmission"
	}

	// 7. Connection reset or EOF after request was sent (Ambiguous Execution)
	if errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "eof") ||
		strings.Contains(errStr, "broken pipe") {
		return false, true, "connection_reset_after_transmission"
	}

	// Conservative default: if we cannot prove request was never transmitted, treat as ambiguous
	return false, true, "ambiguous_execution_failure"
}

