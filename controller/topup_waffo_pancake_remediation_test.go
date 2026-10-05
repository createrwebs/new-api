package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestWaffoPancakeCatalogIgnoresQueryCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Clear persisted creds
	origMerchant := setting.WaffoPancakeMerchantID
	origKey := setting.WaffoPancakePrivateKey
	setting.WaffoPancakeMerchantID = ""
	setting.WaffoPancakePrivateKey = ""
	defer func() {
		setting.WaffoPancakeMerchantID = origMerchant
		setting.WaffoPancakePrivateKey = origKey
	}()

	router := gin.New()
	router.GET("/api/option/waffo-pancake/catalog", ListWaffoPancakeCatalog)

	const leakedKeyCanary = "WAFFO_QUERY_CANARY_RSA_KEY_12345"

	// Calling GET with query credentials must NOT use them (strictly rejected/ignored)
	req := httptest.NewRequest(http.MethodGet, "/api/option/waffo-pancake/catalog?merchant_id=merchant123&private_key="+leakedKeyCanary, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "error", resp["message"])
	assert.Equal(t, "Waffo Pancake 凭证未配置", resp["data"], "GET must not accept credentials from query string")
}

func TestWaffoPancakeCatalogAcceptsPostBodyCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Clear persisted creds
	origMerchant := setting.WaffoPancakeMerchantID
	origKey := setting.WaffoPancakePrivateKey
	setting.WaffoPancakeMerchantID = ""
	setting.WaffoPancakePrivateKey = ""
	defer func() {
		setting.WaffoPancakeMerchantID = origMerchant
		setting.WaffoPancakePrivateKey = origKey
	}()

	router := gin.New()
	router.POST("/api/option/waffo-pancake/catalog", ListWaffoPancakeCatalog)

	// Send POST with credentials in JSON body
	bodyBytes, _ := json.Marshal(map[string]string{
		"merchant_id": "test_merchant",
		"private_key": "test_private_key",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/option/waffo-pancake/catalog", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	// Because service.ListWaffoPancakeCatalog attempts mock network request or returns failure,
	// the data will not be "Waffo Pancake 凭证未配置", proving credentials were read from body!
	assert.NotEqual(t, "Waffo Pancake 凭证未配置", resp["data"], "POST body credentials must be recognized")
}
