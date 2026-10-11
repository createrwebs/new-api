package service

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
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
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	xdraw "golang.org/x/image/draw"
	"gorm.io/gorm"
)

// ProductPackDimension defines target canvas dimensions and formatting.
type ProductPackDimension struct {
	Key         string  `json:"key"`
	Filename    string  `json:"filename"`
	Marketplace string  `json:"marketplace"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	AspectRatio string  `json:"aspect_ratio"`
	PaddingPct  float64 `json:"padding_pct"`
	BgColor     color.Color `json:"-"`
	Format      string  `json:"format"`
	Quality     int     `json:"quality"`
	Description string  `json:"description"`
}

var DefaultProductPackDimensions = []ProductPackDimension{
	{
		Key:         "shopee",
		Filename:    "01_shopee_800x800.jpg",
		Marketplace: "Shopee 1:1",
		Width:       800,
		Height:      800,
		AspectRatio: "1:1",
		PaddingPct:  0.10,
		BgColor:     color.White,
		Format:      "jpg",
		Quality:     92,
		Description: "Shopee White Catalog Standard (800x800)",
	},
	{
		Key:         "lazada",
		Filename:    "02_lazada_1000x1000.jpg",
		Marketplace: "Lazada 1:1",
		Width:       1000,
		Height:      1000,
		AspectRatio: "1:1",
		PaddingPct:  0.10,
		BgColor:     color.White,
		Format:      "jpg",
		Quality:     92,
		Description: "Lazada HD Product Listing (1000x1000)",
	},
	{
		Key:         "instagram",
		Filename:    "03_instagram_1080x1350.jpg",
		Marketplace: "Instagram Post 4:5",
		Width:       1080,
		Height:      1350,
		AspectRatio: "4:5",
		PaddingPct:  0.12,
		BgColor:     color.RGBA{R: 250, G: 250, B: 250, A: 255},
		Format:      "jpg",
		Quality:     95,
		Description: "Instagram Feed Engagement Post (1080x1350)",
	},
	{
		Key:         "story",
		Filename:    "04_story_tiktok_1080x1920.jpg",
		Marketplace: "IG Story / TikTok 9:16",
		Width:       1080,
		Height:      1920,
		AspectRatio: "9:16",
		PaddingPct:  0.15,
		BgColor:     color.RGBA{R: 248, G: 249, B: 250, A: 255},
		Format:      "jpg",
		Quality:     95,
		Description: "Vertical Story & TikTok Showcase (1080x1920)",
	},
}

// ProductPackGeneratedAsset holds an exported variant.
type ProductPackGeneratedAsset struct {
	Key         string `json:"key"`
	Marketplace string `json:"marketplace"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	URL         string `json:"url"`
	AssetId     string `json:"asset_id"`
	FileSize    int64  `json:"file_size"`
	SHA256      string `json:"sha256"`
}

// ProductPackResult represents the complete generated pack output.
type ProductPackResult struct {
	PackId      string                      `json:"pack_id"`
	Variants    []ProductPackGeneratedAsset `json:"variants"`
	Transparent ProductPackGeneratedAsset   `json:"transparent_cutout"`
	ZipPackage  ProductPackGeneratedAsset   `json:"zip_package"`
	SubjectBox  image.Rectangle             `json:"subject_box"`
	CreatedAt   int64                       `json:"created_at"`
}

// FindSubjectBoundingBox locates the non-transparent foreground region of an image.
func FindSubjectBoundingBox(img image.Image) image.Rectangle {
	b := img.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y
	found := false

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			// If alpha > 5% (13/255 -> 3300/65535)
			if a > 3300 {
				found = true
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if !found || minX >= maxX || minY >= maxY {
		return b
	}

	// Add 1px margin around bounding box
	if minX > b.Min.X {
		minX--
	}
	if minY > b.Min.Y {
		minY--
	}
	if maxX < b.Max.X-1 {
		maxX++
	}
	if maxY < b.Max.Y-1 {
		maxY++
	}

	return image.Rect(minX, minY, maxX+1, maxY+1)
}

// CropImage extracts a sub-region from an image.
func CropImage(src image.Image, rect image.Rectangle) image.Image {
	cropped := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			cropped.Set(x, y, src.At(rect.Min.X+x, rect.Min.Y+y))
		}
	}
	return cropped
}

// RenderDropShadow paints a natural radial contact shadow beneath the subject's base.
func RenderDropShadow(dst *image.RGBA, subjectDstRect image.Rectangle) {
	// Shadow parameters
	cx := float64(subjectDstRect.Min.X + subjectDstRect.Max.X) / 2.0
	cy := float64(subjectDstRect.Max.Y) + 4.0
	rx := float64(subjectDstRect.Dx()) * 0.42
	ry := math.Max(8.0, float64(subjectDstRect.Dy())*0.04)

	minX := int(math.Floor(cx - rx*1.4))
	maxX := int(math.Ceil(cx + rx*1.4))
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
				// Gaussian-like falloff
				alphaNorm := math.Exp(-distSq * 2.2)
				targetAlpha := uint8(alphaNorm * 90.0) // max ~35% opacity

				orig := dst.RGBAAt(x, y)
				// Alpha blend over background
				invA := 255 - int(targetAlpha)
				r := (int(orig.R)*invA + 30*int(targetAlpha)) / 255
				g := (int(orig.G)*invA + 30*int(targetAlpha)) / 255
				bCol := (int(orig.B)*invA + 30*int(targetAlpha)) / 255

				dst.SetRGBA(x, y, color.RGBA{R: uint8(r), G: uint8(g), B: uint8(bCol), A: 255})
			}
		}
	}
}

// ComposeProductCanvas generates a centered product presentation on a canvas.
func ComposeProductCanvas(croppedSubject image.Image, dim ProductPackDimension) image.Image {
	canvas := image.NewRGBA(image.Rect(0, 0, dim.Width, dim.Height))

	// 1. Fill background
	bgColor, ok := dim.BgColor.(color.RGBA)
	if !ok {
		bgColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	for y := 0; y < dim.Height; y++ {
		for x := 0; x < dim.Width; x++ {
			canvas.SetRGBA(x, y, bgColor)
		}
	}

	// 2. Compute available area with padding
	availW := float64(dim.Width) * (1.0 - 2.0*dim.PaddingPct)
	availH := float64(dim.Height) * (1.0 - 2.0*dim.PaddingPct)

	subW := float64(croppedSubject.Bounds().Dx())
	subH := float64(croppedSubject.Bounds().Dy())

	scale := math.Min(availW/subW, availH/subH)
	if scale > 3.0 {
		scale = 3.0 // Guard against excessive pixel magnification
	}

	targetW := int(math.Round(subW * scale))
	targetH := int(math.Round(subH * scale))
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}

	// Center horizontally and vertically (slightly offset up for shadow)
	targetX := (dim.Width - targetW) / 2
	targetY := (dim.Height - targetH) / 2 - int(float64(dim.Height)*0.015)
	if targetY < 0 {
		targetY = (dim.Height - targetH) / 2
	}

	destRect := image.Rect(targetX, targetY, targetX+targetW, targetY+targetH)

	// 3. Render drop shadow underneath
	RenderDropShadow(canvas, destRect)

	// 4. Scale subject and draw onto canvas
	xdraw.BiLinear.Scale(canvas, destRect, croppedSubject, croppedSubject.Bounds(), xdraw.Over, nil)

	return canvas
}

// PersistPackAsset saves an image buffer to the studio asset store.
func PersistPackAsset(db *gorm.DB, userId int, jobId string, filename string, data []byte, mimeType string) (*ProductPackGeneratedAsset, error) {
	if err := CheckStorageQuota(db, userId, int64(len(data))); err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	uploadDir := GetStudioUploadDir()
	actualFilename := fmt.Sprintf("prodpack_%d_%s_%s", now, common.GetUUID()[:6], filename)
	filePath := filepath.Join(uploadDir, actualFilename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed saving product pack asset: %w", err)
	}

	hashBytes := sha256.Sum256(data)
	sha256Hex := hex.EncodeToString(hashBytes[:])

	serverURL := strings.TrimRight(os.Getenv("SERVER_URL"), "/")
	if serverURL == "" {
		serverURL = "https://www.toraapi.com"
	}
	assetURL := fmt.Sprintf("%s/api/studio/assets/%s", serverURL, actualFilename)

	assetId := fmt.Sprintf("ast_%d_%s", now, common.GetUUID()[:8])
	asset := &model.StudioAsset{
		Id:                 assetId,
		UserId:             userId,
		JobId:              jobId,
		AssetType:          "output",
		MIMEType:           mimeType,
		FileSize:           int64(len(data)),
		SHA256:             sha256Hex,
		StorageURL:         assetURL,
		AvailabilityStatus: "available",
		LifecycleState:     model.AssetLifecycleJobOutput,
		ExpiryAt:           now + int64(DefaultOutputAssetTTL.Seconds()),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if db != nil {
		if err := model.CreateStudioAsset(asset); err != nil {
			_ = os.Remove(filePath)
			return nil, err
		}
	}

	return &ProductPackGeneratedAsset{
		Key:      filename,
		URL:      assetURL,
		AssetId:  assetId,
		FileSize: int64(len(data)),
		SHA256:   sha256Hex,
	}, nil
}

// GenerateMarketplaceProductPack executes the full deterministic pipeline from a source image.
func GenerateMarketplaceProductPack(db *gorm.DB, userId int, jobId string, rawImageBytes []byte) (*ProductPackResult, error) {
	if len(rawImageBytes) == 0 {
		return nil, errors.New("raw image bytes is empty")
	}

	srcImg, _, err := image.Decode(bytes.NewReader(rawImageBytes))
	if err != nil {
		return nil, fmt.Errorf("failed decoding source image: %w", err)
	}

	// 1. Locate subject bounding box and crop
	bbox := FindSubjectBoundingBox(srcImg)
	cropped := CropImage(srcImg, bbox)

	now := common.GetTimestamp()
	packId := fmt.Sprintf("pack_%d_%s", now, common.GetUUID()[:8])

	var variants []ProductPackGeneratedAsset
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	// 2. Export transparent cutout
	cutoutBuf := new(bytes.Buffer)
	if err := png.Encode(cutoutBuf, cropped); err == nil {
		cutoutData := cutoutBuf.Bytes()
		asset, err := PersistPackAsset(db, userId, jobId, "cutout_transparent.png", cutoutData, "image/png")
		if err == nil {
			asset.Marketplace = "Transparent Cutout PNG"
			asset.Width = bbox.Dx()
			asset.Height = bbox.Dy()

			// Add to ZIP
			f, _ := zipWriter.Create("05_product_cutout_transparent.png")
			if f != nil {
				_, _ = f.Write(cutoutData)
			}
		}
	}

	// 3. Render and persist all marketplace presets
	for _, dim := range DefaultProductPackDimensions {
		composed := ComposeProductCanvas(cropped, dim)

		buf := new(bytes.Buffer)
		if dim.Format == "jpg" {
			_ = jpeg.Encode(buf, composed, &jpeg.Options{Quality: dim.Quality})
		} else {
			_ = png.Encode(buf, composed)
		}

		imgData := buf.Bytes()
		mime := "image/jpeg"
		if dim.Format == "png" {
			mime = "image/png"
		}

		asset, err := PersistPackAsset(db, userId, jobId, dim.Filename, imgData, mime)
		if err != nil {
			continue
		}
		asset.Key = dim.Key
		asset.Marketplace = dim.Marketplace
		asset.Width = dim.Width
		asset.Height = dim.Height
		variants = append(variants, *asset)

		// Add variant to ZIP package
		if f, err := zipWriter.Create(dim.Filename); err == nil {
			_, _ = f.Write(imgData)
		}
	}

	// 4. Add metadata manifest to ZIP
	manifestData, _ := json.MarshalIndent(map[string]interface{}{
		"pack_id":              packId,
		"generated_at":         time.Now().UTC().Format(time.RFC3339),
		"marketplace_channels": []string{"Shopee", "Lazada", "Instagram", "TikTok / Reels"},
		"bounding_box": map[string]int{
			"width":  bbox.Dx(),
			"height": bbox.Dy(),
		},
		"compliance": "Standard E-commerce Pure White Background & Mobile Aspect Ratios",
	}, "", "  ")

	if f, err := zipWriter.Create("manifest.json"); err == nil {
		_, _ = f.Write(manifestData)
	}

	_ = zipWriter.Close()

	// 5. Persist ZIP package
	zipData := zipBuffer.Bytes()
	zipAsset, err := PersistPackAsset(db, userId, jobId, fmt.Sprintf("%s_full_pack.zip", packId), zipData, "application/zip")
	if err != nil {
		return nil, fmt.Errorf("failed persisting zip package: %w", err)
	}
	zipAsset.Marketplace = "All-In-One ZIP Export"

	return &ProductPackResult{
		PackId:     packId,
		Variants:   variants,
		ZipPackage: *zipAsset,
		SubjectBox: bbox,
		CreatedAt:  now,
	}, nil
}
