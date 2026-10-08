package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupNativeTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:test_native_v2_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.WalletPreConsumeRecord{},
	))
	require.NoError(t, model.EnsureStudioTables(db))

	return db
}

// Test 1: Native Quotes & Pricing Specs
func TestStudioNative_QuotesAndCatalogSpecs(t *testing.T) {
	quote, err := GetNativeToolQuote("background-remove", model.ExecutionClassNativeBrowser)
	require.NoError(t, err)
	assert.Equal(t, "background-remove", quote.ToolId)
	assert.Equal(t, model.ExecutionClassNativeBrowser, quote.ExecutionClass)
	assert.Equal(t, model.BillingPolicyPrepaidExecution, quote.BillingPolicy)
	assert.Equal(t, 2, quote.Credits, "Native bg-remove should cost 2 Tora Credits")
	assert.Equal(t, 2000, quote.Quota, "2 Credits must strictly equal 2,000 Quota units")
	assert.InDelta(t, 0.0040, quote.USDEquivalent, 0.0001)
	assert.Equal(t, "309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8", quote.Model.SHA256, "Verified u2netp SHA256 digest")
	assert.Equal(t, "Apache-2.0", quote.Model.License)

	upscaleQuote, err := GetNativeToolQuote("image-upscale-2x", model.ExecutionClassNativeBrowser)
	require.NoError(t, err)
	assert.Equal(t, 3, upscaleQuote.Credits)
	assert.Equal(t, 3000, upscaleQuote.Quota)
	assert.Equal(t, model.BillingPolicyPrepaidExecution, upscaleQuote.BillingPolicy)
	assert.Equal(t, "c4c0b7430ebb554f3939a720f9e8445fdeca3d15edb69979c7ace39b997f3483", upscaleQuote.Model.SHA256)

	packQuote, err := GetNativeToolQuote("product-pack", model.ExecutionClassDeterministicServer)
	require.NoError(t, err)
	assert.Equal(t, 5, packQuote.Credits)
	assert.Equal(t, 5000, packQuote.Quota)
	assert.Equal(t, model.ExecutionClassDeterministicServer, packQuote.ExecutionClass)
	assert.Equal(t, model.BillingPolicySuccessSettlement, packQuote.BillingPolicy)
}

// Test 2: Prepaid Execution - Charge Committed at Activation BEFORE Client Runs Local Inference (Sections 4 & 5)
func TestStudioNative_PrepaidExecution_ReserveAndSettleAtActivation(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "native_prepaid_user",
		Quota:    50000, // 50 Tora Credits ($0.10)
	}
	require.NoError(t, db.Create(&user).Error)

	inputs := map[string]interface{}{
		"tool":   "background-remove",
		"width":  512,
		"height": 512,
	}

	// 1. Issue & Activate Ticket -> Quota Charged Atomically at Activation
	ticket, err := CreateNativeExecutionTicket(user.Id, "background-remove", model.ExecutionClassNativeBrowser, inputs, "mac_safari_webgpu", "")
	require.NoError(t, err)
	assert.NotEmpty(t, ticket.TicketId)
	assert.Equal(t, model.TicketStatusCharged, ticket.Status, "NATIVE_BROWSER ticket must be in CHARGED status upon issuance")
	assert.Equal(t, 2000, ticket.ChargedQuota)
	assert.Equal(t, 2, ticket.ChargedCredits)
	assert.NotEmpty(t, ticket.AuthToken, "Server must issue a signed cryptographic authorization token")
	assert.True(t, VerifyTicketAuthToken(ticket, ticket.AuthToken), "Token signature must verify against server secret")

	// Quota is ALREADY deducted from user's wallet
	userQuota, err := model.GetUserQuota(user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, 48000, userQuota, "User quota must be charged permanently at ticket activation")

	// 2. Client runs inference locally and calls /complete -> Records telemetry, quota does NOT change
	outputHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	completedTicket, err := CompleteNativeExecutionTicket(user.Id, ticket.TicketId, outputHash, 85, "mac_safari_webgpu")
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusCompleted, completedTicket.Status)
	assert.Equal(t, int64(85), completedTicket.ClientExecutionMs)
	assert.Equal(t, outputHash, completedTicket.OutputAssetHash)

	// User Quota remains exactly 48,000 (no double-deduction)
	userQuota, err = model.GetUserQuota(user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, 48000, userQuota)
}

// Test 3: Idempotency - Same Key Yields Existing Ticket With Zero Double-Charge (Section 12)
func TestStudioNative_Idempotency_NoDoubleCharge(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "idempotency_tester",
		Quota:    50000,
	}
	require.NoError(t, db.Create(&user).Error)

	inputs := map[string]interface{}{"mode": "portrait"}
	idempotencyKey := "idem_key_unique_998877"

	// Call 1: Provisions ticket
	ticket1, err := CreateNativeExecutionTicket(user.Id, "portrait-matting", model.ExecutionClassNativeBrowser, inputs, "chrome_desktop", idempotencyKey)
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusCharged, ticket1.Status)

	// Balance drops from 50,000 to 48,000 (2 credits)
	q1, _ := model.GetUserQuota(user.Id, false)
	assert.Equal(t, 48000, q1)

	// Call 2: Repeated call with SAME idempotency key
	ticket2, err := CreateNativeExecutionTicket(user.Id, "portrait-matting", model.ExecutionClassNativeBrowser, inputs, "chrome_desktop", idempotencyKey)
	require.NoError(t, err)
	assert.Equal(t, ticket1.TicketId, ticket2.TicketId, "Idempotent call must return the exact same ticket")

	// Balance MUST NOT drop again
	q2, _ := model.GetUserQuota(user.Id, false)
	assert.Equal(t, 48000, q2, "Balance must not be double-charged on idempotent replay")
}

// Test 4: Abuse Test - Old Exploit Fixed (Section 2 & 71)
// Attack: User activates ticket, obtains local cutout, blocks /complete, waits past TTL.
// Verification: Wallet remains charged, server NEVER auto-refunds charged browser tickets!
func TestStudioNative_AbuseTest_OldExploitFixed(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "attacker_account",
		Quota:    30000, // 30 Credits
	}
	require.NoError(t, db.Create(&user).Error)

	inputs := map[string]interface{}{"exploit_attempt": true}
	ticket, err := CreateNativeExecutionTicket(user.Id, "background-remove", model.ExecutionClassNativeBrowser, inputs, "headless_chromium", "")
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusCharged, ticket.Status)

	// User wallet charged 2 Credits (28,000 left)
	q, _ := model.GetUserQuota(user.Id, false)
	assert.Equal(t, 28000, q)

	// Attacker intentionally suppresses /complete and waits past TTL (retry window also elapses)
	past := common.GetTimestamp() - 3600
	require.NoError(t, db.Model(&model.NativeExecutionTicket{}).
		Where("ticket_id = ?", ticket.TicketId).
		Updates(map[string]interface{}{
			"expires_at":  past,
			"retry_until": past,
		}).Error)

	// Run background reconciliation worker sweep
	sweptCount, err := ReconcileExpiredNativeTickets(0)
	require.NoError(t, err)
	assert.Equal(t, 1, sweptCount, "Ticket should be marked expired")

	// Verify ticket status transitioned to EXPIRED
	tkt, err := model.GetNativeTicket(ticket.TicketId)
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusExpired, tkt.Status)

	// CRITICAL TEST ASSERTION: Wallet was NOT refunded! Attacker keeps 28,000 quota, NOT 30,000!
	finalQuota, _ := model.GetUserQuota(user.Id, false)
	assert.Equal(t, 28000, finalQuota, "SECURITY FIX: Charged browser ticket must NEVER auto-refund to prevent free-use exploit")
}

// Test 5: Pre-Execution Failure Safely Refunds Escrow Quota (Section 6 & 72)
func TestStudioNative_PreActivationFailure_RefundsQuota(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "preflight_fail_user",
		Quota:    20000,
	}
	require.NoError(t, db.Create(&user).Error)

	// Simulate ticket in pre-activation RESERVED state
	now := common.GetTimestamp()
	ticket := &model.NativeExecutionTicket{
		Id:                  "tkt_preflight_1",
		TicketId:            "tkt_preflight_1",
		UserId:              user.Id,
		ToolId:              "background-remove",
		ExecutionClass:      model.ExecutionClassNativeBrowser,
		BillingPolicy:       model.BillingPolicyPrepaidExecution,
		RequestId:           "req_preflight_1",
		ReservedQuota:       2000,
		ReservedCredits:     2,
		NormalizedInputHash: "hash123",
		Status:              model.TicketStatusReserved,
		IssuedAt:            now,
		ExpiresAt:           now + 300,
	}
	// Reserve in wallet
	require.NoError(t, model.PreConsumeUserWallet(ticket.RequestId, user.Id, ticket.ReservedQuota))
	require.NoError(t, model.CreateNativeTicket(ticket))

	// Balance dropped to 18,000 during pre-flight reservation
	q, _ := model.GetUserQuota(user.Id, false)
	assert.Equal(t, 18000, q)

	// Compatibility preflight discovers device lacks WebGPU/WASM before activation
	refundedTicket, err := RefundPreExecutionTicket(user.Id, ticket.TicketId, "device lacks required WebGPU/WASM shader capabilities")
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusSupportRefunded, refundedTicket.Status)

	// Balance is safely restored to 20,000
	q, _ = model.GetUserQuota(user.Id, false)
	assert.Equal(t, 20000, q, "Pre-activation failures must safely refund reserved wallet quota")
}

// Test 6: Fair Same-Ticket Retry Policy with Zero Additional Credits (Sections 7 & 8 & 73)
func TestStudioNative_FairRetry_ZeroCredits(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "retry_beneficiary",
		Quota:    50000,
	}
	require.NoError(t, db.Create(&user).Error)

	// Issue ticket
	ticket, err := CreateNativeExecutionTicket(user.Id, "image-upscale-2x", model.ExecutionClassNativeBrowser, nil, "android_firefox", "")
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusCharged, ticket.Status)

	// User charged 3 Credits (47,000 balance remaining)
	q1, _ := model.GetUserQuota(user.Id, false)
	assert.Equal(t, 47000, q1)

	// Browser tab crashes mid-inference -> client reports failure
	failedTicket, err := FailNativeExecutionTicket(user.Id, ticket.TicketId, "WebGL context lost during tile 3")
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusFailedClient, failedTicket.Status)

	// User retries execution using the same ticket
	retriedTicket, err := RetryNativeExecutionTicket(user.Id, ticket.TicketId)
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusCharged, retriedTicket.Status)
	assert.Equal(t, 1, retriedTicket.RetryCount)
	assert.NotEmpty(t, retriedTicket.AuthToken)
	assert.True(t, VerifyTicketAuthToken(retriedTicket, retriedTicket.AuthToken))

	// CRITICAL ASSERTION: Zero additional wallet charge!
	q2, _ := model.GetUserQuota(user.Id, false)
	assert.Equal(t, 47000, q2, "Fair retry must cost exactly 0 additional Tora Credits")
}

// Test 7: Tamper-Resistant Cryptographic Signing (Section 11)
func TestStudioNative_HMACSignature_TamperResistance(t *testing.T) {
	now := common.GetTimestamp()
	ticket := &model.NativeExecutionTicket{
		TicketId:            "tkt_test_sign_123",
		UserId:              101,
		ToolId:              "background-remove",
		ToolVersion:         "v1.0.0",
		ExecutionClass:      model.ExecutionClassNativeBrowser,
		BillingPolicy:       model.BillingPolicyPrepaidExecution,
		ModelId:             "u2netp",
		ModelVersionHash:    "309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8",
		NormalizedInputHash: "input_digest_abc",
		QuoteId:             "qte_123",
		ChargedQuota:        2000,
		IssuedAt:            now,
		ExpiresAt:           now + 300,
		RetryUntil:          now + 1800,
		Nonce:               "random_nonce_99",
	}

	validToken := GenerateTicketAuthToken(ticket)
	assert.NotEmpty(t, validToken)
	assert.True(t, VerifyTicketAuthToken(ticket, validToken))

	// Tampering tests: verify altering any single signed field fails verification
	tamperCases := []struct {
		name   string
		mutate func(tkt *model.NativeExecutionTicket)
	}{
		{"tampered tool", func(tkt *model.NativeExecutionTicket) { tkt.ToolId = "image-upscale-4x" }},
		{"tampered quota (zero free)", func(tkt *model.NativeExecutionTicket) { tkt.ChargedQuota = 0 }},
		{"tampered user ID", func(tkt *model.NativeExecutionTicket) { tkt.UserId = 999 }},
		{"tampered ticket ID", func(tkt *model.NativeExecutionTicket) { tkt.TicketId = "tkt_other_id" }},
		{"tampered model hash", func(tkt *model.NativeExecutionTicket) { tkt.ModelVersionHash = "tampered_hash_abc" }},
		{"tampered input hash", func(tkt *model.NativeExecutionTicket) { tkt.NormalizedInputHash = "tampered_input" }},
		{"tampered execution class", func(tkt *model.NativeExecutionTicket) { tkt.ExecutionClass = model.ExecutionClassDeterministicServer }},
		{"tampered billing policy", func(tkt *model.NativeExecutionTicket) { tkt.BillingPolicy = model.BillingPolicySuccessSettlement }},
		{"tampered expiry", func(tkt *model.NativeExecutionTicket) { tkt.ExpiresAt = now + 99999 }},
		{"tampered retry window", func(tkt *model.NativeExecutionTicket) { tkt.RetryUntil = now + 99999 }},
		{"tampered nonce", func(tkt *model.NativeExecutionTicket) { tkt.Nonce = "nonce_spoof" }},
	}

	for _, tc := range tamperCases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := *ticket
			tc.mutate(&mutated)
			assert.False(t, VerifyTicketAuthToken(&mutated, validToken), "Tampering %s must fail signature verification", tc.name)
		})
	}
}

// Test 8: Deterministic Marketplace Product Pack Pipeline (Section 48)
func TestStudioProductPack_DeterministicPipeline(t *testing.T) {
	db := setupNativeTestDB(t)

	tmpDir := t.TempDir()
	os.Setenv("STUDIO_UPLOAD_DIR", tmpDir)
	defer os.Unsetenv("STUDIO_UPLOAD_DIR")

	user := model.User{
		Username: "seller_user",
		Quota:    50000,
	}
	require.NoError(t, db.Create(&user).Error)

	// Create synthetic transparent 200x200 PNG image with a 80x80 circle product in the center
	srcImg := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 60; y < 140; y++ {
		for x := 60; x < 140; x++ {
			dx := float64(x - 100)
			dy := float64(y - 100)
			if dx*dx+dy*dy <= 40*40 {
				srcImg.SetRGBA(x, y, color.RGBA{R: 220, G: 50, B: 50, A: 255})
			}
		}
	}

	buf := new(bytes.Buffer)
	require.NoError(t, png.Encode(buf, srcImg))
	rawBytes := buf.Bytes()

	// Run deterministic marketplace pipeline
	packResult, err := GenerateMarketplaceProductPack(db, user.Id, "job_test_pack_1", rawBytes)
	require.NoError(t, err)
	require.NotNil(t, packResult)

	// Verify Subject Bounding Box Detection
	bbox := packResult.SubjectBox
	assert.InDelta(t, 80, bbox.Dx(), 4, "Detected subject width should be ~80px")
	assert.InDelta(t, 80, bbox.Dy(), 4, "Detected subject height should be ~80px")

	// Verify Variants Generated
	require.Len(t, packResult.Variants, 4, "Should generate Shopee, Lazada, Instagram, and Story presets")

	shopeeFound := false
	lazadaFound := false
	igFound := false
	storyFound := false

	for _, v := range packResult.Variants {
		switch v.Key {
		case "shopee":
			shopeeFound = true
			assert.Equal(t, 800, v.Width)
			assert.Equal(t, 800, v.Height)
		case "lazada":
			lazadaFound = true
			assert.Equal(t, 1000, v.Width)
			assert.Equal(t, 1000, v.Height)
		case "instagram":
			igFound = true
			assert.Equal(t, 1080, v.Width)
			assert.Equal(t, 1350, v.Height)
		case "story":
			storyFound = true
			assert.Equal(t, 1080, v.Width)
			assert.Equal(t, 1920, v.Height)
		}
		assert.NotEmpty(t, v.URL)
		assert.NotEmpty(t, v.SHA256)
		assert.Greater(t, v.FileSize, int64(1000))
	}

	assert.True(t, shopeeFound, "Shopee preset missing")
	assert.True(t, lazadaFound, "Lazada preset missing")
	assert.True(t, igFound, "Instagram preset missing")
	assert.True(t, storyFound, "Story preset missing")

	// Verify All-In-One ZIP package
	assert.NotEmpty(t, packResult.ZipPackage.URL)
	assert.Greater(t, packResult.ZipPackage.FileSize, int64(2000))

	// Inspect ZIP contents from disk
	zipPath := filepath.Join(tmpDir, filepath.Base(packResult.ZipPackage.URL))
	zipReader, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer zipReader.Close()

	var filenames []string
	var manifestFound bool
	for _, f := range zipReader.File {
		filenames = append(filenames, f.Name)
		if f.Name == "manifest.json" {
			manifestFound = true
			rc, err := f.Open()
			require.NoError(t, err)
			manifestBytes, _ := io.ReadAll(rc)
			rc.Close()

			var m map[string]interface{}
			require.NoError(t, json.Unmarshal(manifestBytes, &m))
			assert.NotEmpty(t, m["pack_id"])
			assert.NotEmpty(t, m["compliance"])
		}
	}

	assert.True(t, manifestFound, "manifest.json must be bundled in ZIP package")
	assert.Contains(t, filenames, "01_shopee_800x800.jpg")
	assert.Contains(t, filenames, "02_lazada_1000x1000.jpg")
	assert.Contains(t, filenames, "03_instagram_1080x1350.jpg")
	assert.Contains(t, filenames, "04_story_tiktok_1080x1920.jpg")
	assert.Contains(t, filenames, "05_product_cutout_transparent.png")
}

// Test 9: Native Mobile Execution Class & Prepaid LifeCycle (Queue N3 Section 3)
func TestStudioNative_NativeMobile_QuoteAndPrepaidExecution(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "mobile_user_ios_android",
		Quota:    50000, // 50 Tora Credits
	}
	require.NoError(t, db.Create(&user).Error)

	// 1. Quote for NATIVE_MOBILE
	quote, err := GetNativeToolQuote("background-remove", model.ExecutionClassNativeMobile)
	require.NoError(t, err)
	assert.Equal(t, "background-remove", quote.ToolId)
	assert.Equal(t, model.ExecutionClassNativeMobile, quote.ExecutionClass)
	assert.Equal(t, model.BillingPolicyPrepaidExecution, quote.BillingPolicy, "NATIVE_MOBILE must be PREPAID_EXECUTION")
	assert.Equal(t, 2, quote.Credits)
	assert.Equal(t, 2000, quote.Quota)

	// 2. Ticket Creation for NATIVE_MOBILE on Android/iOS
	inputs := map[string]interface{}{
		"tool":   "background-remove",
		"width":  1080,
		"height": 1080,
	}
	ticket, err := CreateNativeExecutionTicket(user.Id, "background-remove", model.ExecutionClassNativeMobile, inputs, "android_nnapi_arm64", "idem_mobile_123")
	require.NoError(t, err)
	assert.Equal(t, model.ExecutionClassNativeMobile, ticket.ExecutionClass)
	assert.Equal(t, model.BillingPolicyPrepaidExecution, ticket.BillingPolicy)
	assert.Equal(t, model.TicketStatusCharged, ticket.Status, "NATIVE_MOBILE must be charged at activation")
	assert.Equal(t, 2000, ticket.ChargedQuota)
	assert.Equal(t, "android_nnapi_arm64", ticket.ClientDeviceClass)
	assert.True(t, VerifyTicketAuthToken(ticket, ticket.AuthToken))

	// User wallet should be deducted
	remQuota, err := model.GetUserQuota(user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, 48000, remQuota)

	// 3. Fair Retry for NATIVE_MOBILE (zero credits)
	retryTicket, err := RetryNativeExecutionTicket(user.Id, ticket.TicketId)
	require.NoError(t, err)
	assert.Equal(t, 1, retryTicket.RetryCount)
	assert.Equal(t, model.TicketStatusCharged, retryTicket.Status)

	remQuotaAfterRetry, err := model.GetUserQuota(user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, 48000, remQuotaAfterRetry, "Retry on NATIVE_MOBILE must cost 0 additional credits")

	// 4. Complete NATIVE_MOBILE inference
	completed, err := CompleteNativeExecutionTicket(user.Id, ticket.TicketId, "sha256_output_mobile_mask", 450, "android_nnapi_arm64")
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusCompleted, completed.Status)
	assert.Equal(t, "sha256_output_mobile_mask", completed.OutputAssetHash)
	assert.Equal(t, int64(450), completed.ClientExecutionMs)
}

