package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var (
	ErrNativeToolNotSupported = errors.New("unsupported native tool or execution class")
	ErrNativeUnauthorized     = errors.New("unauthorized access to native ticket")
	ErrPreExecutionRefundOnly = errors.New("automatic refund is permitted only for pre-activation tickets; charged browser executions use fair retry")
)

// NativeModelMeta details an open-weight model verified and cached for native inference.
type NativeModelMeta struct {
	ModelId      string `json:"model_id"`
	ModelName    string `json:"model_name"`
	Filename     string `json:"filename"`
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"size_bytes"`
	Format       string `json:"format"`
	License      string `json:"license"`
	ExecutionEnv string `json:"execution_env"`
	InputShape   string `json:"input_shape"`
	DownloadURL  string `json:"download_url"`
}

// NativeModels holds the authoritative verified digests and licenses of harvested models.
var NativeModels = map[string]NativeModelMeta{
	"u2netp": {
		ModelId:      "u2netp",
		ModelName:    "U2Net-P Fast Mobile Cutout",
		Filename:     "u2netp.onnx",
		SHA256:       "309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8",
		SizeBytes:    4572242,
		Format:       "onnx",
		License:      "Apache-2.0",
		ExecutionEnv: "NATIVE_BROWSER / NATIVE_MOBILE",
		InputShape:   "1x3x320x320",
		DownloadURL:  "/api/studio/native/models/u2netp",
	},
	"modnet": {
		ModelId:      "modnet",
		ModelName:    "MODNet Portrait Matting",
		Filename:     "modnet_photographic_portrait_matting.onnx",
		SHA256:       "07c308cf0fc7e6e8b2065a12ed7fc07e1de8febb7dc7839d7b7f15dd66584df9",
		SizeBytes:    25890000,
		Format:       "onnx",
		License:      "Apache-2.0",
		ExecutionEnv: "NATIVE_BROWSER / NATIVE_MOBILE",
		InputShape:   "1x3x512x512",
		DownloadURL:  "/api/studio/native/models/modnet",
	},
	"realesrgan_2x": {
		ModelId:      "realesrgan_2x",
		ModelName:    "Real-ESRGAN x2plus",
		Filename:     "2x-realesrgan-x2plus.onnx",
		SHA256:       "c4c0b7430ebb554f3939a720f9e8445fdeca3d15edb69979c7ace39b997f3483",
		SizeBytes:    67192664,
		Format:       "onnx",
		License:      "BSD-3-Clause",
		ExecutionEnv: "NATIVE_BROWSER / NATIVE_MOBILE",
		InputShape:   "dynamic (tiled)",
		DownloadURL:  "/api/studio/native/models/realesrgan_2x",
	},
	"realesrgan_4x": {
		ModelId:      "realesrgan_4x",
		ModelName:    "Real-ESRGAN x4plus",
		Filename:     "RealESRGAN_x4plus.onnx",
		SHA256:       "cd0ec097469c94c903e6f74d4f43f545683250ec0a54bc0c2ab1ff4c6364d8da",
		SizeBytes:    67174378,
		Format:       "onnx",
		License:      "BSD-3-Clause",
		ExecutionEnv: "NATIVE_BROWSER",
		InputShape:   "dynamic (tiled)",
		DownloadURL:  "/api/studio/native/models/realesrgan_4x",
	},
	"isnet": {
		ModelId:      "isnet",
		ModelName:    "ISNet General Use Cutout",
		Filename:     "isnet-general-use.onnx",
		SHA256:       "60920e99c45464f2ba57bee2ad08c919a52bbf852739e96947fbb4358c0d964a",
		SizeBytes:    178643896,
		Format:       "onnx",
		License:      "MIT",
		ExecutionEnv: "NATIVE_LOCAL_CPU",
		InputShape:   "1x3x1024x1024",
		DownloadURL:  "/api/studio/native/models/isnet",
	},
}

// NativeToolSpec defines billing, model bindings, and execution routing for native tools.
type NativeToolSpec struct {
	ToolId         string                     `json:"tool_id"`
	DefaultClass   model.NativeExecutionClass `json:"default_class"`
	BillingPolicy  model.NativeBillingPolicy  `json:"billing_policy"`
	ModelKey       string                     `json:"model_key"`
	Credits        int                        `json:"credits"`
	Quota          int                        `json:"quota"`
	USDEquivalent  float64                    `json:"usd_equivalent"`
	RouteVersion   string                     `json:"route_version"`
}

var NativeToolCatalog = map[string]NativeToolSpec{
	"background-remove": {
		ToolId:        "background-remove",
		DefaultClass:  model.ExecutionClassNativeBrowser,
		BillingPolicy: model.BillingPolicyPrepaidExecution,
		ModelKey:      "u2netp",
		Credits:       2,
		Quota:         2000,
		USDEquivalent: 0.004,
		RouteVersion:  "v1_native_u2netp",
	},
	"portrait-matting": {
		ToolId:        "portrait-matting",
		DefaultClass:  model.ExecutionClassNativeBrowser,
		BillingPolicy: model.BillingPolicyPrepaidExecution,
		ModelKey:      "modnet",
		Credits:       2,
		Quota:         2000,
		USDEquivalent: 0.004,
		RouteVersion:  "v1_native_modnet",
	},
	"image-upscale-2x": {
		ToolId:        "image-upscale-2x",
		DefaultClass:  model.ExecutionClassNativeBrowser,
		BillingPolicy: model.BillingPolicyPrepaidExecution,
		ModelKey:      "realesrgan_2x",
		Credits:       3,
		Quota:         3000,
		USDEquivalent: 0.006,
		RouteVersion:  "v1_native_realesrgan2x",
	},
	"image-upscale": {
		ToolId:        "image-upscale",
		DefaultClass:  model.ExecutionClassNativeBrowser,
		BillingPolicy: model.BillingPolicyPrepaidExecution,
		ModelKey:      "realesrgan_4x",
		Credits:       5,
		Quota:         5000,
		USDEquivalent: 0.010,
		RouteVersion:  "v1_native_realesrgan4x",
	},
	"product-pack": {
		ToolId:        "product-pack",
		DefaultClass:  model.ExecutionClassDeterministicServer,
		BillingPolicy: model.BillingPolicySuccessSettlement,
		ModelKey:      "u2netp",
		Credits:       5,
		Quota:         5000,
		USDEquivalent: 0.010,
		RouteVersion:  "v1_deterministic_pack",
	},
}

// NativeQuoteResult contains quote details for a client execution request.
type NativeQuoteResult struct {
	QuoteId          string                     `json:"quote_id"`
	ToolId           string                     `json:"tool_id"`
	ExecutionClass   model.NativeExecutionClass `json:"execution_class"`
	BillingPolicy    model.NativeBillingPolicy  `json:"billing_policy"`
	Credits          int                        `json:"credits"`
	Quota            int                        `json:"quota"`
	USDEquivalent    float64                    `json:"usd_equivalent"`
	Model            NativeModelMeta            `json:"model"`
	RouteVersion     string                     `json:"route_version"`
	ExpiresAt        int64                      `json:"expires_at"`
	FallbackProvider string                     `json:"fallback_provider,omitempty"`
}

// getTicketSigningKey returns a domain-separated cryptographic secret for ticket authorization.
func getTicketSigningKey() []byte {
	if secret := os.Getenv("STUDIO_TICKET_SECRET"); secret != "" {
		mac := hmac.New(sha256.New, []byte("tora-studio-native-ticket-v2-salt"))
		mac.Write([]byte(secret))
		return mac.Sum(nil)
	}
	baseSecret := common.SessionSecret
	if baseSecret == "" {
		baseSecret = common.CryptoSecret
	}
	mac := hmac.New(sha256.New, []byte("tora-studio-native-ticket-v2"))
	mac.Write([]byte(baseSecret))
	return mac.Sum(nil)
}

// GenerateTicketAuthToken creates a tamper-resistant HMAC signature for client execution authorization.
func GenerateTicketAuthToken(ticket *model.NativeExecutionTicket) string {
	key := getTicketSigningKey()
	payload := fmt.Sprintf("%s|%d|%s|%s|%s|%s|%s|%s|%s|%s|%d|%d|%d|%d|%s",
		ticket.TicketId,
		ticket.UserId,
		ticket.ToolId,
		ticket.ToolVersion,
		ticket.ExecutionClass,
		ticket.BillingPolicy,
		ticket.ModelId,
		ticket.ModelVersionHash,
		ticket.NormalizedInputHash,
		ticket.QuoteId,
		ticket.ChargedQuota,
		ticket.IssuedAt,
		ticket.ExpiresAt,
		ticket.RetryUntil,
		ticket.Nonce,
	)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyTicketAuthToken validates that a client-presented authorization token was signed by the server.
func VerifyTicketAuthToken(ticket *model.NativeExecutionTicket, token string) bool {
	expected := GenerateTicketAuthToken(ticket)
	return hmac.Equal([]byte(expected), []byte(token))
}

func generateRandomNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ResolveNativeToolSpec finds the tool specification or matching alias.
func ResolveNativeToolSpec(toolId string) (*NativeToolSpec, bool) {
	if spec, ok := NativeToolCatalog[toolId]; ok {
		return &spec, true
	}
	switch toolId {
	case "bg-remove", "bg-remove-native", "background-remove-native":
		spec := NativeToolCatalog["background-remove"]
		return &spec, true
	case "upscale", "upscale-2x", "upscale-2x-native":
		spec := NativeToolCatalog["image-upscale-2x"]
		return &spec, true
	case "upscale-4x", "upscale-4x-native":
		spec := NativeToolCatalog["image-upscale"]
		return &spec, true
	case "product-photo-pack", "product_pack", "product-pack-native":
		spec := NativeToolCatalog["product-pack"]
		return &spec, true
	}
	return nil, false
}

// GetNativeToolQuote provides server-authoritative pricing for native execution.
func GetNativeToolQuote(toolId string, reqClass model.NativeExecutionClass) (*NativeQuoteResult, error) {
	spec, found := ResolveNativeToolSpec(toolId)
	if !found {
		return nil, ErrNativeToolNotSupported
	}

	execClass := reqClass
	if execClass == "" {
		execClass = spec.DefaultClass
	}

	billingPolicy := spec.BillingPolicy
	if execClass == model.ExecutionClassNativeBrowser || execClass == model.ExecutionClassNativeMobile {
		billingPolicy = model.BillingPolicyPrepaidExecution
	} else if execClass == model.ExecutionClassDeterministicServer || execClass == model.ExecutionClassNativeServer {
		billingPolicy = model.BillingPolicySuccessSettlement
	}

	modelMeta, ok := NativeModels[spec.ModelKey]
	if !ok {
		modelMeta = NativeModels["u2netp"]
	}

	now := common.GetTimestamp()
	quoteId := fmt.Sprintf("qte_native_%d_%s", now, common.GetUUID()[:8])

	return &NativeQuoteResult{
		QuoteId:          quoteId,
		ToolId:           spec.ToolId,
		ExecutionClass:   execClass,
		BillingPolicy:    billingPolicy,
		Credits:          spec.Credits,
		Quota:            spec.Quota,
		USDEquivalent:    spec.USDEquivalent,
		Model:            modelMeta,
		RouteVersion:     spec.RouteVersion,
		ExpiresAt:        now + 300, // 5 min TTL
		FallbackProvider: "wavespeed",
	}, nil
}

// CreateNativeExecutionTicket authorizes and provisions an execution ticket.
// For NATIVE_BROWSER / NATIVE_MOBILE (PREPAID_EXECUTION), wallet quota is atomically reserved and charged BEFORE authorization.
// For DETERMINISTIC_SERVER (SUCCESS_SETTLEMENT), quota is reserved in escrow and settled upon success.
func CreateNativeExecutionTicket(userId int, toolId string, reqClass model.NativeExecutionClass, inputs map[string]interface{}, clientDeviceClass string, idempotencyKey string) (*model.NativeExecutionTicket, error) {
	// 1. Idempotency Check (Section 12)
	if idempotencyKey != "" {
		if existing, err := model.GetNativeTicketByIdempotencyKey(userId, idempotencyKey); err == nil && existing != nil {
			return existing, nil
		}
	}

	spec, found := ResolveNativeToolSpec(toolId)
	if !found {
		return nil, ErrNativeToolNotSupported
	}

	execClass := reqClass
	if execClass == "" {
		execClass = spec.DefaultClass
	}

	billingPolicy := spec.BillingPolicy
	if execClass == model.ExecutionClassNativeBrowser || execClass == model.ExecutionClassNativeMobile {
		billingPolicy = model.BillingPolicyPrepaidExecution
	}

	modelMeta, ok := NativeModels[spec.ModelKey]
	if !ok {
		modelMeta = NativeModels["u2netp"]
	}

	now := common.GetTimestamp()
	ticketId := fmt.Sprintf("tkt_%d_%s", now, common.GetUUID()[:8])
	requestId := fmt.Sprintf("req_native_%s", ticketId)

	// 2. Authoritative Wallet Operation
	if err := model.PreConsumeUserWallet(requestId, userId, spec.Quota); err != nil {
		return nil, fmt.Errorf("wallet quota reservation failed: %w", err)
	}

	status := model.TicketStatusReserved
	chargedQuota := 0
	chargedCredits := 0
	var chargedAt int64 = 0

	// 3. For NATIVE_BROWSER / NATIVE_MOBILE: Commit the charge immediately (Section 4 PREPAID_EXECUTION)
	if billingPolicy == model.BillingPolicyPrepaidExecution {
		if err := model.SettleUserWalletPreConsume(requestId); err != nil {
			_ = model.RefundUserWalletPreConsume(requestId)
			return nil, fmt.Errorf("failed committing prepaid wallet charge: %w", err)
		}
		status = model.TicketStatusCharged
		chargedQuota = spec.Quota
		chargedCredits = spec.Credits
		chargedAt = now
	}

	inputHash := model.ComputeNormalizedInputHash(inputs)
	nonce := generateRandomNonce()

	retryUntil := now + 300 // 5-minute default
	if execClass == model.ExecutionClassNativeBrowser || execClass == model.ExecutionClassNativeMobile {
		retryUntil = now + 1800 // 30-minute fair retry window (Section 7)
	}

	ticket := &model.NativeExecutionTicket{
		Id:                  ticketId,
		TicketId:            ticketId,
		UserId:              userId,
		ToolId:              spec.ToolId,
		ToolVersion:         "v1.0.0",
		RouteVersion:        spec.RouteVersion,
		ExecutionClass:      execClass,
		BillingPolicy:       billingPolicy,
		ModelId:             modelMeta.ModelId,
		ModelVersion:        "1.0.0",
		ModelVersionHash:    modelMeta.SHA256,
		QuoteId:             fmt.Sprintf("qte_%s", ticketId),
		RequestId:           requestId,
		IdempotencyKey:      idempotencyKey,
		ReservedQuota:       spec.Quota,
		ReservedCredits:     spec.Credits,
		ChargedQuota:        chargedQuota,
		ChargedCredits:      chargedCredits,
		NormalizedInputHash: inputHash,
		Status:              status,
		Nonce:               nonce,
		IssuedAt:            now,
		ExpiresAt:           now + 300,
		ChargedAt:           chargedAt,
		SettledAt:           chargedAt,
		RetryUntil:          retryUntil,
		RetryCount:          0,
		ClientDeviceClass:   clientDeviceClass,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	ticket.AuthToken = GenerateTicketAuthToken(ticket)

	if err := model.CreateNativeTicket(ticket); err != nil {
		if billingPolicy == model.BillingPolicyPrepaidExecution {
			_ = model.RefundUserWalletPreConsume(requestId)
		} else {
			_ = model.RefundUserWalletPreConsume(requestId)
		}
		return nil, fmt.Errorf("failed creating native ticket record: %w", err)
	}

	return ticket, nil
}

// CompleteNativeExecutionTicket marks client-side completion or settles server-side execution.
// For NATIVE_BROWSER (PREPAID_EXECUTION): Records telemetry and hashes; billing was already committed at activation.
// For DETERMINISTIC_SERVER (SUCCESS_SETTLEMENT): Settles escrow quota upon successful generation.
func CompleteNativeExecutionTicket(userId int, ticketId string, outputAssetHash string, clientExecutionMs int64, clientDeviceClass string) (*model.NativeExecutionTicket, error) {
	ticket, err := model.GetNativeTicket(ticketId)
	if err != nil {
		return nil, err
	}

	if ticket.UserId != userId {
		return nil, ErrNativeUnauthorized
	}

	// Idempotency: already completed or settled
	if ticket.Status == model.TicketStatusCompleted {
		return ticket, nil
	}
	if ticket.Status == model.TicketStatusSupportRefunded {
		return nil, model.ErrNativeTicketAlreadyRefunded
	}

	now := common.GetTimestamp()

	if ticket.BillingPolicy == model.BillingPolicyPrepaidExecution {
		// NATIVE_BROWSER flow: Ticket was charged upfront. Settle is already finalized.
		updates := map[string]interface{}{
			"completed_at":        now,
			"output_asset_hash":   outputAssetHash,
			"client_execution_ms": clientExecutionMs,
		}
		if clientDeviceClass != "" {
			updates["client_device_class"] = clientDeviceClass
		}
		if err := model.UpdateNativeTicketStatus(ticket.TicketId, model.TicketStatusCompleted, updates); err != nil {
			return nil, fmt.Errorf("failed recording completion: %w", err)
		}
		ticket.Status = model.TicketStatusCompleted
		ticket.CompletedAt = now
		ticket.OutputAssetHash = outputAssetHash
		ticket.ClientExecutionMs = clientExecutionMs
		return ticket, nil
	}

	// SUCCESS_SETTLEMENT flow (DETERMINISTIC_SERVER):
	if err := model.SettleUserWalletPreConsume(ticket.RequestId); err != nil {
		return nil, fmt.Errorf("failed settling wallet reservation: %w", err)
	}

	updates := map[string]interface{}{
		"completed_at":        now,
		"charged_at":          now,
		"settled_at":          now,
		"charged_quota":       ticket.ReservedQuota,
		"charged_credits":     ticket.ReservedCredits,
		"output_asset_hash":   outputAssetHash,
		"client_execution_ms": clientExecutionMs,
	}
	if clientDeviceClass != "" {
		updates["client_device_class"] = clientDeviceClass
	}
	if err := model.UpdateNativeTicketStatus(ticket.TicketId, model.TicketStatusCompleted, updates); err != nil {
		return nil, fmt.Errorf("failed updating settled ticket state: %w", err)
	}

	ticket.Status = model.TicketStatusCompleted
	ticket.CompletedAt = now
	ticket.ChargedAt = now
	ticket.SettledAt = now
	ticket.ChargedQuota = ticket.ReservedQuota
	ticket.ChargedCredits = ticket.ReservedCredits
	ticket.OutputAssetHash = outputAssetHash
	ticket.ClientExecutionMs = clientExecutionMs

	return ticket, nil
}

// FailNativeExecutionTicket records client-side runtime failure without refunding charged browser tickets (Section 7).
func FailNativeExecutionTicket(userId int, ticketId string, reason string) (*model.NativeExecutionTicket, error) {
	ticket, err := model.GetNativeTicket(ticketId)
	if err != nil {
		return nil, err
	}

	if ticket.UserId != userId {
		return nil, ErrNativeUnauthorized
	}

	// Update ticket to FAILED_CLIENT
	updates := map[string]interface{}{
		"error_reason": reason,
	}
	if err := model.UpdateNativeTicketStatus(ticket.TicketId, model.TicketStatusFailedClient, updates); err != nil {
		return nil, err
	}

	ticket.Status = model.TicketStatusFailedClient
	ticket.ErrorReason = reason
	return ticket, nil
}

// RetryNativeExecutionTicket allows fair same-ticket retry within the 30-minute window with ZERO additional credits (Sections 7 & 8).
func RetryNativeExecutionTicket(userId int, ticketId string) (*model.NativeExecutionTicket, error) {
	ticket, err := model.GetNativeTicket(ticketId)
	if err != nil {
		return nil, err
	}

	if ticket.UserId != userId {
		return nil, ErrNativeUnauthorized
	}

	if ticket.ExecutionClass != model.ExecutionClassNativeBrowser && ticket.ExecutionClass != model.ExecutionClassNativeMobile {
		return nil, errors.New("same-ticket fair retry applies to NATIVE_BROWSER and NATIVE_MOBILE executions")
	}

	now := common.GetTimestamp()
	if now > ticket.RetryUntil {
		return nil, model.ErrNativeTicketRetryExceeded
	}

	// Refresh nonce and authorization token
	ticket.RetryCount++
	ticket.Nonce = generateRandomNonce()
	ticket.AuthToken = GenerateTicketAuthToken(ticket)

	updates := map[string]interface{}{
		"status":      model.TicketStatusCharged,
		"retry_count": ticket.RetryCount,
		"nonce":       ticket.Nonce,
		"auth_token":  ticket.AuthToken,
	}

	if err := model.UpdateNativeTicketStatus(ticket.TicketId, model.TicketStatusCharged, updates); err != nil {
		return nil, fmt.Errorf("failed updating retry state: %w", err)
	}

	ticket.Status = model.TicketStatusCharged
	return ticket, nil
}

// RefundPreExecutionTicket refunds quota ONLY if failure occurred before execution authorization (Section 6).
func RefundPreExecutionTicket(userId int, ticketId string, reason string) (*model.NativeExecutionTicket, error) {
	ticket, err := model.GetNativeTicket(ticketId)
	if err != nil {
		return nil, err
	}

	if ticket.UserId != userId {
		return nil, ErrNativeUnauthorized
	}

	// Security Gate: If ticket was already CHARGED, automatic refund is prohibited
	if ticket.Status == model.TicketStatusCharged || ticket.Status == model.TicketStatusStarted || ticket.Status == model.TicketStatusCompleted {
		return nil, ErrPreExecutionRefundOnly
	}

	now := common.GetTimestamp()
	if err := model.RefundUserWalletPreConsume(ticket.RequestId); err != nil {
		return nil, fmt.Errorf("failed refunding wallet reservation: %w", err)
	}

	updates := map[string]interface{}{
		"refunded_at":  now,
		"error_reason": reason,
	}
	if err := model.UpdateNativeTicketStatus(ticket.TicketId, model.TicketStatusSupportRefunded, updates); err != nil {
		return nil, fmt.Errorf("failed updating refunded ticket state: %w", err)
	}

	ticket.Status = model.TicketStatusSupportRefunded
	ticket.RefundedAt = now
	ticket.ErrorReason = reason

	return ticket, nil
}

// SupportRefundTicket enables manual administrator/support refund for genuine verified system failures.
func SupportRefundTicket(userId int, ticketId string, reason string) (*model.NativeExecutionTicket, error) {
	ticket, err := model.GetNativeTicket(ticketId)
	if err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	// Compensate user quota
	if ticket.ChargedQuota > 0 {
		_ = model.IncreaseUserQuota(ticket.UserId, ticket.ChargedQuota, false)
	} else if ticket.ReservedQuota > 0 && ticket.Status == model.TicketStatusReserved {
		_ = model.RefundUserWalletPreConsume(ticket.RequestId)
	}

	updates := map[string]interface{}{
		"refunded_at":  now,
		"error_reason": "support_refund: " + reason,
	}
	_ = model.UpdateNativeTicketStatus(ticket.TicketId, model.TicketStatusSupportRefunded, updates)
	ticket.Status = model.TicketStatusSupportRefunded
	ticket.RefundedAt = now
	ticket.ErrorReason = "support_refund: " + reason

	return ticket, nil
}

// ReconcileExpiredNativeTickets sweeps abandoned tickets past TTL.
// CRITICAL SECURITY ENFORCEMENT:
// - RESERVED tickets (pre-activation): Safe to refund.
// - CHARGED / FAILED_CLIENT / STARTED tickets: Transition to EXPIRED past RetryUntil, DO NOT AUTO-REFUND WALLET (Sections 5 & 71).
func ReconcileExpiredNativeTickets(olderThanSeconds int64) (int, error) {
	if model.DB == nil {
		return 0, errors.New("database not initialized")
	}

	now := common.GetTimestamp()
	cutoff := now
	if olderThanSeconds > 0 {
		cutoff = now - olderThanSeconds
	}

	reconciled := 0

	// 1. Clean abandoned pre-activation reservations (Safe to refund)
	var reservedTickets []model.NativeExecutionTicket
	err := model.DB.Where("status = ? AND expires_at <= ?", string(model.TicketStatusReserved), cutoff).
		Limit(200).Find(&reservedTickets).Error
	if err == nil {
		for _, t := range reservedTickets {
			_ = model.RefundUserWalletPreConsume(t.RequestId)
			_ = model.UpdateNativeTicketStatus(t.TicketId, model.TicketStatusExpired, map[string]interface{}{
				"refunded_at":  now,
				"error_reason": "pre-activation reservation expired",
			})
			reconciled++
		}
	}

	// 2. Clean charged browser tickets whose fair retry window has fully elapsed (DO NOT REFUND)
	var chargedTickets []model.NativeExecutionTicket
	err = model.DB.Where("status IN (?) AND retry_until <= ?", []string{
		string(model.TicketStatusCharged),
		string(model.TicketStatusStarted),
		string(model.TicketStatusFailedClient),
	}, cutoff).Limit(200).Find(&chargedTickets).Error
	if err == nil {
		for _, t := range chargedTickets {
			_ = model.UpdateNativeTicketStatus(t.TicketId, model.TicketStatusExpired, map[string]interface{}{
				"error_reason": "fair retry window closed without completion",
			})
			reconciled++
		}
	}

	return reconciled, nil
}

var nativeReconcileOnce sync.Once

// StartNativeTicketReconciliationWorker runs a background cleaner for abandoned client executions.
func StartNativeTicketReconciliationWorker() {
	nativeReconcileOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(2 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				count, err := ReconcileExpiredNativeTickets(0)
				if err != nil {
					common.SysError(fmt.Sprintf("[NativeTicketWorker] error reconciling expired tickets: %v", err))
				} else if count > 0 {
					common.SysLog(fmt.Sprintf("[NativeTicketWorker] swept %d expired native tickets (billing secured)", count))
				}
			}
		}()
	})
}
