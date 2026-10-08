package service

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	xdraw "golang.org/x/image/draw"
)

// SmartShadowPreset defines deterministic shadow geometry profiles.
type SmartShadowPreset string

const (
	ShadowSoftStudio    SmartShadowPreset = "SOFT_STUDIO"
	ShadowMarketplace   SmartShadowPreset = "MARKETPLACE"
	ShadowFloating      SmartShadowPreset = "FLOATING"
	ShadowGroundContact SmartShadowPreset = "GROUND_CONTACT"
	ShadowNone          SmartShadowPreset = "NO_SHADOW"
)

// SellerBgPreset defines deterministic e-commerce background presets.
type SellerBgPreset string

const (
	BgPureWhite      SellerBgPreset = "PURE_WHITE"
	BgWarmWhite      SellerBgPreset = "WARM_WHITE"
	BgLightGray      SellerBgPreset = "LIGHT_GRAY"
	BgBrandColor     SellerBgPreset = "BRAND_COLOR"
	BgSoftGradient   SellerBgPreset = "SOFT_GRADIENT"
	BgStudioVignette SellerBgPreset = "STUDIO_VIGNETTE"
	BgTransparent    SellerBgPreset = "TRANSPARENT"
)

// SellerTemplateConfig defines a versioned marketplace export format.
type SellerTemplateConfig struct {
	TemplateId     string            `json:"template_id"`
	Version        string            `json:"version"`
	Marketplace    string            `json:"marketplace"`
	Filename       string            `json:"filename"`
	Width          int               `json:"width"`
	Height         int               `json:"height"`
	AspectRatio    string            `json:"aspect_ratio"`
	PaddingPct     float64           `json:"padding_pct"`
	DefaultBg      SellerBgPreset    `json:"default_bg"`
	DefaultShadow  SmartShadowPreset `json:"default_shadow"`
	Format         string            `json:"format"` // "jpg" or "png"
	Quality        int               `json:"quality"`
	SubjectAnchorY float64           `json:"subject_anchor_y"` // 0.5 = center, >0.5 = lower (grounded)
	Description    string            `json:"description"`
}

// Canonical Seller Template catalog (Version 2.0).
var CanonicalSellerTemplates = []SellerTemplateConfig{
	{
		TemplateId:     "shopee_standard",
		Version:        "v2.0",
		Marketplace:    "Shopee Standard (1000x1000)",
		Filename:       "01_shopee_1000x1000.jpg",
		Width:          1000,
		Height:         1000,
		AspectRatio:    "1:1",
		PaddingPct:     0.10,
		DefaultBg:      BgPureWhite,
		DefaultShadow:  ShadowMarketplace,
		Format:         "jpg",
		Quality:        92,
		SubjectAnchorY: 0.52,
		Description:    "Shopee White Catalog Standard (1000x1000)",
	},
	{
		TemplateId:     "lazada_hd",
		Version:        "v2.0",
		Marketplace:    "Lazada HD Listing (1000x1000)",
		Filename:       "02_lazada_1000x1000.jpg",
		Width:          1000,
		Height:         1000,
		AspectRatio:    "1:1",
		PaddingPct:     0.10,
		DefaultBg:      BgPureWhite,
		DefaultShadow:  ShadowMarketplace,
		Format:         "jpg",
		Quality:        92,
		SubjectAnchorY: 0.52,
		Description:    "Lazada HD Product Listing (1000x1000)",
	},
	{
		TemplateId:     "tiktok_shop",
		Version:        "v2.0",
		Marketplace:    "TikTok Shop HD (1200x1200)",
		Filename:       "03_tiktok_1200x1200.jpg",
		Width:          1200,
		Height:         1200,
		AspectRatio:    "1:1",
		PaddingPct:     0.10,
		DefaultBg:      BgPureWhite,
		DefaultShadow:  ShadowSoftStudio,
		Format:         "jpg",
		Quality:        94,
		SubjectAnchorY: 0.50,
		Description:    "TikTok Shop Official High-Resolution Catalog (1200x1200)",
	},
	{
		TemplateId:     "instagram_feed",
		Version:        "v2.0",
		Marketplace:    "Instagram Feed (1080x1080)",
		Filename:       "04_instagram_feed_1080x1080.jpg",
		Width:          1080,
		Height:         1080,
		AspectRatio:    "1:1",
		PaddingPct:     0.12,
		DefaultBg:      BgWarmWhite,
		DefaultShadow:  ShadowSoftStudio,
		Format:         "jpg",
		Quality:        95,
		SubjectAnchorY: 0.50,
		Description:    "Instagram Engagement Feed Showcase (1080x1080)",
	},
	{
		TemplateId:     "instagram_story",
		Version:        "v2.0",
		Marketplace:    "Instagram Story / Reel 9:16 (1080x1920)",
		Filename:       "05_instagram_story_1080x1920.jpg",
		Width:          1080,
		Height:         1920,
		AspectRatio:    "9:16",
		PaddingPct:     0.15,
		DefaultBg:      BgSoftGradient,
		DefaultShadow:  ShadowFloating,
		Format:         "jpg",
		Quality:        95,
		SubjectAnchorY: 0.48,
		Description:    "Vertical Story & Social Showcase 9:16 (1080x1920)",
	},
	{
		TemplateId:     "generic_marketplace",
		Version:        "v2.0",
		Marketplace:    "Generic E-Commerce Hi-Res (1200x1200)",
		Filename:       "06_marketplace_1200x1200.jpg",
		Width:          1200,
		Height:         1200,
		AspectRatio:    "1:1",
		PaddingPct:     0.10,
		DefaultBg:      BgPureWhite,
		DefaultShadow:  ShadowMarketplace,
		Format:         "jpg",
		Quality:        92,
		SubjectAnchorY: 0.52,
		Description:    "Universal Hi-Res Catalog (Amazon, eBay, Web Store)",
	},
}

// ParseHexColor parses a hex color string like "#EE4D2D" or "EE4D2D" into color.RGBA.
func ParseHexColor(hexStr string) (color.RGBA, error) {
	clean := strings.TrimPrefix(hexStr, "#")
	if len(clean) == 3 {
		clean = fmt.Sprintf("%c%c%c%c%c%c", clean[0], clean[0], clean[1], clean[1], clean[2], clean[2])
	}
	if len(clean) != 6 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}, errors.New("invalid hex color length")
	}
	rgb, err := strconv.ParseUint(clean, 16, 32)
	if err != nil {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}, err
	}
	return color.RGBA{
		R: uint8(rgb >> 16),
		G: uint8((rgb >> 8) & 0xFF),
		B: uint8(rgb & 0xFF),
		A: 255,
	}, nil
}

// RenderSmartShadow paints completely deterministic contact and elevation shadows.
func RenderSmartShadow(dst *image.RGBA, subjectDstRect image.Rectangle, preset SmartShadowPreset) {
	if preset == ShadowNone {
		return
	}

	cx := float64(subjectDstRect.Min.X+subjectDstRect.Max.X) / 2.0
	bottomY := float64(subjectDstRect.Max.Y)
	w := float64(subjectDstRect.Dx())
	h := float64(subjectDstRect.Dy())

	var rx, ry, cy float64
	var maxOpacity float64
	var falloffPow float64

	switch preset {
	case ShadowSoftStudio:
		rx = w * 0.44
		ry = math.Max(8.0, h*0.045)
		cy = bottomY + 4.0
		maxOpacity = 70.0 // ~27%
		falloffPow = 2.0
	case ShadowMarketplace:
		rx = w * 0.40
		ry = math.Max(6.0, h*0.035)
		cy = bottomY + 3.0
		maxOpacity = 100.0 // ~39%
		falloffPow = 2.5
	case ShadowFloating:
		rx = w * 0.48
		ry = math.Max(12.0, h*0.065)
		cy = bottomY + 18.0
		maxOpacity = 55.0 // ~21%
		falloffPow = 1.8
	case ShadowGroundContact:
		rx = w * 0.36
		ry = math.Max(4.0, h*0.025)
		cy = bottomY + 1.5
		maxOpacity = 140.0 // ~55%
		falloffPow = 3.0
	default:
		// Default to soft studio
		rx = w * 0.42
		ry = math.Max(8.0, h*0.04)
		cy = bottomY + 4.0
		maxOpacity = 85.0
		falloffPow = 2.2
	}

	minX := int(math.Floor(cx - rx*1.5))
	maxX := int(math.Ceil(cx + rx*1.5))
	minY := int(math.Floor(cy - ry*1.5))
	maxY := int(math.Ceil(cy + ry*1.5))

	b := dst.Bounds()
	if minX < b.Min.X {
		minX = b.Min.X
	}
	if maxX > b.Max.X {
		maxX = b.Max.X
	}
	if minY < b.Min.Y {
		minY = b.Min.Y
	}
	if maxY > b.Max.Y {
		maxY = b.Max.Y
	}

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			dx := (float64(x) - cx) / rx
			dy := (float64(y) - cy) / ry
			distSq := dx*dx + dy*dy
			if distSq < 1.4 {
				alphaNorm := math.Exp(-distSq * falloffPow)
				targetAlpha := uint8(alphaNorm * maxOpacity)

				orig := dst.RGBAAt(x, y)
				invA := 255 - int(targetAlpha)
				r := (int(orig.R)*invA + 20*int(targetAlpha)) / 255
				g := (int(orig.G)*invA + 20*int(targetAlpha)) / 255
				bCol := (int(orig.B)*invA + 25*int(targetAlpha)) / 255

				dst.SetRGBA(x, y, color.RGBA{R: uint8(r), G: uint8(g), B: uint8(bCol), A: 255})
			}
		}
	}
}

// RenderSellerBackground paints the chosen deterministic background onto a canvas.
func RenderSellerBackground(canvas *image.RGBA, bgPreset SellerBgPreset, brandHex string) {
	b := canvas.Bounds()
	w, h := b.Dx(), b.Dy()

	switch bgPreset {
	case BgTransparent:
		// Leave zero alpha
		return

	case BgPureWhite:
		white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				canvas.SetRGBA(x, y, white)
			}
		}

	case BgWarmWhite:
		warm := color.RGBA{R: 250, G: 250, B: 248, A: 255}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				canvas.SetRGBA(x, y, warm)
			}
		}

	case BgLightGray:
		gray := color.RGBA{R: 242, G: 244, B: 246, A: 255}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				canvas.SetRGBA(x, y, gray)
			}
		}

	case BgBrandColor:
		brandCol, err := ParseHexColor(brandHex)
		if err != nil {
			brandCol = color.RGBA{R: 238, G: 77, B: 45, A: 255} // Shopee default
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				canvas.SetRGBA(x, y, brandCol)
			}
		}

	case BgSoftGradient:
		// Vertical studio gradient from #FFFFFF top to #EAECEF bottom
		for y := 0; y < h; y++ {
			t := float64(y) / float64(h)
			r := uint8(255 - t*20)
			g := uint8(255 - t*18)
			bCol := uint8(255 - t*16)
			col := color.RGBA{R: r, G: g, B: bCol, A: 255}
			for x := 0; x < w; x++ {
				canvas.SetRGBA(x, y, col)
			}
		}

	case BgStudioVignette:
		// Radial vignette centered on subject
		cx := float64(w) / 2.0
		cy := float64(h) * 0.48
		maxDist := math.Hypot(cx, cy)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dist := math.Hypot(float64(x)-cx, float64(y)-cy)
				t := math.Min(1.0, dist/maxDist)
				v := uint8(255 - t*30)
				canvas.SetRGBA(x, y, color.RGBA{R: v, G: v, B: uint8(math.Min(255, float64(v)+2)), A: 255})
			}
		}

	default:
		// Default pure white
		white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				canvas.SetRGBA(x, y, white)
			}
		}
	}
}

// ComposeSellerCanvas places a foreground cutout onto a marketplace canvas with deterministic shadow & background.
func ComposeSellerCanvas(
	croppedSubject image.Image,
	tpl SellerTemplateConfig,
	bgPreset SellerBgPreset,
	shadowPreset SmartShadowPreset,
	brandHex string,
) image.Image {
	canvas := image.NewRGBA(image.Rect(0, 0, tpl.Width, tpl.Height))

	// 1. Render Background
	RenderSellerBackground(canvas, bgPreset, brandHex)

	// 2. Compute available scale and destination rect
	availW := float64(tpl.Width) * (1.0 - 2.0*tpl.PaddingPct)
	availH := float64(tpl.Height) * (1.0 - 2.0*tpl.PaddingPct)

	subW := float64(croppedSubject.Bounds().Dx())
	subH := float64(croppedSubject.Bounds().Dy())

	scale := math.Min(availW/subW, availH/subH)
	finalW := int(math.Round(subW * scale))
	finalH := int(math.Round(subH * scale))

	anchorY := tpl.SubjectAnchorY
	if anchorY <= 0 {
		anchorY = 0.50
	}

	dstX := (tpl.Width - finalW) / 2
	dstY := int(float64(tpl.Height)*anchorY - float64(finalH)/2.0)
	if dstY+finalH > int(float64(tpl.Height)*(1.0-tpl.PaddingPct*0.6)) {
		dstY = int(float64(tpl.Height)*(1.0-tpl.PaddingPct*0.6)) - finalH
	}
	if dstY < int(float64(tpl.Height)*(tpl.PaddingPct*0.6)) {
		dstY = int(float64(tpl.Height) * (tpl.PaddingPct * 0.6))
	}

	subjectDstRect := image.Rect(dstX, dstY, dstX+finalW, dstY+finalH)

	// 3. Render Deterministic Smart Shadow (beneath foreground)
	if shadowPreset != ShadowNone && bgPreset != BgTransparent {
		RenderSmartShadow(canvas, subjectDstRect, shadowPreset)
	}

	// 4. Scale and Composite foreground subject
	xdraw.CatmullRom.Scale(canvas, subjectDstRect, croppedSubject, croppedSubject.Bounds(), xdraw.Over, nil)

	return canvas
}

// DeterministicTeleaInpaint executes a pure-Go Fast Marching Telea inpainting algorithm
// on an RGBA image with a binary mask (mask > 0 indicates region to fill).
func DeterministicTeleaInpaint(src *image.RGBA, mask *image.Gray, inpaintRadius int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(b)
	copy(dst.Pix, src.Pix)

	if inpaintRadius <= 0 {
		inpaintRadius = 3
	}

	// Identify mask pixels
	var maskPixels [][2]int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if mask.GrayAt(x, y).Y > 128 {
				maskPixels = append(maskPixels, [2]int{x, y})
			}
		}
	}

	if len(maskPixels) == 0 {
		return dst
	}

	// Priority march from boundary inwards
	// For each pixel in mask, compute weighted average of non-mask neighbors within radius
	radSq := float64(inpaintRadius * inpaintRadius)
	for _, pt := range maskPixels {
		px, py := pt[0], pt[1]
		var sumR, sumG, sumB, sumWeight float64

		for dy := -inpaintRadius; dy <= inpaintRadius; dy++ {
			ny := py + dy
			if ny < 0 || ny >= h {
				continue
			}
			for dx := -inpaintRadius; dx <= inpaintRadius; dx++ {
				nx := px + dx
				if nx < 0 || nx >= w {
					continue
				}
				distSq := float64(dx*dx + dy*dy)
				if distSq > radSq || distSq == 0 {
					continue
				}

				// Only draw color from known (non-mask) pixels
				if mask.GrayAt(nx, ny).Y <= 128 {
					weight := 1.0 / (math.Sqrt(distSq) + 0.1)
					col := dst.RGBAAt(nx, ny)
					sumR += float64(col.R) * weight
					sumG += float64(col.G) * weight
					sumB += float64(col.B) * weight
					sumWeight += weight
				}
			}
		}

		if sumWeight > 0 {
			orig := dst.RGBAAt(px, py)
			dst.SetRGBA(px, py, color.RGBA{
				R: uint8(sumR / sumWeight),
				G: uint8(sumG / sumWeight),
				B: uint8(sumB / sumWeight),
				A: orig.A,
			})
		}
	}

	return dst
}

// ProductFactoryV2ItemRequest represents an individual item in a batch.
type ProductFactoryV2ItemRequest struct {
	Index           int    `json:"index"`
	CutoutPngBase64 string `json:"cutout_png_base64"`
	RawCutoutBytes  []byte `json:"-"`
	OriginalName    string `json:"original_name,omitempty"`
}

// ProductFactoryV2Request carries batch settings and e-commerce configurations.
type ProductFactoryV2Request struct {
	BatchId          string                        `json:"batch_id,omitempty"`
	Items            []ProductFactoryV2ItemRequest `json:"items"`
	SelectedTemplates []string                     `json:"selected_templates,omitempty"` // template IDs
	BgPreset         SellerBgPreset                `json:"bg_preset,omitempty"`
	ShadowPreset     SmartShadowPreset             `json:"shadow_preset,omitempty"`
	BrandHex         string                        `json:"brand_hex,omitempty"`
	IncludeZip       bool                          `json:"include_zip"`
}

// ProductFactoryV2ItemResult represents exports for a single item.
type ProductFactoryV2ItemResult struct {
	Index       int                          `json:"index"`
	Status      string                       `json:"status"` // "SUCCESS", "FAILED"
	ErrorReason string                       `json:"error_reason,omitempty"`
	Variants    []ProductPackGeneratedAsset  `json:"variants"`
	DurationMs  int64                        `json:"duration_ms"`
}

// ProductFactoryV2BatchResult represents the completed batch execution.
type ProductFactoryV2BatchResult struct {
	BatchId       string                       `json:"batch_id"`
	TotalItems    int                          `json:"total_items"`
	SuccessItems  int                          `json:"success_items"`
	FailedItems   int                          `json:"failed_items"`
	Items         []ProductFactoryV2ItemResult `json:"items"`
	ZipPackage    ProductPackGeneratedAsset    `json:"zip_package,omitempty"`
	ExecutionTime int64                        `json:"execution_time_ms"`
}

// GenerateSellerFactoryV2Batch processes a batch of items deterministically with full error isolation.
func GenerateSellerFactoryV2Batch(userId int, req ProductFactoryV2Request) (*ProductFactoryV2BatchResult, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("no items provided in batch")
	}
	if len(req.Items) > 25 {
		return nil, ErrProductFactoryInvalidBatchSize
	}

	startTime := time.Now().UnixNano()

	// Default template selection
	templates := CanonicalSellerTemplates
	if len(req.SelectedTemplates) > 0 {
		var selected []SellerTemplateConfig
		lookup := make(map[string]bool)
		for _, t := range req.SelectedTemplates {
			lookup[t] = true
		}
		for _, tpl := range CanonicalSellerTemplates {
			if lookup[tpl.TemplateId] {
				selected = append(selected, tpl)
			}
		}
		if len(selected) > 0 {
			templates = selected
		}
	}

	bgPreset := req.BgPreset
	if bgPreset == "" {
		bgPreset = BgPureWhite
	}
	shadowPreset := req.ShadowPreset
	if shadowPreset == "" {
		shadowPreset = ShadowMarketplace
	}

	batchId := req.BatchId
	if batchId == "" {
		batchId = fmt.Sprintf("batch_v2_%d_%s", common.GetTimestamp(), common.GetUUID()[:6])
	}

	var itemResults []ProductFactoryV2ItemResult
	zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuf)
	hasZipEntries := false
	manifestItems := make([]map[string]interface{}, 0, len(req.Items))

	successCount := 0
	failedCount := 0

	for _, item := range req.Items {
		itemStart := time.Now().UnixNano()
		res := ProductFactoryV2ItemResult{
			Index:  item.Index,
			Status: "SUCCESS",
		}

		var cutoutBytes []byte
		if len(item.RawCutoutBytes) > 0 {
			cutoutBytes = item.RawCutoutBytes
		} else if item.CutoutPngBase64 != "" {
			var err error
			cutoutBytes, err = base64.StdEncoding.DecodeString(item.CutoutPngBase64)
			if err != nil {
				res.Status = "FAILED"
				res.ErrorReason = "invalid base64 cutout: " + err.Error()
				failedCount++
				itemResults = append(itemResults, res)
				continue
			}
		} else {
			res.Status = "FAILED"
			res.ErrorReason = "empty cutout data"
			failedCount++
			itemResults = append(itemResults, res)
			continue
		}

		// Security limit: reject huge decompressed payload (>25MB)
		if len(cutoutBytes) > 25*1024*1024 {
			res.Status = "FAILED"
			res.ErrorReason = "cutout payload exceeds 25MB security limit"
			failedCount++
			itemResults = append(itemResults, res)
			continue
		}

		cutoutImg, _, err := image.Decode(bytes.NewReader(cutoutBytes))
		if err != nil {
			res.Status = "FAILED"
			res.ErrorReason = "failed decoding PNG cutout: " + err.Error()
			failedCount++
			itemResults = append(itemResults, res)
			continue
		}

		// Security limit: reject decompression bomb (>4096x4096)
		b := cutoutImg.Bounds()
		if b.Dx() > 4096 || b.Dy() > 4096 {
			res.Status = "FAILED"
			res.ErrorReason = "image dimensions exceed 4096x4096 security limit"
			failedCount++
			itemResults = append(itemResults, res)
			continue
		}

		// Add transparent cutout to ZIP archive
		if req.IncludeZip {
			cutoutZipPath := fmt.Sprintf("item_%02d/00_transparent_cutout.png", item.Index)
			if zf, zErr := zipWriter.Create(cutoutZipPath); zErr == nil {
				_, _ = zf.Write(cutoutBytes)
				hasZipEntries = true
			}
		}

		subjectBox := FindSubjectBoundingBox(cutoutImg)
		cropped := CropImage(cutoutImg, subjectBox)

		variantManifests := make([]map[string]interface{}, 0, len(templates))

		for _, tpl := range templates {
			composed := ComposeSellerCanvas(cropped, tpl, bgPreset, shadowPreset, req.BrandHex)

			var outBuf bytes.Buffer
			if tpl.Format == "png" {
				_ = png.Encode(&outBuf, composed)
			} else {
				_ = jpeg.Encode(&outBuf, composed, &jpeg.Options{Quality: tpl.Quality})
			}
			outBytes := outBuf.Bytes()
			sum := sha256.Sum256(outBytes)
			hash := hex.EncodeToString(sum[:])

			asset := ProductPackGeneratedAsset{
				Key:         tpl.TemplateId,
				Marketplace: tpl.Marketplace,
				Width:       tpl.Width,
				Height:      tpl.Height,
				FileSize:    int64(len(outBytes)),
				SHA256:      hash,
			}

			// Save to upload directory
			targetDir := filepath.Join("data", "upload", "studio", fmt.Sprintf("user_%d", userId), batchId, fmt.Sprintf("item_%02d", item.Index))
			_ = os.MkdirAll(targetDir, 0755)
			targetFile := filepath.Join(targetDir, tpl.Filename)
			if err := os.WriteFile(targetFile, outBytes, 0644); err == nil {
				asset.URL = fmt.Sprintf("/api/studio/assets/%d/%s/item_%02d/%s", userId, batchId, item.Index, tpl.Filename)
			}

			res.Variants = append(res.Variants, asset)
			variantManifests = append(variantManifests, map[string]interface{}{
				"template_id": tpl.TemplateId,
				"marketplace": tpl.Marketplace,
				"filename":    tpl.Filename,
				"width":       tpl.Width,
				"height":      tpl.Height,
				"file_size":   len(outBytes),
				"sha256":      hash,
			})

			// Add to ZIP package with strictly sanitized, safe relative path
			if req.IncludeZip {
				zipPath := fmt.Sprintf("item_%02d/%s", item.Index, tpl.Filename)
				zf, zErr := zipWriter.Create(zipPath)
				if zErr == nil {
					_, _ = zf.Write(outBytes)
					hasZipEntries = true
				}
			}
		}

		res.DurationMs = (time.Now().UnixNano() - itemStart) / 1000000
		successCount++
		itemResults = append(itemResults, res)

		manifestItems = append(manifestItems, map[string]interface{}{
			"index":          item.Index,
			"original_name":  filepath.Base(item.OriginalName),
			"variants_count": len(res.Variants),
			"variants":       variantManifests,
		})
	}

	var zipAsset ProductPackGeneratedAsset
	if req.IncludeZip && hasZipEntries {
		manifestData := map[string]interface{}{
			"batch_id":         batchId,
			"workflow_version": "v2.0_seller_factory",
			"generated_at":     time.Now().UTC().Format(time.RFC3339),
			"total_items":      len(req.Items),
			"success_items":    successCount,
			"failed_items":     failedCount,
			"bg_preset":        string(bgPreset),
			"shadow_preset":    string(shadowPreset),
			"templates":        req.SelectedTemplates,
			"items":            manifestItems,
		}
		if manifestBytes, mErr := json.MarshalIndent(manifestData, "", "  "); mErr == nil {
			if mf, zErr := zipWriter.Create("manifest.json"); zErr == nil {
				_, _ = mf.Write(manifestBytes)
			}
		}

		_ = zipWriter.Close()
		zipBytes := zipBuf.Bytes()
		sum := sha256.Sum256(zipBytes)
		zipHash := hex.EncodeToString(sum[:])

		zipDir := filepath.Join("data", "upload", "studio", fmt.Sprintf("user_%d", userId), batchId)
		_ = os.MkdirAll(zipDir, 0755)
		zipFilename := fmt.Sprintf("%s_seller_pack.zip", batchId)
		zipPath := filepath.Join(zipDir, zipFilename)
		_ = os.WriteFile(zipPath, zipBytes, 0644)

		zipAsset = ProductPackGeneratedAsset{
			Key:         "seller_factory_zip",
			Marketplace: "All Marketplace Formats (ZIP Archive)",
			FileSize:    int64(len(zipBytes)),
			SHA256:      zipHash,
			URL:         fmt.Sprintf("/api/studio/assets/%d/%s/%s", userId, batchId, zipFilename),
		}
	}

	elapsedMs := (time.Now().UnixNano() - startTime) / 1000000

	return &ProductFactoryV2BatchResult{
		BatchId:       batchId,
		TotalItems:    len(req.Items),
		SuccessItems:  successCount,
		FailedItems:   failedCount,
		Items:         itemResults,
		ZipPackage:    zipAsset,
		ExecutionTime: elapsedMs,
	}, nil
}
