package relay

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// PrepareRequestBilling estimates and reserves one request's charge. Transports
// provide the current request body through BodyStorage or BillingRequestInput;
// channel retries retain the resulting billing session and pricing snapshot.
func PrepareRequestBilling(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	if (c != nil && (c.GetBool("is_byok") || common.GetContextKeyBool(c, constant.ContextKeyIsBYOK))) || (info != nil && info.IsBYOK) {
		return nil
	}
	needSensitiveCheck := setting.ShouldCheckPromptSensitive()
	meta := &types.TokenCountMeta{TokenType: types.TokenTypeTokenizer}
	if info.Request != nil && (needSensitiveCheck || constant.CountToken) {
		meta = info.Request.GetTokenCountMeta()
	} else {
		// Avoid building CombineText when only the pricing quantities are needed.
		switch request := info.Request.(type) {
		case *dto.GeneralOpenAIRequest:
			meta.MaxTokens = int(max(lo.FromPtr(request.MaxTokens), lo.FromPtr(request.MaxCompletionTokens)))
		case *dto.OpenAIResponsesRequest:
			meta.MaxTokens = int(lo.FromPtr(request.MaxOutputTokens))
		case *dto.ClaudeRequest:
			meta.MaxTokens = int(lo.FromPtr(request.MaxTokens))
		case *dto.ImageRequest:
			meta = request.GetTokenCountMeta()
		}
	}

	if meta.MaxTokens == 0 && info.Request != nil {
		switch request := info.Request.(type) {
		case *dto.GeneralOpenAIRequest:
			meta.MaxTokens = int(max(lo.FromPtr(request.MaxTokens), lo.FromPtr(request.MaxCompletionTokens)))
		case *dto.OpenAIResponsesRequest:
			meta.MaxTokens = int(lo.FromPtr(request.MaxOutputTokens))
		case *dto.ClaudeRequest:
			meta.MaxTokens = int(lo.FromPtr(request.MaxTokens))
		case *dto.GeminiChatRequest:
			meta.MaxTokens = int(lo.FromPtr(request.GenerationConfig.MaxOutputTokens))
		}
	}

	if needSensitiveCheck && meta != nil {
		if contains, words := service.CheckSensitiveText(meta.CombineText); contains {
			service.RequestPolicy(c).AddEvent(service.PolicyEvent{ErrorCode: string(types.ErrorCodeSensitiveWordsDetected), ErrorSource: "local", Decision: service.PolicyDecision{Action: "stop", Reason: "local_rejection", Source: "global"}, Health: "unchanged"})
			message := fmt.Sprintf("user sensitive words detected: %s", strings.Join(words, ", "))
			logger.LogWarn(c, message)
			return types.NewError(errors.New(message), types.ErrorCodeSensitiveWordsDetected)
		}
	}

	tokens, err := service.EstimateRequestToken(c, meta, info)
	if err != nil {
		return types.NewError(err, types.ErrorCodeCountTokenFailed)
	}
	info.SetEstimatePromptTokens(tokens)

	priceData, err := helper.ModelPriceHelper(c, info, tokens, meta)
	if err != nil {
		return types.NewError(err, types.ErrorCodeModelPriceError, types.ErrOptionWithStatusCode(http.StatusBadRequest))
	}
	if priceData.FreeModel {
		logger.LogInfo(c, fmt.Sprintf("模型 %s 免费，跳过预扣费", info.OriginModelName))
		return nil
	}
	quotaToReserve := priceData.QuotaToPreConsume
	if c != nil {
		if rawChain, exists := c.Get("token_route_chain"); exists {
			if chain, ok := rawChain.([]int); ok && len(chain) > 0 {
				userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
				isBYOK := c.GetBool("is_byok") || common.GetContextKeyBool(c, constant.ContextKeyIsBYOK)
				maxMultiplier := 1.0
				var snapshots []*service.RouteSnapshot
				for _, rId := range chain {
					r, err := service.CacheGetRoute(rId)
					if err != nil || r == nil || !r.Enabled {
						continue
					}
					// Invariant: Entitlement validation
					if err := service.ValidateRouteEntitlement(r, userGroup); err != nil {
						continue
					}
					// Invariant: BYOK isolation
					if isBYOK && r.Kind != "byok" {
						continue
					}
					// Invariant: Model coverage check (Section 9 Reservation Eligibility)
					if !service.RouteSupportsModel(r, info.OriginModelName, userGroup) {
						continue
					}
					snapshots = append(snapshots, service.SnapshotRoute(r))
					if r.CostMultiplier > maxMultiplier {
						maxMultiplier = r.CostMultiplier
					}
				}
				if len(snapshots) > 0 {
					c.Set("token_route_snapshots", snapshots)
				}
				if maxMultiplier > 1.0 {
					quotaToReserve = int(math.Ceil(float64(quotaToReserve) * maxMultiplier))
				}
				c.Set("route_chain_max_multiplier", maxMultiplier)
			}
		}
	}
	return service.PreConsumeBilling(c, quotaToReserve, info)
}

// RefundFailedRequestBilling applies the common final-failure policy after all
// eligible attempts have ended. A settled BillingSession never refunds again.
func RefundFailedRequestBilling(c *gin.Context, info *relaycommon.RelayInfo, apiErr *types.NewAPIError) *types.NewAPIError {
	if apiErr == nil {
		return nil
	}
	apiErr = service.NormalizeViolationFeeError(apiErr)
	if info.Billing != nil {
		info.Billing.Refund(c)
	}
	service.ChargeViolationFeeIfNeeded(c, info, apiErr)
	return apiErr
}
