package service

import (
	"bytes"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	cachedThaiFont     *opentype.Font
	cachedThaiFontOnce sync.Once
)

func loadSystemThaiFont() *opentype.Font {
	cachedThaiFontOnce.Do(func() {
		candidates := []string{
			os.Getenv("THAI_FONT_PATH"),
			"/System/Library/Fonts/Supplemental/Ayuthaya.ttf",
			"/System/Library/Fonts/Supplemental/Thonburi.ttc",
			"/System/Library/Fonts/Supplemental/Sarabun-Regular.ttf",
			"/usr/share/fonts/truetype/tlwg/Sarabun.ttf",
			"/usr/share/fonts/truetype/tlwg/Waree.ttf",
			"/usr/share/fonts/truetype/thai/Sarabun.ttf",
		}
		for _, p := range candidates {
			if p == "" {
				continue
			}
			data, err := os.ReadFile(p)
			if err == nil {
				f, err := opentype.Parse(data)
				if err == nil {
					cachedThaiFont = f
					return
				}
			}
		}
	})
	return cachedThaiFont
}

func getThaiFace(size float64) (font.Face, error) {
	f := loadSystemThaiFont()
	if f == nil {
		return nil, fmt.Errorf("no system thai font found")
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size: size,
		DPI:  72,
	})
}

// GenerateOGCardSVG creates a 1200x630 branded SVG social preview card
func GenerateOGCardSVG(post *model.NewsPost) string {
	title := post.Title
	if len(title) > 90 {
		title = title[:87] + "..."
	}

	dateStr := time.Unix(post.PublishedAt, 0).Format("02 Jan 2006")
	category := "TECH & AI"
	if post.ContentType != "" && post.ContentType != model.ContentTypeNews {
		category = strings.ToUpper(post.ContentType)
	} else if post.SeoKeywords != "" {
		parts := strings.Split(post.SeoKeywords, ",")
		if len(parts) > 0 {
			category = strings.ToUpper(strings.TrimSpace(parts[0]))
		}
	}

	lines := wrapText(title, 42)
	line1 := ""
	line2 := ""
	if len(lines) > 0 {
		line1 = html.EscapeString(lines[0])
	}
	if len(lines) > 1 {
		line2 = html.EscapeString(lines[1])
	}

	svg := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="1200" height="630" viewBox="0 0 1200 630" fill="none" xmlns="http://www.w3.org/2000/svg">
  <!-- Background gradient -->
  <rect width="1200" height="630" fill="#0B0F19"/>
  <circle cx="1050" cy="150" r="350" fill="#4F46E5" fill-opacity="0.18" filter="blur(80px)"/>
  <circle cx="150" cy="500" r="300" fill="#06B6D4" fill-opacity="0.15" filter="blur(80px)"/>
  <rect x="0" y="622" width="1200" height="8" fill="url(#brand-bar)"/>
  <rect x="0" y="0" width="1200" height="8" fill="url(#brand-bar)"/>

  <defs>
    <linearGradient id="brand-bar" x1="0" y1="0" x2="1200" y2="0" gradientUnits="userSpaceOnUse">
      <stop stop-color="#4F46E5"/>
      <stop offset="0.5" stop-color="#06B6D4"/>
      <stop offset="1" stop-color="#10B981"/>
    </linearGradient>
  </defs>

  <!-- Brand header -->
  <g transform="translate(80, 80)">
    <rect width="44" height="44" rx="10" fill="#4F46E5"/>
    <text x="22" y="29" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-weight="900" font-size="22" fill="#FFFFFF" text-anchor="middle">T</text>
    <text x="58" y="29" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-weight="800" font-size="24" fill="#FFFFFF" letter-spacing="-0.5">Tora AI</text>
    <text x="145" y="29" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-weight="600" font-size="16" fill="#818CF8">| NEWSROOM</text>
  </g>

  <!-- Category badge -->
  <g transform="translate(80, 180)">
    <rect width="%d" height="34" rx="6" fill="#1E1B4B" stroke="#4F46E5" stroke-width="1"/>
    <text x="14" y="22" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-weight="700" font-size="13" fill="#A5B4FC" letter-spacing="1">%s</text>
  </g>

  <!-- Title lines -->
  <text x="80" y="280" font-family="-apple-system, BlinkMacSystemFont, 'Sarabun', 'Segoe UI', Roboto, sans-serif" font-weight="800" font-size="44" fill="#F8FAFC" letter-spacing="-0.5">
    %s
  </text>
  <text x="80" y="340" font-family="-apple-system, BlinkMacSystemFont, 'Sarabun', 'Segoe UI', Roboto, sans-serif" font-weight="800" font-size="44" fill="#F8FAFC" letter-spacing="-0.5">
    %s
  </text>

  <!-- Meta footer -->
  <g transform="translate(80, 520)">
    <text x="0" y="0" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-weight="500" font-size="16" fill="#94A3B8">
      Published: %s  •  Author: %s  •  toraapi.com
    </text>
  </g>
</svg>`,
		len(category)*11+28,
		html.EscapeString(category),
		line1,
		line2,
		dateStr,
		html.EscapeString(post.AuthorName),
	)

	return svg
}

// GenerateOGCardPNG creates a 1200x630 branded raster PNG social preview image
func GenerateOGCardPNG(post *model.NewsPost) ([]byte, error) {
	const width = 1200
	const height = 630

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 1. Fill Background (#0B0F19)
	bgColor := color.RGBA{R: 11, G: 15, B: 25, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	// 2. Decorative glowing rects
	glowIndigo := color.RGBA{R: 79, G: 70, B: 229, A: 35}
	drawFillRect(img, 850, 50, 1150, 350, glowIndigo)
	glowCyan := color.RGBA{R: 6, G: 182, B: 212, A: 30}
	drawFillRect(img, 50, 350, 400, 600, glowCyan)

	// 3. Top and Bottom Brand Accent Bars
	drawGradientBar(img, 0, 0, width, 8)
	drawGradientBar(img, 0, height-8, width, height)

	// 4. Brand Mark Box: 48x48 rounded rectangle in Indigo (#4F46E5)
	brandIndigo := color.RGBA{R: 79, G: 70, B: 229, A: 255}
	drawFillRect(img, 80, 70, 80+48, 70+48, brandIndigo)

	// Draw 'T' inside logo box (Scale 3x)
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	drawScaledString(img, 96, 78, "T", white, 3)

	// Brand Title: "Tora AI | NEWSROOM"
	drawScaledString(img, 142, 82, "Tora AI", white, 2)
	indigoText := color.RGBA{R: 129, G: 140, B: 248, A: 255}
	drawScaledString(img, 245, 82, "| NEWSROOM", indigoText, 2)

	// 5. Category Badge
	category := "TECH & AI"
	if post.ContentType != "" && post.ContentType != model.ContentTypeNews {
		category = strings.ToUpper(post.ContentType)
	} else if post.SeoKeywords != "" {
		parts := strings.Split(post.SeoKeywords, ",")
		if len(parts) > 0 {
			category = strings.ToUpper(strings.TrimSpace(parts[0]))
		}
	}
	badgeBg := color.RGBA{R: 30, G: 27, B: 75, A: 255}
	badgeWidth := len(category)*14 + 30
	drawFillRect(img, 80, 160, 80+badgeWidth, 160+34, badgeBg)
	drawStrokeRect(img, 80, 160, 80+badgeWidth, 160+34, brandIndigo)
	drawScaledString(img, 95, 170, category, indigoText, 1)

	// 6. Title Lines (Prominent headline)
	title := post.Title
	isAscii := true
	for _, r := range title {
		if r > 127 {
			isAscii = false
			break
		}
	}
	if len(title) > 85 {
		title = title[:82] + "..."
	}

	titleColor := color.RGBA{R: 248, G: 250, B: 252, A: 255}
	if !isAscii {
		// Attempt rendering native Thai/Unicode using system TrueType font
		face, err := getThaiFace(40)
		if err == nil && face != nil {
			defer face.Close()
			lines := wrapText(title, 34)
			yOffset := 245
			for i, l := range lines {
				if i >= 2 {
					break
				}
				d := &font.Drawer{
					Dst:  img,
					Src:  image.NewUniform(titleColor),
					Face: face,
					Dot:  fixed.Point26_6{X: fixed.I(80), Y: fixed.I(yOffset)},
				}
				d.DrawString(l)
				yOffset += 56
			}
		} else {
			// Fallback: draw title lines with scaled font
			lines := wrapText(title, 42)
			yOffset := 240
			for i, l := range lines {
				if i >= 2 {
					break
				}
				drawScaledString(img, 80, yOffset, l, titleColor, 3)
				yOffset += 60
			}
		}
	} else {
		lines := wrapText(title, 42)
		yOffset := 240
		for i, l := range lines {
			if i >= 2 {
				break
			}
			drawScaledString(img, 80, yOffset, l, titleColor, 3)
			yOffset += 60
		}
	}

	// 7. Footer Metadata
	dateStr := time.Unix(post.PublishedAt, 0).Format("02 Jan 2006")
	metaColor := color.RGBA{R: 148, G: 163, B: 184, A: 255}
	footerText := fmt.Sprintf("Published: %s  |  Author: %s  |  toraapi.com", dateStr, post.AuthorName)
	drawScaledString(img, 80, 530, footerText, metaColor, 2)

	// 8. Encode to PNG buffer
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func drawFillRect(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	for y := y1; y < y2 && y < img.Rect.Max.Y; y++ {
		for x := x1; x < x2 && x < img.Rect.Max.X; x++ {
			if x >= 0 && y >= 0 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func drawStrokeRect(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	for x := x1; x < x2 && x < img.Rect.Max.X; x++ {
		if x >= 0 {
			if y1 >= 0 && y1 < img.Rect.Max.Y {
				img.SetRGBA(x, y1, c)
			}
			if y2-1 >= 0 && y2-1 < img.Rect.Max.Y {
				img.SetRGBA(x, y2-1, c)
			}
		}
	}
	for y := y1; y < y2 && y < img.Rect.Max.Y; y++ {
		if y >= 0 {
			if x1 >= 0 && x1 < img.Rect.Max.X {
				img.SetRGBA(x1, y, c)
			}
			if x2-1 >= 0 && x2-1 < img.Rect.Max.X {
				img.SetRGBA(x2-1, y, c)
			}
		}
	}
}

func drawGradientBar(img *image.RGBA, x1, y1, x2, y2 int) {
	totalWidth := x2 - x1
	if totalWidth <= 0 {
		return
	}
	for x := x1; x < x2 && x < img.Rect.Max.X; x++ {
		ratio := float64(x-x1) / float64(totalWidth)
		var c color.RGBA
		if ratio < 0.5 {
			subRatio := ratio / 0.5
			c = color.RGBA{
				R: uint8(79 + subRatio*(6-79)),
				G: uint8(70 + subRatio*(182-70)),
				B: uint8(229 + subRatio*(212-229)),
				A: 255,
			}
		} else {
			subRatio := (ratio - 0.5) / 0.5
			c = color.RGBA{
				R: uint8(6 + subRatio*(16-6)),
				G: uint8(182 + subRatio*(185-182)),
				B: uint8(212 + subRatio*(129-212)),
				A: 255,
			}
		}
		for y := y1; y < y2 && y < img.Rect.Max.Y; y++ {
			if x >= 0 && y >= 0 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func formatSlugToHeadline(slug string) string {
	parts := strings.Split(slug, "-")
	var words []string
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		words = append(words, strings.ToUpper(p[:1])+p[1:])
	}
	return strings.Join(words, " ")
}

func sanitizeASCII(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 32 && r <= 126 {
			sb.WriteRune(r)
		} else if r == '•' || r == '·' {
			sb.WriteByte('-')
		}
	}
	return sb.String()
}

func drawScaledString(dst *image.RGBA, startX, startY int, text string, col color.RGBA, scale int) {
	cleanText := sanitizeASCII(text)
	if cleanText == "" {
		return
	}
	if scale <= 1 {
		d := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(col),
			Face: basicfont.Face7x13,
			Dot:  fixed.Point26_6{X: fixed.I(startX), Y: fixed.I(startY + 11)},
		}
		d.DrawString(cleanText)
		return
	}

	// Render to scratch image then nearest-neighbor blit
	scratchWidth := len(cleanText)*7 + 4
	scratchHeight := 14
	if scratchWidth <= 0 {
		return
	}
	scratch := image.NewRGBA(image.Rect(0, 0, scratchWidth, scratchHeight))
	d := &font.Drawer{
		Dst:  scratch,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.I(0), Y: fixed.I(11)},
	}
	d.DrawString(cleanText)

	// Blit scaled
	for sy := 0; sy < scratchHeight; sy++ {
		for sx := 0; sx < scratchWidth; sx++ {
			pixel := scratch.RGBAAt(sx, sy)
			if pixel.A > 0 {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						targetX := startX + sx*scale + dx
						targetY := startY + sy*scale + dy
						if targetX >= 0 && targetX < dst.Rect.Max.X && targetY >= 0 && targetY < dst.Rect.Max.Y {
							dst.SetRGBA(targetX, targetY, pixel)
						}
					}
				}
			}
		}
	}
}

func wrapText(text string, maxChars int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	var current strings.Builder

	for _, w := range words {
		if current.Len()+len(w)+1 > maxChars && current.Len() > 0 {
			lines = append(lines, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(w)
		if len(lines) >= 2 {
			break
		}
	}
	if current.Len() > 0 && len(lines) < 2 {
		lines = append(lines, current.String())
	}
	return lines
}
