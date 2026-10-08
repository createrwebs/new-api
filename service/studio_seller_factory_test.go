package service

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// Helper to create a synthetic transparent cutout image (bottle shape).
func createTestCutoutPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Draw a central opaque bottle rectangle
	boxMinX, boxMaxX := w/4, 3*w/4
	boxMinY, boxMaxY := h/4, 3*h/4
	for y := boxMinY; y < boxMaxY; y++ {
		for x := boxMinX; x < boxMaxX; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 50, G: 100, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestSellerFactory_SmartShadows_AllPresets(t *testing.T) {
	canvasW, canvasH := 500, 500
	subjRect := image.Rect(100, 100, 400, 400)

	presets := []SmartShadowPreset{
		ShadowSoftStudio,
		ShadowMarketplace,
		ShadowFloating,
		ShadowGroundContact,
		ShadowNone,
	}

	for _, p := range presets {
		canvas1 := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
		canvas2 := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
		white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		for y := 0; y < canvasH; y++ {
			for x := 0; x < canvasW; x++ {
				canvas1.SetRGBA(x, y, white)
				canvas2.SetRGBA(x, y, white)
			}
		}

		RenderSmartShadow(canvas1, subjRect, p)
		RenderSmartShadow(canvas2, subjRect, p)

		// 100% Determinism check: pixel for pixel identical
		h1 := sha256.Sum256(canvas1.Pix)
		h2 := sha256.Sum256(canvas2.Pix)
		if h1 != h2 {
			t.Fatalf("Shadow preset %s is not deterministic: %x != %x", p, h1, h2)
		}

		if p == ShadowNone {
			// Pixels should remain pure white
			for i := 0; i < len(canvas1.Pix); i += 4 {
				if canvas1.Pix[i] != 255 || canvas1.Pix[i+1] != 255 || canvas1.Pix[i+2] != 255 {
					t.Fatalf("ShadowNone modified canvas pixels")
				}
			}
		} else {
			// At least some pixels beneath subject should be darkened
			darkened := false
			for y := subjRect.Max.Y; y < canvasH; y++ {
				for x := subjRect.Min.X; x < subjRect.Max.X; x++ {
					col := canvas1.RGBAAt(x, y)
					if col.R < 250 {
						darkened = true
						break
					}
				}
				if darkened {
					break
				}
			}
			if !darkened {
				t.Fatalf("Shadow preset %s did not darken pixels beneath subject base", p)
			}
		}
	}
}

func TestSellerFactory_SellerBackgrounds(t *testing.T) {
	bgPresets := []SellerBgPreset{
		BgPureWhite,
		BgWarmWhite,
		BgLightGray,
		BgBrandColor,
		BgSoftGradient,
		BgStudioVignette,
		BgTransparent,
	}

	for _, bg := range bgPresets {
		canvas := image.NewRGBA(image.Rect(0, 0, 200, 200))
		RenderSellerBackground(canvas, bg, "#EE4D2D")

		switch bg {
		case BgTransparent:
			if canvas.RGBAAt(100, 100).A != 0 {
				t.Fatalf("BgTransparent expected alpha 0, got %d", canvas.RGBAAt(100, 100).A)
			}
		case BgPureWhite:
			col := canvas.RGBAAt(100, 100)
			if col.R != 255 || col.G != 255 || col.B != 255 {
				t.Fatalf("BgPureWhite expected 255,255,255, got %v", col)
			}
		case BgBrandColor:
			col := canvas.RGBAAt(100, 100)
			// #EE4D2D -> R:238, G:77, B:45
			if col.R != 238 || col.G != 77 || col.B != 45 {
				t.Fatalf("BgBrandColor expected 238,77,45, got %v", col)
			}
		}
	}
}

func TestSellerFactory_DeterministicTeleaInpaint(t *testing.T) {
	W, H := 100, 100
	src := image.NewRGBA(image.Rect(0, 0, W, H))
	mask := image.NewGray(image.Rect(0, 0, W, H))

	// Solid blue background
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			src.SetRGBA(x, y, color.RGBA{R: 50, G: 100, B: 200, A: 255})
		}
	}

	// White scratch in center (to be inpainted)
	for x := 45; x <= 55; x++ {
		src.SetRGBA(x, 50, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		mask.SetGray(x, 50, color.Gray{Y: 255})
	}

	res := DeterministicTeleaInpaint(src, mask, 3)

	// Center pixel (50, 50) was white (255, 255, 255); inpaint should fill it close to background (50, 100, 200)
	filledCol := res.RGBAAt(50, 50)
	if filledCol.R > 100 || filledCol.B < 150 {
		t.Fatalf("Telea inpaint failed to reconstruct background: got %v", filledCol)
	}
}

func TestSellerFactory_ObjectCleanup_InteractiveSessionBilling(t *testing.T) {
	db := setupNativeTestDB(t)

	user := model.User{
		Username: "cleanup_user",
		Quota:    50000,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed creating test user: %v", err)
	}
	userId := user.Id

	sourceHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// 1. Start interactive session
	session, ticket, err := StartObjectCleanupSession(userId, sourceHash, "DESKTOP_MAC")
	if err != nil {
		t.Fatalf("StartObjectCleanupSession failed: %v", err)
	}
	if session.Status != "ACTIVE" {
		t.Fatalf("expected session status ACTIVE, got %s", session.Status)
	}
	if ticket.ChargedCredits != 3 || ticket.ChargedQuota != 3000 {
		t.Fatalf("expected 3 credits charged for session, got %d credits, %d quota", ticket.ChargedCredits, ticket.ChargedQuota)
	}

	// 2. Validate session against matching source image
	validated, err := ValidateObjectCleanupSession(userId, session.SessionId, sourceHash)
	if err != nil {
		t.Fatalf("ValidateObjectCleanupSession failed: %v", err)
	}
	if validated.SessionId != session.SessionId {
		t.Fatalf("mismatched session ID")
	}

	// 3. Reject session transfer to different source image
	otherHash := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	_, err = ValidateObjectCleanupSession(userId, session.SessionId, otherHash)
	if err == nil {
		t.Fatalf("expected error when validating against mismatched source image, got nil")
	}

	// 4. Record mask edits within session without extra charge
	for i := 0; i < 3; i++ {
		if err := RecordObjectCleanupMaskEdit(userId, session.SessionId); err != nil {
			t.Fatalf("RecordObjectCleanupMaskEdit failed on edit %d: %v", i+1, err)
		}
	}
	updated, _ := model.GetObjectCleanupSessionRecord(session.SessionId)
	if updated.MaskEditsCount != 3 {
		t.Fatalf("expected 3 mask edits recorded, got %d", updated.MaskEditsCount)
	}

	// 5. Record export within session
	if err := RecordObjectCleanupExport(userId, session.SessionId); err != nil {
		t.Fatalf("RecordObjectCleanupExport failed: %v", err)
	}
	updated, _ = model.GetObjectCleanupSessionRecord(session.SessionId)
	if updated.ExportCount != 1 {
		t.Fatalf("expected 1 export recorded, got %d", updated.ExportCount)
	}
}

func TestSellerFactory_V2BatchExecution(t *testing.T) {
	userId := 101
	cutoutBytes := createTestCutoutPNG(400, 400)

	// Test 3-item batch
	req := ProductFactoryV2Request{
		BatchId: "test_batch_3",
		Items: []ProductFactoryV2ItemRequest{
			{Index: 0, RawCutoutBytes: cutoutBytes, OriginalName: "shoe_black.png"},
			{Index: 1, RawCutoutBytes: cutoutBytes, OriginalName: "shoe_red.png"},
			{Index: 2, RawCutoutBytes: cutoutBytes, OriginalName: "shoe_blue.png"},
		},
		SelectedTemplates: []string{"shopee_standard", "lazada_hd", "tiktok_shop"},
		BgPreset:          BgPureWhite,
		ShadowPreset:      ShadowMarketplace,
		IncludeZip:        true,
	}

	batchRes, err := GenerateSellerFactoryV2Batch(userId, req)
	if err != nil {
		t.Fatalf("GenerateSellerFactoryV2Batch failed: %v", err)
	}

	if batchRes.TotalItems != 3 {
		t.Fatalf("expected 3 total items, got %d", batchRes.TotalItems)
	}
	if batchRes.SuccessItems != 3 {
		t.Fatalf("expected 3 successful items, got %d (failed: %d)", batchRes.SuccessItems, batchRes.FailedItems)
	}

	// Check each item has 3 variants
	for _, it := range batchRes.Items {
		if len(it.Variants) != 3 {
			t.Fatalf("expected 3 variants per item, got %d", len(it.Variants))
		}
	}

	// Check ZIP package was created
	if batchRes.ZipPackage.FileSize == 0 {
		t.Fatalf("expected non-empty ZIP archive, got size 0")
	}
}

func TestSellerFactory_V2BatchExecution_PartialFailureIsolation(t *testing.T) {
	userId := 102
	validBytes := createTestCutoutPNG(200, 200)

	// Item 1 is corrupt/invalid bytes, Item 0 and 2 are valid
	req := ProductFactoryV2Request{
		BatchId: "test_batch_partial",
		Items: []ProductFactoryV2ItemRequest{
			{Index: 0, RawCutoutBytes: validBytes},
			{Index: 1, RawCutoutBytes: []byte("corrupt_not_an_image")},
			{Index: 2, RawCutoutBytes: validBytes},
		},
		SelectedTemplates: []string{"shopee_standard"},
		BgPreset:          BgPureWhite,
		IncludeZip:        false,
	}

	batchRes, err := GenerateSellerFactoryV2Batch(userId, req)
	if err != nil {
		t.Fatalf("GenerateSellerFactoryV2Batch failed: %v", err)
	}

	if batchRes.SuccessItems != 2 || batchRes.FailedItems != 1 {
		t.Fatalf("expected 2 successes and 1 failure, got %d successes, %d failures", batchRes.SuccessItems, batchRes.FailedItems)
	}
	if batchRes.Items[1].Status != "FAILED" {
		t.Fatalf("expected item 1 status FAILED, got %s", batchRes.Items[1].Status)
	}
	if batchRes.Items[0].Status != "SUCCESS" || batchRes.Items[2].Status != "SUCCESS" {
		t.Fatalf("expected items 0 and 2 to succeed despite item 1 failure")
	}
}

func TestSellerFactory_LoadBenchmark_Scenarios(t *testing.T) {
	userId := 105
	cutout := createTestCutoutPNG(400, 400)
	templates := []string{
		"shopee_standard",
		"lazada_hd",
		"tiktok_shop",
		"instagram_feed",
		"instagram_story",
		"generic_marketplace",
	}

	scenarios := []struct {
		name      string
		itemCount int
	}{
		{"1_Item_Batch", 1},
		{"5_Item_Batch", 5},
		{"10_Item_Batch", 10},
	}

	for _, sc := range scenarios {
		var memBefore runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memBefore)

		items := make([]ProductFactoryV2ItemRequest, sc.itemCount)
		for i := 0; i < sc.itemCount; i++ {
			items[i] = ProductFactoryV2ItemRequest{
				Index:          i,
				RawCutoutBytes: cutout,
			}
		}

		req := ProductFactoryV2Request{
			BatchId:           "load_bench_" + sc.name,
			Items:             items,
			SelectedTemplates: templates,
			BgPreset:          BgPureWhite,
			ShadowPreset:      ShadowMarketplace,
			IncludeZip:        true,
		}

		start := time.Now()
		res, err := GenerateSellerFactoryV2Batch(userId, req)
		duration := time.Since(start)

		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		if err != nil {
			t.Fatalf("scenario %s failed: %v", sc.name, err)
		}
		if res.SuccessItems != sc.itemCount {
			t.Fatalf("scenario %s: expected %d successes, got %d", sc.name, sc.itemCount, res.SuccessItems)
		}

		heapAllocMB := float64(memAfter.Alloc-memBefore.Alloc) / (1024 * 1024)
		if heapAllocMB < 0 {
			heapAllocMB = float64(memAfter.Alloc) / (1024 * 1024)
		}

		t.Logf("BENCHMARK [%s]: items=%d | wall_time=%v | per_item=%v | zip_size=%d bytes | heap_delta=%.2f MB",
			sc.name, sc.itemCount, duration, duration/time.Duration(sc.itemCount), res.ZipPackage.FileSize, heapAllocMB)
	}
}

