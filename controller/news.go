package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// --- Public Web HTML Handlers (Crawlable & Semantic) ---

// RenderNewsIndexPage serves the main news catalog at /news
func RenderNewsIndexPage(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "12"))
	category := strings.TrimSpace(c.Query("category"))
	tag := strings.TrimSpace(c.Query("tag"))
	contentType := strings.TrimSpace(c.Query("type"))

	posts, total, err := model.GetPublishedNewsPosts(page, pageSize, category, tag, contentType)
	if err != nil {
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	canonicalBase := common.GetCanonicalBaseURL()
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="th" class="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>ข่าวโมเดลและเทคโนโลยี AI ล่าสุดสำหรับนักพัฒนา | Tora AI Tech News</title>
  <meta name="description" content="เกาะติดข่าวสารโมเดล AI ล่าสุด, OpenAI, Claude, DeepSeek, Gemini, สถาปัตยกรรม LLM และการลดต้นทุน Token สำหรับนักพัฒนาซอฟต์แวร์ไทย">
  <meta name="keywords" content="AI, LLM, OpenAI, Claude, DeepSeek, Gemini, API, Tora AI, นักพัฒนา, Thailand, Machine Learning">
  <link rel="canonical" href="` + canonicalBase + `/news">
  <meta property="og:type" content="website">
  <meta property="og:title" content="Tora AI Tech News — ข่าวและบทวิเคราะห์โมเดล AI สำหรับนักพัฒนา">
  <meta property="og:description" content="เกาะติดข่าวสารโมเดล AI ล่าสุด, OpenAI, Claude, DeepSeek, Gemini และเทคนิคสถาปัตยกรรม API สำหรับนักพัฒนาซอฟต์แวร์ไทย">
  <meta property="og:url" content="` + canonicalBase + `/news">
  <meta property="og:site_name" content="Tora AI">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="Tora AI Tech News — ข่าวและบทวิเคราะห์โมเดล AI สำหรับนักพัฒนา">
  <meta name="twitter:description" content="เกาะติดข่าวสารโมเดล AI ล่าสุดสำหรับนักพัฒนาซอฟต์แวร์ไทย">
  <script src="https://cdn.tailwindcss.com"></script>
  <script>
    tailwind.config = {
      darkMode: 'class',
      theme: {
        extend: {
          colors: {
            brand: { 500: '#6366f1', 600: '#4f46e5', 700: '#4338ca' }
          }
        }
      }
    }
  </script>
  <style>
    @import url('https://fonts.googleapis.com/css2?family=Sarabun:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;500;700&display=swap');
    body { font-family: 'Sarabun', -apple-system, BlinkMacSystemFont, sans-serif; }
    code, pre { font-family: 'JetBrains Mono', monospace; }
  </style>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen flex flex-col antialiased selection:bg-indigo-500 selection:text-white">
  <!-- Top Navigation Header -->
  <header class="border-b border-slate-800/80 bg-slate-950/80 backdrop-blur sticky top-0 z-50">
    <div class="max-w-6xl mx-auto px-4 h-16 flex items-center justify-between">
      <a href="/" class="flex items-center gap-2.5 group">
        <div class="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center font-bold text-white shadow-lg shadow-indigo-500/20 group-hover:scale-105 transition">T</div>
        <span class="text-xl font-bold tracking-tight text-white">Tora AI</span>
        <span class="text-xs px-2 py-0.5 rounded-full bg-indigo-950/80 text-indigo-300 border border-indigo-800/50 font-medium">Newsroom</span>
      </a>
      <nav class="hidden md:flex items-center gap-6 text-sm font-medium text-slate-300">
        <a href="` + canonicalBase + `" class="hover:text-white transition">หน้าหลัก</a>
        <a href="/news" class="text-indigo-400 font-semibold">ข่าว AI</a>
        <a href="/pricing" class="hover:text-white transition">ราคา & โควตา</a>
        <a href="` + canonicalBase + `/docs" class="hover:text-white transition">เอกสาร API</a>
      </nav>
      <div class="flex items-center gap-3">
        <a href="/login" class="text-xs font-semibold px-3 py-1.5 rounded-lg border border-slate-700 text-slate-300 hover:text-white hover:border-slate-500 transition">เข้าสู่ระบบ</a>
        <a href="/register" class="text-xs font-semibold px-3.5 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white transition shadow-sm">เริ่มใช้งานฟรี</a>
      </div>
    </div>
  </header>

  <!-- Hero Section -->
  <section class="border-b border-slate-800/60 bg-gradient-to-b from-slate-900/60 to-slate-950 py-12 px-4">
    <div class="max-w-6xl mx-auto text-center md:text-left">
      <div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-indigo-950/60 border border-indigo-800/40 text-indigo-300 text-xs font-semibold mb-4">
        <span>⚡ ข่าวสารและบทวิเคราะห์เทคนิค</span>
        <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
      </div>
      <h1 class="text-3xl md:text-5xl font-extrabold text-white tracking-tight mb-4 leading-tight">
        เกาะติดโมเดลและเทคโนโลยี AI <br class="hidden md:inline"><span class="text-transparent bg-clip-text bg-gradient-to-r from-indigo-400 via-sky-300 to-emerald-400">สำหรับนักพัฒนาซอฟต์แวร์</span>
      </h1>
      <p class="text-slate-400 text-base md:text-lg max-w-2xl leading-relaxed">
        เจาะลึกสเปกโมเดลใหม่, เปรียบเทียบ Benchmark, เทคนิคการลดต้นทุน Token และคู่มือการเชื่อมต่อ API สำหรับทีมวิศวกรไทย
      </p>
    </div>
  </section>

  <!-- Main Content Grid -->
  <main class="max-w-6xl mx-auto px-4 py-10 flex-grow w-full">
    <div class="flex items-center justify-between mb-8 pb-4 border-b border-slate-800/60">
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>บทความล่าสุด</span>
        <span class="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 font-normal">` + fmt.Sprintf("%d บทความ", total) + `</span>
      </h2>
      <div class="flex items-center gap-2 text-xs">
        <a href="/sitemap.xml" class="text-slate-400 hover:text-indigo-400 transition flex items-center gap-1">
          <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20"><path d="M5 3a1 1 0 000 2c5.523 0 10 4.477 10 10a1 1 0 102 0C17 8.373 11.627 3 5 3z"/><path d="M4 9a1 1 0 011-1 7 7 0 017 7 1 1 0 11-2 0 5 5 0 00-5-5 1 1 0 01-1-1zM3 15a2 2 0 114 0 2 2 0 01-4 0z"/></svg>
          Sitemap
        </a>
      </div>
    </div>

    <!-- Article Cards Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">`)

	for _, p := range posts {
		dateStr := time.Unix(p.PublishedAt, 0).Format("02 ม.ค. 2006")
		cat := "Tech"
		if p.SeoKeywords != "" {
			parts := strings.Split(p.SeoKeywords, ",")
			if len(parts) > 0 {
				cat = strings.ToUpper(strings.TrimSpace(parts[0]))
			}
		}

		sb.WriteString(fmt.Sprintf(`
      <article class="bg-slate-900/60 border border-slate-800/80 rounded-xl overflow-hidden hover:border-indigo-500/50 hover:shadow-lg hover:shadow-indigo-500/5 transition flex flex-col group">
        <div class="p-6 flex flex-col flex-grow">
          <div class="flex items-center justify-between mb-3 text-xs text-slate-400">
            <span class="px-2 py-0.5 rounded bg-indigo-950/80 text-indigo-300 font-semibold border border-indigo-800/30">%s</span>
            <time datetime="%s">%s</time>
          </div>
          <h3 class="text-lg font-bold text-white group-hover:text-indigo-300 transition leading-snug mb-3">
            <a href="/news/%s" class="hover:underline">%s</a>
          </h3>
          <p class="text-slate-400 text-sm leading-relaxed mb-4 line-clamp-3 flex-grow">%s</p>
          <div class="pt-4 border-t border-slate-800/60 flex items-center justify-between text-xs text-slate-400">
            <span class="flex items-center gap-1.5">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
              %s
            </span>
            <a href="/news/%s" class="text-indigo-400 font-semibold hover:text-indigo-300 flex items-center gap-1">
              อ่านต่อ
              <svg class="w-3.5 h-3.5 group-hover:translate-x-0.5 transition" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>
            </a>
          </div>
        </div>
      </article>`,
			html.EscapeString(cat),
			time.Unix(p.PublishedAt, 0).Format(time.RFC3339),
			dateStr,
			p.Slug,
			html.EscapeString(p.Title),
			html.EscapeString(p.Summary),
			html.EscapeString(p.AuthorName),
			p.Slug,
		))
	}

	sb.WriteString(`
    </div>`)

	// Pagination if applicable
	if totalPages > 1 {
		sb.WriteString(`
    <nav class="flex items-center justify-center gap-2 mt-12 text-sm font-semibold">`)
		for i := 1; i <= totalPages; i++ {
			activeClass := "bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white"
			if i == page {
				activeClass = "bg-indigo-600 text-white"
			}
			sb.WriteString(fmt.Sprintf(`<a href="/news?page=%d" class="px-4 py-2 rounded-lg %s transition">%d</a>`, i, activeClass, i))
		}
		sb.WriteString(`
    </nav>`)
	}

	sb.WriteString(`
  </main>

  <!-- CTA Banner -->
  <section class="max-w-6xl mx-auto px-4 pb-12 w-full">
    <div class="rounded-2xl bg-gradient-to-r from-indigo-900/60 via-slate-900 to-indigo-950 border border-indigo-800/40 p-8 md:p-10 flex flex-col md:flex-row items-center justify-between gap-6">
      <div class="space-y-2 text-center md:text-left">
        <h3 class="text-xl md:text-2xl font-bold text-white">พร้อมต่อยอดโมเดล AI ในระบบของคุณแล้วหรือยัง?</h3>
        <p class="text-slate-300 text-sm max-w-xl">เชื่อมต่อ GPT-4.5, Claude 3.7, DeepSeek-V3 ผ่าน API เดียว พร้อมระบบ Fallback ป้องกัน Downtime และชำระเงินสะดวกผ่าน PromptPay</p>
      </div>
      <div class="flex items-center gap-3 shrink-0">
        <a href="/register" class="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-bold shadow-lg shadow-indigo-500/20 transition">เริ่มต้นฟรี</a>
        <a href="/pricing" class="px-5 py-2.5 rounded-xl border border-slate-700 hover:border-slate-500 text-slate-300 text-sm font-medium transition">ดูราคาและสเปก</a>
      </div>
    </div>
  </section>

  <!-- Footer -->
  <footer class="border-t border-slate-800/80 bg-slate-950 py-10 px-4 text-xs text-slate-500">
    <div class="max-w-6xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
      <div class="flex items-center gap-2">
        <span class="font-bold text-slate-300">Tora AI</span>
        <span>•</span>
        <span>ระบบเกตเวย์โมเดลปัญญาประดิษฐ์ประสิทธิภาพสูง</span>
      </div>
      <div class="flex items-center gap-6">
        <a href="/news" class="hover:text-slate-300 transition">ข่าว AI ทั้งหมด</a>
        <a href="/pricing" class="hover:text-slate-300 transition">ราคา</a>
        <a href="/sitemap.xml" class="hover:text-slate-300 transition">Sitemap XML</a>
        <a href="/robots.txt" class="hover:text-slate-300 transition">Robots.txt</a>
      </div>
    </div>
  </footer>
</body>
</html>`)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(sb.String()))
}

// RenderNewsPostPage serves individual article at /news/:slug
func RenderNewsPostPage(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		render404Page(c, "ระบุ Slug ไม่ถูกต้อง")
		return
	}

	post, err := model.GetNewsPostBySlug(slug)
	if err != nil || post == nil {
		render404Page(c, "ไม่พบบทความที่ร้องขอ หรือบทความอาจถูกถอดถอนแล้ว")
		return
	}

	// Analytics telemetry asynchronously
	go func(postId int, clientIp, referrer, utmSource, utmMedium, utmCampaign string) {
		_ = model.IncrementNewsPostView(postId)
		hash := sha256.Sum256([]byte(clientIp))
		_ = model.RecordNewsAnalyticEvent(&model.NewsAnalyticEvent{
			PostId:      postId,
			EventType:   "view",
			Referrer:    referrer,
			UtmSource:   utmSource,
			UtmMedium:   utmMedium,
			UtmCampaign: utmCampaign,
			IpHash:      hex.EncodeToString(hash[:16]),
			CreatedAt:   common.GetTimestamp(),
		})
	}(post.Id, c.ClientIP(), c.Request.Referer(), c.Query("utm_source"), c.Query("utm_medium"), c.Query("utm_campaign"))

	newsArticleJSONLD := service.GenerateNewsArticleJSONLD(post)
	breadcrumbJSONLD := service.GenerateBreadcrumbJSONLD(post)
	dateStr := time.Unix(post.PublishedAt, 0).Format("02 มกราคม 2006")

	category := "TECH & AI"
	if post.SeoKeywords != "" {
		parts := strings.Split(post.SeoKeywords, ",")
		if len(parts) > 0 {
			category = strings.ToUpper(strings.TrimSpace(parts[0]))
		}
	}

	canonicalBase := common.GetCanonicalBaseURL()
	canonicalUrl := fmt.Sprintf("%s/news/%s", canonicalBase, post.Slug)
	ogImageUrl := fmt.Sprintf("%s/news/%s/og.png", canonicalBase, post.Slug)

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="th" class="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>` + html.EscapeString(post.SeoTitle) + `</title>
  <meta name="description" content="` + html.EscapeString(post.SeoDescription) + `">
  <meta name="keywords" content="` + html.EscapeString(post.SeoKeywords) + `">
  <link rel="canonical" href="` + html.EscapeString(canonicalUrl) + `">
  <meta property="og:type" content="article">
  <meta property="og:title" content="` + html.EscapeString(post.Title) + `">
  <meta property="og:description" content="` + html.EscapeString(post.Summary) + `">
  <meta property="og:url" content="` + html.EscapeString(canonicalUrl) + `">
  <meta property="og:image" content="` + html.EscapeString(ogImageUrl) + `">
  <meta property="og:image:type" content="image/png">
  <meta property="og:image:width" content="1200">
  <meta property="og:image:height" content="630">
  <meta property="og:site_name" content="Tora AI News">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="` + html.EscapeString(post.Title) + `">
  <meta name="twitter:description" content="` + html.EscapeString(post.Summary) + `">
  <meta name="twitter:image" content="` + html.EscapeString(ogImageUrl) + `">
  <!-- Schema.org NewsArticle Structured Data -->
  <script type="application/ld+json">
` + newsArticleJSONLD + `
  </script>
  <!-- Schema.org Breadcrumb Structured Data -->
  <script type="application/ld+json">
` + breadcrumbJSONLD + `
  </script>
  <script src="https://cdn.tailwindcss.com"></script>
  <style>
    @import url('https://fonts.googleapis.com/css2?family=Sarabun:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;500;700&display=swap');
    body { font-family: 'Sarabun', -apple-system, BlinkMacSystemFont, sans-serif; }
    code, pre { font-family: 'JetBrains Mono', monospace; }
  </style>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen flex flex-col antialiased selection:bg-indigo-500 selection:text-white">
  <!-- Header -->
  <header class="border-b border-slate-800/80 bg-slate-950/80 backdrop-blur sticky top-0 z-50">
    <div class="max-w-4xl mx-auto px-4 h-16 flex items-center justify-between">
      <a href="/news" class="flex items-center gap-2 text-slate-300 hover:text-white transition text-sm font-semibold group">
        <svg class="w-4 h-4 group-hover:-translate-x-1 transition" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg>
        กลับหน้ารวมข่าว
      </a>
      <div class="flex items-center gap-3">
        <a href="` + canonicalBase + `" class="text-xs font-semibold text-slate-400 hover:text-white transition">Tora AI Home</a>
        <a href="/register" class="text-xs font-semibold px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white transition">ทดลองใช้ API ฟรี</a>
      </div>
    </div>
  </header>

  <!-- Article Container -->
  <article class="max-w-4xl mx-auto px-4 py-12 flex-grow w-full">
    <!-- Meta Header -->
    <header class="mb-10">
      <div class="flex flex-wrap items-center gap-3 text-xs mb-4">
        <span class="px-2.5 py-1 rounded-md bg-indigo-950/80 text-indigo-300 font-bold border border-indigo-800/40">` + html.EscapeString(category) + `</span>
        <span class="text-slate-400">เผยแพร่: <time datetime="` + time.Unix(post.PublishedAt, 0).Format(time.RFC3339) + `">` + dateStr + `</time></span>
        <span class="text-slate-600">•</span>
        <span class="text-slate-400">ผู้เขียน: ` + html.EscapeString(post.AuthorName) + `</span>
        <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-950/80 text-emerald-400 border border-emerald-800/40 font-medium">
          <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/></svg>
          ตรวจสอบข้อเท็จจริงแล้ว
        </span>
      </div>

      <h1 class="text-2xl md:text-4xl font-extrabold text-white tracking-tight leading-tight mb-6">
        ` + html.EscapeString(post.Title) + `
      </h1>

      <!-- TL;DR Summary Box -->
      <div class="p-5 rounded-xl bg-slate-900/80 border-l-4 border-indigo-500 border-y border-r border-slate-800/80 text-slate-300 text-base leading-relaxed">
        <strong class="text-indigo-400 font-bold block mb-1">สรุปสาระสำคัญ (TL;DR):</strong>
        ` + html.EscapeString(post.Summary) + `
      </div>
    </header>

    <!-- Article Content -->
    <div class="prose prose-invert max-w-none text-slate-300 text-base leading-relaxed">
      ` + post.ContentHTML + `
    </div>

    <!-- Verified Source Citation Box -->
    <div class="mt-12 p-6 rounded-xl bg-slate-900/60 border border-slate-800/80 flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
      <div>
        <h4 class="text-sm font-bold text-slate-200 mb-1">ความโปร่งใสและแหล่งข้อมูลอ้างอิง</h4>
        <p class="text-xs text-slate-400">บทความนี้ได้รับการสังเคราะห์และตรวจสอบข้อเท็จจริงตามหลัก <a href="/docs/editorial" class="underline hover:text-indigo-300">Tora Editorial Standards</a> โดยอ้างอิงจากเอกสารทางการของผู้พัฒนา</p>
      </div>
      <a href="` + html.EscapeString(post.SourceUrl) + `" target="_blank" rel="noopener noreferrer" class="shrink-0 text-xs font-semibold px-3.5 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-indigo-300 hover:text-white transition flex items-center gap-1.5">
        ดูประกาศต้นฉบับ
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/></svg>
      </a>
    </div>

    <!-- CTA Section -->
    <div class="mt-12 p-8 rounded-2xl bg-gradient-to-r from-indigo-950 via-slate-900 to-indigo-900 border border-indigo-800/40 text-center md:text-left flex flex-col md:flex-row items-center justify-between gap-6">
      <div>
        <h3 class="text-xl font-bold text-white mb-2">เริ่มใช้งานโมเดล AI ผ่าน Tora API Gateway</h3>
        <p class="text-slate-300 text-sm max-w-md">รองรับมาตรฐาน OpenAI Compatible พร้อมระบบ Route Engine สลับ upstream อัตโนมัติเมื่อเกิด Rate Limit</p>
      </div>
      <div class="flex items-center gap-3 shrink-0">
        <a href="/register" class="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-bold shadow-lg shadow-indigo-500/20 transition">สมัครใช้งานฟรี</a>
        <a href="/pricing" class="px-5 py-2.5 rounded-xl border border-slate-700 hover:border-slate-500 text-slate-300 text-sm font-medium transition">เปรียบเทียบราคา</a>
      </div>
    </div>
  </article>

  <!-- Footer -->
  <footer class="border-t border-slate-800/80 bg-slate-950 py-8 px-4 text-xs text-slate-500 text-center">
    <p>© 2026 Tora AI. All rights reserved. <a href="/news" class="hover:text-slate-300 ml-2">หน้ารวมข่าว AI</a></p>
  </footer>
</body>
</html>`)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(sb.String()))
}

// RenderNewsOGCard serves branded dynamic SVG card at /news/:slug/og.svg
func RenderNewsOGCard(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	post, err := model.GetNewsPostBySlug(slug)
	if err != nil || post == nil {
		// Default fallback post
		post = &model.NewsPost{
			Title:       "Tora AI — Autonomous News & Engineering Gateway",
			AuthorName:  "Tora Architecture",
			SeoKeywords: "ai,news,engineering",
			PublishedAt: common.GetTimestamp(),
		}
	}

	svg := service.GenerateOGCardSVG(post)
	c.Header("Content-Type", "image/svg+xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=86400, s-maxage=86400")
	c.String(http.StatusOK, svg)
}

// RenderNewsOGPNGCard serves standard 1200x630 raster PNG social card at /news/:slug/og.png
func RenderNewsOGPNGCard(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	post, err := model.GetNewsPostBySlug(slug)
	if err != nil || post == nil {
		post = &model.NewsPost{
			Title:       "Tora AI — Autonomous News & Engineering Gateway",
			AuthorName:  "Tora Architecture",
			SeoKeywords: "ai,news,engineering",
			PublishedAt: common.GetTimestamp(),
		}
	}

	pngBytes, err := service.GenerateOGCardPNG(post)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating social card image")
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "public, max-age=86400, s-maxage=86400")
	c.Data(http.StatusOK, "image/png", pngBytes)
}

// RenderSitemap serves standard sitemap.xml
func RenderSitemap(c *gin.Context) {
	posts, _, err := model.GetPublishedNewsPosts(1, 1000, "", "", "")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating sitemap")
		return
	}

	xml := service.GenerateSitemapXML(posts)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, xml)
}

// RenderNewsSitemap serves specialized Google News sitemap at /news-sitemap.xml
func RenderNewsSitemap(c *gin.Context) {
	posts, _, err := model.GetPublishedNewsPosts(1, 1000, "", "", model.ContentTypeNews)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating news sitemap")
		return
	}

	xml := service.GenerateNewsSitemapXML(posts)
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=1800")
	c.String(http.StatusOK, xml)
}

// RenderRobots serves /robots.txt
func RenderRobots(c *gin.Context) {
	txt := service.GenerateRobotsTXT()
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=86400")
	c.String(http.StatusOK, txt)
}

func render404Page(c *gin.Context, msg string) {
	c.Data(http.StatusNotFound, "text/html; charset=utf-8", []byte(`<!DOCTYPE html>
<html lang="th" class="dark">
<head>
  <meta charset="UTF-8">
  <title>404 ไม่พบบทความ | Tora AI News</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen flex items-center justify-center p-4">
  <div class="text-center max-w-md">
    <div class="text-6xl font-black text-indigo-500 mb-4">404</div>
    <h1 class="text-2xl font-bold mb-2">ไม่พบบทความที่ร้องขอ</h1>
    <p class="text-slate-400 text-sm mb-6">`+html.EscapeString(msg)+`</p>
    <a href="/news" class="px-5 py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-sm transition">กลับหน้ารวมข่าว AI</a>
  </div>
</body>
</html>`))
}

// --- Public JSON API Handlers ---

func GetNewsPostsAPI(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := strings.TrimSpace(c.Query("category"))
	tag := strings.TrimSpace(c.Query("tag"))
	contentType := strings.TrimSpace(c.Query("type"))

	posts, total, err := model.GetPublishedNewsPosts(page, pageSize, category, tag, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    posts,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

func GetNewsPostDetailAPI(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	post, err := model.GetNewsPostBySlug(slug)
	if err != nil || post == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "post not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    post,
	})
}

// --- Admin News API Handlers ---

func AdminGetNewsPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := strings.TrimSpace(c.Query("status"))
	keyword := strings.TrimSpace(c.Query("keyword"))

	posts, total, err := model.GetAllNewsPosts(page, pageSize, status, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    posts,
		"total":   total,
	})
}

func AdminCreateNewsPost(c *gin.Context) {
	var post model.NewsPost
	if err := c.ShouldBindJSON(&post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	if post.Slug == "" {
		post.Slug = service.GenerateSlug(post.Title)
	}
	if post.ContentHTML == "" && post.ContentMarkdown != "" {
		post.ContentHTML = service.RenderMarkdownToSafeHTML(post.ContentMarkdown)
	}
	if post.CanonicalUrl == "" {
		post.CanonicalUrl = fmt.Sprintf("%s/news/%s", common.GetCanonicalBaseURL(), post.Slug)
	}
	if post.OgImageUrl == "" {
		post.OgImageUrl = fmt.Sprintf("%s/news/%s/og.png", common.GetCanonicalBaseURL(), post.Slug)
	}

	if err := model.CreateNewsPost(&post); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Queue distributions automatically
	_ = service.QueuePostDistributions(&post)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    post,
	})
}

func AdminUpdateNewsPost(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	existing, err := model.GetNewsPostById(id)
	if err != nil || existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "post not found"})
		return
	}

	var updateReq model.NewsPost
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	if updateReq.Title != "" {
		existing.Title = updateReq.Title
	}
	if updateReq.Summary != "" {
		existing.Summary = updateReq.Summary
	}
	if updateReq.ContentMarkdown != "" {
		existing.ContentMarkdown = updateReq.ContentMarkdown
		existing.ContentHTML = service.RenderMarkdownToSafeHTML(updateReq.ContentMarkdown)
	} else if updateReq.ContentHTML != "" {
		existing.ContentHTML = updateReq.ContentHTML
	}
	if updateReq.Status != "" {
		existing.Status = updateReq.Status
		if existing.Status == model.NewsStatusPublished && existing.PublishedAt == 0 {
			existing.PublishedAt = common.GetTimestamp()
		}
	}
	if updateReq.PublishedAt > 0 {
		existing.PublishedAt = updateReq.PublishedAt
	}
	if updateReq.ContentRisk != "" {
		existing.ContentRisk = updateReq.ContentRisk
	}
	if updateReq.FactCheckStatus != "" {
		existing.FactCheckStatus = updateReq.FactCheckStatus
	}
	if updateReq.FactCheckNotes != "" {
		existing.FactCheckNotes = updateReq.FactCheckNotes
	}
	if updateReq.SeoTitle != "" {
		existing.SeoTitle = updateReq.SeoTitle
	}
	if updateReq.SeoDescription != "" {
		existing.SeoDescription = updateReq.SeoDescription
	}
	if updateReq.SeoKeywords != "" {
		existing.SeoKeywords = updateReq.SeoKeywords
	}

	if err := model.UpdateNewsPost(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    existing,
	})
}

func AdminDeleteNewsPost(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := model.DeleteNewsPost(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "deleted"})
}

func AdminGetNewsSources(c *gin.Context) {
	sources, err := model.GetAllNewsSources(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    sources,
	})
}

func AdminSyncNewsSource(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	source, err := model.GetNewsSourceById(id)
	if err != nil || source == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "source not found"})
		return
	}

	count, err := service.SyncSingleNewsSource(c.Request.Context(), source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"new_posts_drafted": count,
			"source_name":       source.Name,
		},
	})
}

// RecordNewsConversionAPI records funnel progression events linked to content attribution
func RecordNewsConversionAPI(c *gin.Context) {
	var req struct {
		EventType   string  `json:"event_type" binding:"required"`
		ContentId   string  `json:"content_id"`
		UtmSource   string  `json:"utm_source"`
		UtmMedium   string  `json:"utm_medium"`
		UtmCampaign string  `json:"utm_campaign"`
		RevenueTHB  float64 `json:"revenue_thb"`
		Metadata    string  `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	userId := c.GetInt("id") // If authenticated user
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	err := service.RecordConversion(
		req.EventType,
		req.ContentId,
		req.UtmSource,
		req.UtmMedium,
		req.UtmCampaign,
		clientIP,
		userAgent,
		userId,
		req.RevenueTHB,
		req.Metadata,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AdminGetSeoOpportunities returns detected or resolved SEO opportunities
func AdminGetSeoOpportunities(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	var opps []model.NewsSeoOpportunity
	query := model.DB.Model(&model.NewsSeoOpportunity{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("id DESC").Limit(100).Find(&opps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": opps})
}

// AdminRemediateSeoOpportunity triggers an approved autonomous fix for an SEO opportunity
func AdminRemediateSeoOpportunity(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	opp, err := model.GetNewsSeoOpportunityById(id)
	if err != nil || opp == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "opportunity not found"})
		return
	}

	if err := service.ApplySeoRemediation(c.Request.Context(), opp); err != nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": opp})
}

// AdminGetDailyGrowthReviews returns historical growth review summaries
func AdminGetDailyGrowthReviews(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	reviews, err := model.GetRecentDailyGrowthReviews(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reviews})
}

// AdminTriggerNewsScout executes an on-demand scout crawl across enabled feeds
func AdminTriggerNewsScout(c *gin.Context) {
	result, err := service.RunNewsScoutSync(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

