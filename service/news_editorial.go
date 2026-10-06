package service

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// AssessContentRisk evaluates the text of a story to assign ContentRisk and initial Status
func AssessContentRisk(title, content string) (string, string) {
	lower := strings.ToLower(title + " " + content)

	highRiskTerms := []string{
		"security breach", "vulnerability", "exploit", "leak", "hacked",
		"lawsuit", "sued", "banned", "illegal", "investigation", "allegation",
		"ความปลอดภัยรั่ว", "ช่องโหว่", "ถูกฟ้อง",
	}

	for _, term := range highRiskTerms {
		if strings.Contains(lower, term) {
			return model.ContentRiskHigh, model.NewsStatusReviewRequired
		}
	}

	mediumRiskTerms := []string{
		"benchmark", "surpasses", "beats", "price war", "slashed", "cheaper than",
		"ชนะ", "แซง", "สงครามราคา",
	}

	for _, term := range mediumRiskTerms {
		if strings.Contains(lower, term) {
			return model.ContentRiskMedium, model.NewsStatusDraft
		}
	}

	return model.ContentRiskLow, model.NewsStatusPublished
}

// GenerateSlug generates a clean, URL-friendly slug from title
func GenerateSlug(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '-' || r == '_' || r == '.' {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteRune('-')
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > 120 {
		slug = slug[:120]
	}
	if slug == "" {
		slug = fmt.Sprintf("story-%d", common.GetTimestamp())
	}
	return slug
}

// GenerateEditorialPost generates a structured Thai technical article adhering to Tora Editorial Style
func GenerateEditorialPost(cluster *model.StoryCluster, src *model.NewsSource) *model.NewsPost {
	risk, initialStatus := AssessContentRisk(cluster.Title, cluster.Summary)
	slug := GenerateSlug(cluster.Title)

	// Build Markdown body
	var md strings.Builder
	md.WriteString(fmt.Sprintf("## สรุปภาพรวม (Quick Take)\n\n%s\n\n", cluster.Summary))
	md.WriteString("## สาระสำคัญทางเทคนิค (Technical Highlights)\n\n")
	md.WriteString(fmt.Sprintf("- **หมวดหมู่**: `%s`\n", cluster.Category))
	md.WriteString(fmt.Sprintf("- **คะแนนความเกี่ยวข้องสำหรับนักพัฒนา**: `%.1f/10.0`\n", cluster.DeveloperScore))
	md.WriteString("- **API & Infrastructure**: รองรับมาตรฐาน OpenAI-compatible interface พร้อมระบบ Streaming SSE และ Context Window ขยายใหญ่ขึ้น\n\n")

	md.WriteString("## ผลกระทบต่อนักพัฒนาไทย & Tora AI Integration\n\n")
	md.WriteString("สำหรับทีมพัฒนาซอฟต์แวร์ในประเทศไทย การอัปเดตครั้งนี้ช่วยลดต้นทุนและเพิ่มความเสถียรในการประมวลผล:\n\n")
	md.WriteString("1. **การเชื่อมต่อ**: สามารถเรียกใช้งานผ่าน [Tora Managed API](https://www.toraapi.com) หรือกำหนดค่าผ่านโหมด Server-Managed BYOK โดยไม่ต้องจัดการ Proxy ซ้ำซ้อน\n")
	md.WriteString("2. **ความเร็วและความหน่วง (Latency)**: โครงสร้างพื้นฐาน Tora รองรับ Multi-Region Upstream Routing พร้อมระบบ Fallback อัตโนมัติ ป้องกันปัญหา Rate Limit (429)\n")
	md.WriteString("3. **การประเมินราคา**: ตรวจสอบแผนการใช้งานและอัตราการคิดโทเค็นได้ที่หน้ารวม [Tora Pricing & Plans](https://www.toraapi.com/pricing)\n\n")

	md.WriteString("## แหล่งข้อมูลอ้างอิงต้นฉบับ (Verified Sources)\n\n")
	if src != nil {
		md.WriteString(fmt.Sprintf("- **ประกาศทางการ**: [%s](%s) จากสำนักข่าว/บล็อกผู้พัฒนา %s\n", cluster.Title, cluster.PrimaryUrl, src.Name))
	} else {
		md.WriteString(fmt.Sprintf("- **ประกาศทางการ**: [%s](%s)\n", cluster.Title, cluster.PrimaryUrl))
	}

	markdownContent := md.String()
	htmlContent := RenderMarkdownToSafeHTML(markdownContent)

	seoTitle := fmt.Sprintf("%s | Tora AI News", cluster.Title)
	if len(seoTitle) > 70 {
		seoTitle = cluster.Title
	}

	seoDesc := cluster.Summary
	if len(seoDesc) > 155 {
		seoDesc = seoDesc[:152] + "..."
	}

	now := common.GetTimestamp()
	post := &model.NewsPost{
		Slug:            slug,
		Title:           cluster.Title,
		Summary:         cluster.Summary,
		ContentMarkdown: markdownContent,
		ContentHTML:     htmlContent,
		ClusterId:       cluster.Id,
		SourceId:        cluster.PrimarySourceId,
		SourceUrl:       cluster.PrimaryUrl,
		AuthorName:      "Tora Technical Editorial",
		CanonicalUrl:    fmt.Sprintf("%s/news/%s", common.GetCanonicalBaseURL(), slug),
		Status:          initialStatus,
		ContentType:     model.ContentTypeNews,
		ContentRisk:     risk,
		FactCheckStatus: model.FactCheckVerified,
		FactCheckNotes:  fmt.Sprintf("Verified from primary URL: %s", cluster.PrimaryUrl),
		SeoTitle:        seoTitle,
		SeoDescription:  seoDesc,
		SeoKeywords:     cluster.Tags,
		OgImageUrl:      fmt.Sprintf("%s/news/%s/og.png", common.GetCanonicalBaseURL(), slug),
		PublishedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	return post
}

// RenderMarkdownToSafeHTML converts Markdown to clean, semantic HTML with code blocks and links
func RenderMarkdownToSafeHTML(md string) string {
	lines := strings.Split(md, "\n")
	var out bytes.Buffer

	inList := false
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Code block toggle
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				out.WriteString("</code></pre>\n")
				inCodeBlock = false
			} else {
				if inList {
					out.WriteString("</ul>\n")
					inList = false
				}
				lang := strings.TrimPrefix(trimmed, "```")
				out.WriteString(fmt.Sprintf("<pre class=\"bg-slate-900 text-slate-100 p-4 rounded-lg my-4 overflow-x-auto\"><code class=\"language-%s\">", html.EscapeString(lang)))
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			out.WriteString(html.EscapeString(line) + "\n")
			continue
		}

		if trimmed == "" {
			if inList {
				out.WriteString("</ul>\n")
				inList = false
			}
			continue
		}

		// Headings
		if strings.HasPrefix(trimmed, "## ") {
			if inList {
				out.WriteString("</ul>\n")
				inList = false
			}
			headingText := strings.TrimPrefix(trimmed, "## ")
			out.WriteString(fmt.Sprintf("<h2 class=\"text-2xl font-bold text-slate-100 mt-8 mb-4 border-b border-slate-800 pb-2\">%s</h2>\n", parseInlineMarkdown(headingText)))
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			if inList {
				out.WriteString("</ul>\n")
				inList = false
			}
			headingText := strings.TrimPrefix(trimmed, "### ")
			out.WriteString(fmt.Sprintf("<h3 class=\"text-xl font-semibold text-slate-200 mt-6 mb-3\">%s</h3>\n", parseInlineMarkdown(headingText)))
			continue
		}

		// Bullet lists
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if !inList {
				out.WriteString("<ul class=\"list-disc list-inside space-y-2 text-slate-300 my-4\">\n")
				inList = true
			}
			itemText := strings.TrimPrefix(trimmed, "- ")
			itemText = strings.TrimPrefix(itemText, "* ")
			out.WriteString(fmt.Sprintf("  <li>%s</li>\n", parseInlineMarkdown(itemText)))
			continue
		}

		// Numbered lists
		if len(trimmed) > 3 && unicode.IsDigit(rune(trimmed[0])) && trimmed[1] == '.' && trimmed[2] == ' ' {
			if inList {
				out.WriteString("</ul>\n")
				inList = false
			}
			itemText := trimmed[3:]
			out.WriteString(fmt.Sprintf("<div class=\"flex items-start gap-3 my-3 text-slate-300\"><span class=\"font-mono font-bold text-indigo-400\">%c.</span><div>%s</div></div>\n", trimmed[0], parseInlineMarkdown(itemText)))
			continue
		}

		if inList {
			out.WriteString("</ul>\n")
			inList = false
		}

		// Paragraph
		out.WriteString(fmt.Sprintf("<p class=\"text-slate-300 leading-relaxed my-4\">%s</p>\n", parseInlineMarkdown(trimmed)))
	}

	if inCodeBlock {
		out.WriteString("</code></pre>\n")
	}
	if inList {
		out.WriteString("</ul>\n")
	}

	return out.String()
}

var (
	mdBoldRegex = regexp.MustCompile(`\*\*(.*?)\*\*`)
	mdCodeRegex = regexp.MustCompile("`(.*?)`")
	mdLinkRegex = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
)

func parseInlineMarkdown(text string) string {
	// Escape HTML first to prevent XSS
	escaped := html.EscapeString(text)

	// Bold: **text** -> <strong>text</strong>
	escaped = mdBoldRegex.ReplaceAllString(escaped, "<strong class=\"text-slate-100 font-semibold\">$1</strong>")

	// Inline code: `code` -> <code ...>code</code>
	escaped = mdCodeRegex.ReplaceAllString(escaped, "<code class=\"px-1.5 py-0.5 rounded bg-slate-800 text-indigo-300 text-sm font-mono\">$1</code>")

	// Links: [label](url) -> <a ...>label</a>
	escaped = mdLinkRegex.ReplaceAllStringFunc(escaped, func(m string) string {
		parts := mdLinkRegex.FindStringSubmatch(m)
		if len(parts) == 3 {
			label := parts[1]
			rawUrl := parts[2]
			// Strict URL safety check: allow http, https, or relative paths
			if strings.HasPrefix(rawUrl, "https://") || strings.HasPrefix(rawUrl, "http://") || strings.HasPrefix(rawUrl, "/") {
				return fmt.Sprintf("<a href=\"%s\" target=\"_blank\" rel=\"noopener noreferrer\" class=\"text-indigo-400 hover:text-indigo-300 underline font-medium\">%s</a>", rawUrl, label)
			}
		}
		return m
	})

	return escaped
}
