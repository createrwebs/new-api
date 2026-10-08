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
	dsn := fmt.Sprintf("file:test_native_%d?mode=memory&cache=shared", time.Now().UnixNano())
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
	assert.Equal(t, 2, quote.Credits, "Native bg-remove should cost 2 Tora Credits")
	assert.Equal(t, 2000, quote.Quota, "2 Credits must strictly equal 2,000 Quota units")
	assert.InDelta(t, 0.0040, quote.USDEquivalent, 0.0001)
	assert.Equal(t, "309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8", quote.Model.SHA256, "Verified u2netp SHA256 digest")
	assert.Equal(t, "Apache-2.0", quote.Model.License)

	upscaleQuote, err := GetNativeToolQuote("image-upscale-2x", model.ExecutionClassNativeBrowser)
	require.NoError(t, err)
	assert.Equal(t, 3, upscaleQuote.Credits)
	assert.Equal(t, 3000, upscaleQuote.Quota)
	assert.Equal(t, "c4c0b7430ebb554f3939a720f9e8445fdeca3d15edb69979c7ace39b997f3483", upscaleQuote.Model.SHA256)

	packQuote, err := GetNativeToolQuote("product-pack", model.ExecutionClassDeterministicProcess)
	require.NoError(t, err)
	assert.Equal(t, 5, packQuote.Credits)
	assert.Equal(t, 5000, packQuote.Quota)
	assert.Equal(t, model.ExecutionClassDeterministicProcess, packQuote.ExecutionClass)
}

// Test 2: Native Ticket Lifecycle (Reserve -> Complete -> Settle)
func TestStudioNative_TicketLifecycle_ReserveAndComplete(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "native_creator",
		Quota:    50000, // 50 Tora Credits ($0.10)
	}
	require.NoError(t, db.Create(&user).Error)

	inputs := map[string]interface{}{
		"tool":   "background-remove",
		"width":  512,
		"height": 512,
	}

	// 1. Issue Ticket (Wallet Pre-consume 2,000 Quota)
	ticket, err := CreateNativeExecutionTicket(user.Id, "background-remove", model.ExecutionClassNativeBrowser, inputs, "iphone_safari_webgpu")
	require.NoError(t, err)
	assert.NotEmpty(t, ticket.TicketId)
	assert.Equal(t, 2000, ticket.ReservedQuota)
	assert.Equal(t, 2, ticket.ReservedCredits)
	assert.Equal(t, model.TicketStatusReserved, ticket.Status)
	assert.Equal(t, "309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8", ticket.ModelVersionHash)

	// Verify User Quota dropped by 2,000
	userQuota, err := model.GetUserQuota(user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, 48000, userQuota, "User quota must be reserved during ticket issuance")

	// 2. Client Completes Inference (85ms latency)
	outputHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	completedTicket, err := CompleteNativeExecutionTicket(user.Id, ticket.TicketId, outputHash, 85, "iphone_safari_webgpu")
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusSettled, completedTicket.Status)
	assert.Equal(t, int64(85), completedTicket.ClientExecutionMs)
	assert.Equal(t, outputHash, completedTicket.OutputAssetHash)

	// User Quota remains 48,000 (spent permanently)
	userQuota, err = model.GetUserQuota(user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, 48000, userQuota)

	// 3. Idempotent Completion
	reCompleted, err := CompleteNativeExecutionTicket(user.Id, ticket.TicketId, outputHash, 85, "iphone_safari_webgpu")
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusSettled, reCompleted.Status)
}

// Test 3: Native Ticket Refund on Client Failure
func TestStudioNative_TicketLifecycle_Refund(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "refund_tester",
		Quota:    50000,
	}
	require.NoError(t, db.Create(&user).Error)

	inputs := map[string]interface{}{"upscale": 2}
	ticket, err := CreateNativeExecutionTicket(user.Id, "image-upscale-2x", model.ExecutionClassNativeBrowser, inputs, "android_chrome")
	require.NoError(t, err)
	assert.Equal(t, 3000, ticket.ReservedQuota)

	// Verify quota reserved
	q, _ := model.GetUserQuota(user.Id, false)
	assert.Equal(t, 47000, q)

	// Client encounters WebGPU out of memory and reports refund
	refunded, err := RefundNativeExecutionTicket(user.Id, ticket.TicketId, "WebGPU OOM shader allocation failed")
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusRefunded, refunded.Status)
	assert.Equal(t, "WebGPU OOM shader allocation failed", refunded.ErrorReason)

	// Quota is restored
	q, _ = model.GetUserQuota(user.Id, false)
	assert.Equal(t, 50000, q, "User quota must be refunded back to original balance")

	// Attempting to settle a refunded ticket must be blocked
	_, err = CompleteNativeExecutionTicket(user.Id, ticket.TicketId, "dummyhash", 100, "android_chrome")
	assert.ErrorIs(t, err, model.ErrNativeTicketAlreadyRefunded)
}

// Test 4: Expired Ticket Sweep & Reconciliation
func TestStudioNative_Reconciliation_ExpiredTickets(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "abandon_user",
		Quota:    20000,
	}
	require.NoError(t, db.Create(&user).Error)

	ticket, err := CreateNativeExecutionTicket(user.Id, "background-remove", model.ExecutionClassNativeBrowser, nil, "desktop_firefox")
	require.NoError(t, err)

	q, _ := model.GetUserQuota(user.Id, false)
	assert.Equal(t, 18000, q)

	// Simulate ticket expiry by backdating expires_at in database
	past := common.GetTimestamp() - 60
	require.NoError(t, db.Model(&model.NativeExecutionTicket{}).
		Where("ticket_id = ?", ticket.TicketId).
		Update("expires_at", past).Error)

	// Run reconciliation worker sweep
	reconciledCount, err := ReconcileExpiredNativeTickets(0)
	require.NoError(t, err)
	assert.Equal(t, 1, reconciledCount, "Exactly 1 expired ticket should be reconciled")

	// Verify ticket status transitioned to EXPIRED
	tkt, err := model.GetNativeTicket(ticket.TicketId)
	require.NoError(t, err)
	assert.Equal(t, model.TicketStatusExpired, tkt.Status)

	// Verify user balance restored
	q, _ = model.GetUserQuota(user.Id, false)
	assert.Equal(t, 20000, q, "Quota must be fully refunded upon ticket expiration sweep")
}

// Test 5: Deterministic Marketplace Product Pack Pipeline
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
