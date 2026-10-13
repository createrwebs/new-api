package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	common.LogWriterMu.Lock()
	origWriter := gin.DefaultWriter
	origErrWriter := gin.DefaultErrorWriter
	gin.DefaultWriter = buf
	gin.DefaultErrorWriter = buf
	common.LogWriterMu.Unlock()

	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultWriter = origWriter
		gin.DefaultErrorWriter = origErrWriter
		common.LogWriterMu.Unlock()
	})
	return buf
}

func enablePaymentComplianceForTest(t *testing.T) {
	t.Helper()
	ps := operation_setting.GetPaymentSetting()
	origConfirmed := ps.ComplianceConfirmed
	origVersion := ps.ComplianceTermsVersion

	ps.ComplianceConfirmed = true
	ps.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() {
		ps.ComplianceConfirmed = origConfirmed
		ps.ComplianceTermsVersion = origVersion
	})
}

func TestStripeWebhookLogSanitization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enablePaymentComplianceForTest(t)
	logBuf := captureLogs(t)

	origSecret := setting.StripeWebhookSecret
	origApiSecret := setting.StripeApiSecret
	origPriceId := setting.StripePriceId
	setting.StripeWebhookSecret = "whsec_test_secret"
	setting.StripeApiSecret = "sk_test_api"
	setting.StripePriceId = "price_test"
	defer func() {
		setting.StripeWebhookSecret = origSecret
		setting.StripeApiSecret = origApiSecret
		setting.StripePriceId = origPriceId
	}()

	const piiCanary = "STRIPE_PII_CANARY_Customer_Jane_Doe_jane@example.com"
	const sigCanary = "STRIPE_SIG_CANARY_t=12345,v1=abcdef1234567890"

	router := gin.New()
	router.POST("/api/stripe/webhook", StripeWebhook)

	body := []byte(`{"id":"evt_123","object":"event","data":{"object":{"customer_email":"` + piiCanary + `"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/stripe/webhook", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", sigCanary)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	logOutput := logBuf.String()
	assert.NotContains(t, logOutput, piiCanary, "Stripe raw webhook payload/PII must not appear in server logs")
	assert.NotContains(t, logOutput, sigCanary, "Stripe signature must not appear in server logs")
	assert.Contains(t, logOutput, "has_signature=true")
}

func TestCreemWebhookLogSanitization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enablePaymentComplianceForTest(t)
	logBuf := captureLogs(t)

	origSecret := setting.CreemWebhookSecret
	origApiKey := setting.CreemApiKey
	origProducts := setting.CreemProducts
	setting.CreemWebhookSecret = "whsec_creem_secret"
	setting.CreemApiKey = "creem_api_key"
	setting.CreemProducts = `[{"productId":"p1","price":10}]`
	defer func() {
		setting.CreemWebhookSecret = origSecret
		setting.CreemApiKey = origApiKey
		setting.CreemProducts = origProducts
	}()

	const piiCanary = "CREEM_PII_CANARY_buyer@example.com_Customer_Bob"
	const sigCanary = "CREEM_SIG_CANARY_creem_hmac_signature_999"

	router := gin.New()
	router.POST("/api/creem/webhook", CreemWebhook)

	body := []byte(`{"event_type":"checkout.completed","id":"evt_999","object":{"customer":{"email":"` + piiCanary + `"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/creem/webhook", bytes.NewReader(body))
	req.Header.Set(CreemSignatureHeader, sigCanary)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	logOutput := logBuf.String()
	assert.NotContains(t, logOutput, piiCanary, "Creem raw webhook payload/PII must not appear in server logs")
	assert.NotContains(t, logOutput, sigCanary, "Creem signature must not appear in server logs")
	assert.Contains(t, logOutput, "has_signature=true")
}

func TestWaffoWebhookLogSanitization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enablePaymentComplianceForTest(t)
	logBuf := captureLogs(t)

	origEnabled := setting.WaffoEnabled
	origApiKey := setting.WaffoApiKey
	origPrivKey := setting.WaffoPrivateKey
	origCert := setting.WaffoPublicCert
	origSandbox := setting.WaffoSandbox

	setting.WaffoSandbox = false
	setting.WaffoEnabled = true
	setting.WaffoApiKey = "mock_api_key"
	setting.WaffoPrivateKey = "mock_priv_key"
	setting.WaffoPublicCert = "mock_cert"
	defer func() {
		setting.WaffoEnabled = origEnabled
		setting.WaffoApiKey = origApiKey
		setting.WaffoPrivateKey = origPrivKey
		setting.WaffoPublicCert = origCert
		setting.WaffoSandbox = origSandbox
	}()

	const piiCanary = "WAFFO_PII_CANARY_order_cardholder_Alice_Smith"
	const sigCanary = "WAFFO_SIG_CANARY_waffo_rsa_signature_888"

	router := gin.New()
	router.POST("/api/waffo/webhook", WaffoWebhook)

	body := []byte(`{"eventType":"PAYMENT","result":{"merchantOrderId":"ORD_123","buyer":"` + piiCanary + `"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/waffo/webhook", bytes.NewReader(body))
	req.Header.Set("X-SIGNATURE", sigCanary)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	logOutput := logBuf.String()
	assert.NotContains(t, logOutput, piiCanary, "Waffo raw webhook payload/PII must not appear in server logs")
	assert.NotContains(t, logOutput, sigCanary, "Waffo signature must not appear in server logs")
}

func TestWaffoPancakeWebhookLogSanitization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enablePaymentComplianceForTest(t)
	logBuf := captureLogs(t)

	origMerchantID := setting.WaffoPancakeMerchantID
	origPrivateKey := setting.WaffoPancakePrivateKey
	origProductID := setting.WaffoPancakeProductID

	setting.WaffoPancakeMerchantID = "mock_merchant_id"
	setting.WaffoPancakePrivateKey = "mock_private_key"
	setting.WaffoPancakeProductID = "mock_product_id"
	defer func() {
		setting.WaffoPancakeMerchantID = origMerchantID
		setting.WaffoPancakePrivateKey = origPrivateKey
		setting.WaffoPancakeProductID = origProductID
	}()

	const piiCanary = "PANCAKE_PII_CANARY_buyer_charlie@example.com"
	const sigCanary = "PANCAKE_SIG_CANARY_hmac_sha256_777"

	router := gin.New()
	router.POST("/api/waffo-pancake/webhook/:env", WaffoPancakeWebhook)

	body := []byte(`{"id":"evt_777","data":{"orderId":"ORD_777","customer":"` + piiCanary + `"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/waffo-pancake/webhook/prod", bytes.NewReader(body))
	req.Header.Set("X-Waffo-Signature", sigCanary)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	logOutput := logBuf.String()
	assert.NotContains(t, logOutput, piiCanary, "Waffo Pancake raw webhook payload/PII must not appear in server logs")
	assert.NotContains(t, logOutput, sigCanary, "Waffo Pancake signature must not appear in server logs")
}
