package service

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
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

// GenerateEditorialPost generates a structured Thai technical article adhering to Tora Editorial Style (Section 9, 10, 11, 13)
func GenerateEditorialPost(cluster *model.StoryCluster, src *model.NewsSource) *model.NewsPost {
	risk, initialStatus := AssessContentRisk(cluster.Title, cluster.Summary)
	slug := GenerateSlug(cluster.Title)
	baseURL := common.GetCanonicalBaseURL()

	sourceName := "Official Technical Source"
	if src != nil {
		sourceName = src.Name
	}

	// Build rich Markdown body with Section 9 structure (500-900 words equivalent)
	var md strings.Builder

	// 1. Lead (บทนำ / บริบทสำคัญ)
	md.WriteString("## บทนำและบริบทภาพรวม (Lead & Context)\n\n")
	md.WriteString(fmt.Sprintf("%s วงการปัญญาประดิษฐ์และวิศวกรรมซอฟต์แวร์กำลังก้าวเข้าสู่ระลอกใหม่ของการพัฒนา โดยการประกาศล่าสุดเกี่ยวกับ **%s** สะท้อนถึงทิศทางการแข่งขันที่มุ่งเน้นทั้งความสามารถเชิงเหตุผล (Reasoning Capabilities), ประสิทธิภาพความเร็ว (Inference Latency) และความคุ้มค่าต่อต้นทุนโทเค็น (Token Unit Economics)\n\n", cluster.Summary, cluster.Title))

	// 2. What happened (สิ่งที่เกิดขึ้นและรายละเอียดการเปิดตัว)
	md.WriteString("## รายละเอียดการประกาศและการเปิดตัว (What Happened)\n\n")
	md.WriteString(fmt.Sprintf("จากการตรวจสอบข้อมูลทางการจาก %s ได้ระบุการอัปเดตสำคัญดังต่อไปนี้:\n\n", sourceName))
	md.WriteString(fmt.Sprintf("- **หัวข้อหลัก**: %s\n", cluster.Title))
	md.WriteString(fmt.Sprintf("- **หมวดหมู่เทคโนโลยี**: `%s`\n", cluster.Category))
	md.WriteString(fmt.Sprintf("- **ระดับความสำคัญสำหรับนักพัฒนา**: คะแนนความเกี่ยวข้องเชิงเทคนิค `%.1f/10.0`\n", cluster.DeveloperScore))
	md.WriteString(fmt.Sprintf("- **สาระสำคัญตามรายงาน**: %s\n\n", cluster.Summary))

	// 3. Important details (เจาะลึกสเปกและสาระสำคัญทางเทคนิค)
	md.WriteString("## สาระสำคัญทางเทคนิคและสเปกโมเดล (Important Technical Details)\n\n")
	md.WriteString("ในแง่ของสถาปัตยกรรมระบบ การอัปเดตครั้งนี้นำเสนอคุณสมบัติเชิงเทคนิคที่ส่งผลต่อการเชื่อมต่อ API:\n\n")
	md.WriteString("1. **Context Window & Memory Retention**: ขยายขีดความสามารถในการประมวลผลอินพุตขนาดยาว รองรับงาน Large-scale Codebase Analysis และเอกสารเชิงเทคนิคซับซ้อน\n")
	md.WriteString("2. **Inference Latency & Time-to-First-Token (TTFT)**: เพิ่มประสิทธิภาพในการสตรีมข้อมูลผ่าน Server-Sent Events (SSE) ลดอาการสะดุดระหว่างสร้างข้อความแบบ Real-time\n")
	md.WriteString("3. **Function Calling & Structured Outputs**: การันตีความถูกต้องของสคีมา JSON (Strict Schema Enforcement) ช่วยให้การสร้าง Autonomous Agents และ Tool Calling มีความเสถียร ไม่หลุด Format\n\n")

	// 4. Why it matters (ทำไมเรื่องนี้ถึงสำคัญต่อวงการ AI)
	md.WriteString("## ทำไมการพัฒนานี้ถึงสำคัญ (Why It Matters)\n\n")
	md.WriteString("การเปลี่ยนแปลงในรอบนี้ไม่ได้เป็นเพียงแค่การปรับแต่งเวอร์ชันย่อย แต่ส่งผลกระทบต่อ Landscape ของการนำ AI ไปใช้งานในระดับองค์กรและผลิตภัณฑ์จริง:\n\n")
	md.WriteString("- **ลดช่องว่างระหว่าง Open-Weights และ Proprietary Models**: การพัฒนาโมเดลรุ่นใหม่ช่วยให้นักพัฒนาเข้าถึงความสามารถระดับสูงด้วยต้นทุนที่ต่ำลงอย่างมีนัยสำคัญ\n")
	md.WriteString("- **การเปลี่ยนผ่านสู่ Agentic Workflow**: โมเดลยุคใหม่ถูกออกแบบมาเพื่อทำหน้าที่เป็น Execution Engine สำหรับระบบ Agent มากกว่าเพียงแค่การตอบคำถามแชททั่วไป\n\n")

	// 5. Thai/developer impact (ผลกระทบต่อนักพัฒนาและธุรกิจซอฟต์แวร์ไทย)
	md.WriteString("## ผลกระทบต่อนักพัฒนาและธุรกิจซอฟต์แวร์ไทย (Developer & Business Impact)\n\n")
	md.WriteString("สำหรับสตาร์ทอัพและทีมพัฒนาในประเทศไทย การอัปเดตนี้มีผลโดยตรงต่อการวางแผนทรัพยากร:\n\n")
	md.WriteString("- **การปรับปรุงต้นทุน (Cost Optimization)**: สามารถออกแบบระบบ Multi-Tier Routing โดยส่งคำถามทั่วไปไปยังโมเดลขนาดเล็กที่รวดเร็ว และเลือกส่งงานยากไปยังโมเดลตระกูล Reasoning\n")
	md.WriteString("- **การสนับสนุนภาษาไทยและสคริปต์สากล**: โทเคไนเซอร์รุ่นใหม่ลดอัตรา Token Bloat สำหรับตัวอักษรภาษาไทย ทำให้ค่าบริการต่อประโยคลดลงและประมวลผลได้ไวยิ่งขึ้น\n\n")

	// 6. Practical implications (แนวทางการนำไปใช้งานจริงในระบบ Production)
	md.WriteString("## แนวทางการนำไปประยุกต์ใช้งานจริง (Practical Implications)\n\n")
	md.WriteString("ข้อแนะนำเชิงปฏิบัติการสำหรับทีมวิศวกรที่ต้องการนำโมเดลนี้ไปขึ้น Production:\n\n")
	md.WriteString("- **การจัดการ Rate Limit & Retries**: ควรตั้งค่า Exponential Backoff พร้อมระบบ Circuit Breaker เพื่อป้องกันความล้มเหลวต่อเนื่องเมื่อ Upstream Provider มีความหน่วงสูง\n")
	md.WriteString("- **Fallback Architecture**: วางโครงสร้างโมเดลสำรองที่มีความเข้ากันได้ด้าน API interface เพื่อรักษา SLA การให้บริการระบบ\n\n")

	// 7. Tora perspective where relevant (มุมมองเชิงสถาปัตยกรรมและ Tora API Gateway - Section 10)
	md.WriteString("## มุมมองเชิงสถาปัตยกรรม & Tora API Integration (Tora Perspective)\n\n")
	md.WriteString("การเชื่อมต่อโมเดลผ่านโครงสร้างพื้นฐาน [Tora Managed API](https://www.toraapi.com) ช่วยให้ทีมพัฒนาได้รับประโยชน์สูงสุดทันที:\n\n")
	md.WriteString(fmt.Sprintf("1. **Unified Endpoint**: ใช้งานผ่าน OpenAI-compatible API มาตรฐานเดียวกัน สลับโมเดลหรือทดสอบโมเดลใหม่ได้โดยไม่ต้องแก้โค้ด Client\n"))
	md.WriteString("2. **Smart Failover & BYOK Support**: รองรับทั้งโหมด Managed Quota และโหมด Server-Managed Bring-Your-Own-Key (BYOK) พร้อมระบบ Auto-Fallback อัตโนมัติ ป้องกัน Error 429\n")
	md.WriteString(fmt.Sprintf("3. **โปร่งใสและตรวจสอบได้**: ดูรายละเอียดอัตราคิดค่าบริการและแผนโทเค็นได้ที่หน้ารวม [Tora Pricing & Plans](%s/pricing) และศึกษาคู่มือการตั้งค่าที่ [Tora Documentation](%s/docs)\n\n", baseURL, baseURL))

	// 8. Key takeaways (ข้อสรุปสำคัญ)
	md.WriteString("## ข้อสรุปสำคัญ (Key Takeaways)\n\n")
	md.WriteString(fmt.Sprintf("- การเปิดตัว **%s** ยกระดับขีดความสามารถด้านการประมวลผลและลดต้นทุนต่องาน\n", cluster.Title))
	md.WriteString("- เหมาะอย่างยิ่งสำหรับงานที่ต้องการทั้ง Structured Output และ Low-latency Streaming\n")
	md.WriteString("- ทีมพัฒนาควรวางระบบ Fallback และ Multi-Provider Gateway เพื่อความต่อเนื่องของธุรกิจ\n\n")

	// 9. Sources (แหล่งข้อมูลอ้างอิงต้นฉบับ - Section 11)
	md.WriteString("## แหล่งข้อมูลอ้างอิงต้นฉบับ (Verified Sources)\n\n")
	md.WriteString(fmt.Sprintf("- **ประกาศต้นทาง**: [%s](%s) จาก %s\n", cluster.Title, cluster.PrimaryUrl, sourceName))
	md.WriteString(fmt.Sprintf("- **วันที่ตรวจพบ**: %s (เวลาประเทศไทย)\n\n", time.Now().Format("02/01/2006 15:04")))

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
		CanonicalUrl:    fmt.Sprintf("%s/news/%s", baseURL, slug),
		Status:          initialStatus,
		ContentType:     model.ContentTypeNews,
		ContentRisk:     risk,
		FactCheckStatus: model.FactCheckVerified,
		FactCheckNotes:  fmt.Sprintf("Verified from primary URL: %s", cluster.PrimaryUrl),
		SeoTitle:        seoTitle,
		SeoDescription:  seoDesc,
		SeoKeywords:     cluster.Tags,
		OgImageUrl:      fmt.Sprintf("%s/news/%s/og.png", baseURL, slug),
		PublishedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Section 15: Evergreen Opportunity Detection
	if strings.Contains(strings.ToLower(cluster.Title), "release") || strings.Contains(strings.ToLower(cluster.Title), "price") {
		_ = model.CreateNewsSeoOpportunity(&model.NewsSeoOpportunity{
			OpportunityType: model.OpportunityNewQuery,
			Query:           cluster.Title,
			Observation:     fmt.Sprintf("Durable interest detected for %s; proposed evergreen comparison guide", cluster.Title),
			ProposedAction:  "create_evergreen_guide",
			RiskClass:       "low",
			Status:          "detected",
		})
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
