package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
)

func TestClassifyRouteError(t *testing.T) {
	ctx := context.Background()

	// 1. Success case
	d := service.ClassifyRouteError(ctx, nil, false)
	assert.False(t, d.AllowFallback)
	assert.Equal(t, service.ClassFallbackEligible, d.Class)

	// 2. Client cancellation
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()
	d = service.ClassifyRouteError(cancelledCtx, types.NewErrorWithStatusCode(errors.New("canceled"), "canceled", 499), false)
	assert.False(t, d.AllowFallback)
	assert.Equal(t, service.ClassClientCancelled, d.Class)

	// 3. Response committed (streaming started)
	d = service.ClassifyRouteError(ctx, types.NewErrorWithStatusCode(errors.New("stream drop"), "stream_error", 500), true)
	assert.False(t, d.AllowFallback)
	assert.Equal(t, service.ClassTerminalCommitted, d.Class)

	// 4. Upstream 429 Rate Limit (safe for fallback)
	d = service.ClassifyRouteError(ctx, types.NewErrorWithStatusCode(errors.New("rate limit"), "rate_limit", 429), false)
	assert.True(t, d.AllowFallback)
	assert.Equal(t, service.ClassFallbackEligible, d.Class)

	// 5. Upstream 503 Server Error (safe for fallback)
	d = service.ClassifyRouteError(ctx, types.NewErrorWithStatusCode(errors.New("overloaded"), "service_unavailable", 503), false)
	assert.True(t, d.AllowFallback)
	assert.Equal(t, service.ClassFallbackEligible, d.Class)

	// 6. Upstream 401 Provider Invalid Key (allow fallback + auto-ban)
	d = service.ClassifyRouteError(ctx, types.NewErrorWithStatusCode(errors.New("invalid api key"), "invalid_api_key", 401), false)
	assert.True(t, d.AllowFallback)
	assert.Equal(t, service.ClassProviderAuthFailure, d.Class)

	// 7. Client 400 Bad Request (terminal, no fallback)
	d = service.ClassifyRouteError(ctx, types.NewErrorWithStatusCode(errors.New("bad request"), "bad_request", 400), false)
	assert.False(t, d.AllowFallback)
	assert.Equal(t, service.ClassTerminalClient, d.Class)

	// 8. Client 404 Model Not Found (terminal, no fallback)
	d = service.ClassifyRouteError(ctx, types.NewErrorWithStatusCode(errors.New("model not found"), "model_not_found", 404), false)
	assert.False(t, d.AllowFallback)
	assert.Equal(t, service.ClassTerminalClient, d.Class)

	// 9. Local Auth 401 with SkipRetry option
	skipRetry401 := types.NewError(errors.New("invalid tora token"), types.ErrorCodeInvalidRequest, types.ErrOptionWithStatusCode(401), types.ErrOptionWithSkipRetry())
	d = service.ClassifyRouteError(ctx, skipRetry401, false)
	assert.False(t, d.AllowFallback)
	assert.Equal(t, service.ClassTerminalLocalAuth, d.Class)

	// 10. Local Quota 402 with SkipRetry option
	skipRetry402 := types.NewError(errors.New("quota depleted"), types.ErrorCodeInvalidRequest, types.ErrOptionWithStatusCode(402), types.ErrOptionWithSkipRetry())
	d = service.ClassifyRouteError(ctx, skipRetry402, false)
	assert.False(t, d.AllowFallback)
	assert.Equal(t, service.ClassTerminalQuota, d.Class)

	// 11. SAFE_PRE_EXECUTION_FAILURE: DNS resolution failure (Pre-execution, fallback allowed)
	dnsErr := types.NewError(errors.New("dial tcp: lookup api.openai.com: no such host"), types.ErrorCodeDoRequestFailed)
	d = service.ClassifyRouteError(ctx, dnsErr, false)
	assert.True(t, d.AllowFallback)
	assert.Equal(t, service.ClassSafePreExecutionFailure, d.Class)

	// 12. SAFE_PRE_EXECUTION_FAILURE: TCP connect refused (Pre-execution, fallback allowed)
	connRefused := types.NewError(errors.New("dial tcp 127.0.0.1:8080: connect: connection refused"), types.ErrorCodeDoRequestFailed)
	d = service.ClassifyRouteError(ctx, connRefused, false)
	assert.True(t, d.AllowFallback)
	assert.Equal(t, service.ClassSafePreExecutionFailure, d.Class)

	// 13. SAFE_PRE_EXECUTION_FAILURE: TLS handshake failure before request transmission
	tlsErr := types.NewError(errors.New("remote error: tls: handshake failure"), types.ErrorCodeDoRequestFailed)
	d = service.ClassifyRouteError(ctx, tlsErr, false)
	assert.True(t, d.AllowFallback)
	assert.Equal(t, service.ClassSafePreExecutionFailure, d.Class)

	// 14. AMBIGUOUS_EXECUTION_FAILURE: Read timeout while awaiting headers (Body transmitted; NO fallback!)
	awaitingHeadersErr := types.NewError(errors.New("net/http: Client.Timeout exceeded while awaiting headers"), types.ErrorCodeDoRequestFailed)
	d = service.ClassifyRouteError(ctx, awaitingHeadersErr, false)
	assert.False(t, d.AllowFallback, "ambiguous execution must NOT fallback automatically (Denial-of-Wallet defense)")
	assert.Equal(t, service.ClassAmbiguousExecutionFailure, d.Class)

	// 15. AMBIGUOUS_EXECUTION_FAILURE: Connection reset after transmission (Body transmitted; NO fallback!)
	connResetErr := types.NewError(errors.New("read: connection reset by peer"), types.ErrorCodeDoRequestFailed)
	d = service.ClassifyRouteError(ctx, connResetErr, false)
	assert.False(t, d.AllowFallback, "connection reset after request transmission must not fallback")
	assert.Equal(t, service.ClassAmbiguousExecutionFailure, d.Class)

	// 16. AMBIGUOUS_EXECUTION_FAILURE: EOF after request sent (Remote accepted and closed without response; NO fallback!)
	eofErr := types.NewError(errors.New("unexpected EOF"), types.ErrorCodeDoRequestFailed)
	d = service.ClassifyRouteError(ctx, eofErr, false)
	assert.False(t, d.AllowFallback, "unexpected EOF must be treated as ambiguous execution without fallback")
	assert.Equal(t, service.ClassAmbiguousExecutionFailure, d.Class)
}
