package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// GenerateNewsArticleJSONLD creates valid Schema.org Article structured data matching content type
func GenerateNewsArticleJSONLD(post *model.NewsPost) string {
	pubDate := time.Unix(post.PublishedAt, 0).UTC().Format(time.RFC3339)
	modDate := time.Unix(post.UpdatedAt, 0).UTC().Format(time.RFC3339)

	articleType := "NewsArticle"
	if post.ContentType == model.ContentTypeGuide {
		articleType = "TechArticle"
	} else if post.ContentType == model.ContentTypeAnalysis {
		articleType = "AnalysisNewsArticle"
	} else if post.ContentType == model.ContentTypeChangelog {
		articleType = "Article"
	}

	canonicalBase := common.GetCanonicalBaseURL()
	canonicalUrl := fmt.Sprintf("%s/news/%s", canonicalBase, post.Slug)
	ogImageUrl := fmt.Sprintf("%s/news/%s/og.png", canonicalBase, post.Slug)
	data := map[string]any{
		"@context":         "https://schema.org",
		"@type":            articleType,
		"mainEntityOfPage": canonicalUrl,
		"headline":         post.Title,
		"description":      post.Summary,
		"datePublished":    pubDate,
		"dateModified":     modDate,
		"image":            []string{ogImageUrl},
		"author": map[string]any{
			"@type": "Organization",
			"name":  post.AuthorName,
			"url":   canonicalBase,
		},
		"publisher": map[string]any{
			"@type": "Organization",
			"name":  "Tora AI",
			"url":   canonicalBase,
			"logo": map[string]any{
				"@type": "ImageObject",
				"url":   canonicalBase + "/assets/logo.png",
			},
		},
	}

	bytes, _ := json.Marshal(data)
	return string(bytes)
}

// GenerateBreadcrumbJSONLD creates valid Schema.org BreadcrumbList structured data
func GenerateBreadcrumbJSONLD(post *model.NewsPost) string {
	canonicalBase := common.GetCanonicalBaseURL()
	data := map[string]any{
		"@context": "https://schema.org",
		"@type":    "BreadcrumbList",
		"itemListElement": []map[string]any{
			{
				"@type":    "ListItem",
				"position": 1,
				"name":     "Home",
				"item":     canonicalBase,
			},
			{
				"@type":    "ListItem",
				"position": 2,
				"name":     "News",
				"item":     canonicalBase + "/news",
			},
			{
				"@type":    "ListItem",
				"position": 3,
				"name":     post.Title,
				"item":     canonicalBase + "/news/" + post.Slug,
			},
		},
	}

	bytes, _ := json.Marshal(data)
	return string(bytes)
}

// GenerateSitemapXML outputs standard XML sitemap for Google and search crawlers
func GenerateSitemapXML(posts []*model.NewsPost) string {
	canonicalBase := common.GetCanonicalBaseURL()
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")

	// Core root pages
	corePages := []struct {
		url      string
		priority string
		freq     string
	}{
		{canonicalBase + "/", "1.0", "daily"},
		{canonicalBase + "/news", "0.9", "hourly"},
		{canonicalBase + "/pricing", "0.8", "weekly"},
		{canonicalBase + "/docs", "0.8", "weekly"},
	}

	for _, cp := range corePages {
		sb.WriteString("  <url>\n")
		sb.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", cp.url))
		sb.WriteString(fmt.Sprintf("    <changefreq>%s</changefreq>\n", cp.freq))
		sb.WriteString(fmt.Sprintf("    <priority>%s</priority>\n", cp.priority))
		sb.WriteString("  </url>\n")
	}

	for _, p := range posts {
		if p.Status != model.NewsStatusPublished {
			continue
		}
		modTime := time.Unix(p.UpdatedAt, 0).UTC().Format("2006-01-02")
		sb.WriteString("  <url>\n")
		sb.WriteString(fmt.Sprintf("    <loc>%s/news/%s</loc>\n", canonicalBase, p.Slug))
		sb.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", modTime))
		sb.WriteString("    <changefreq>weekly</changefreq>\n")
		sb.WriteString("    <priority>0.7</priority>\n")
		sb.WriteString("  </url>\n")
	}

	sb.WriteString("</urlset>")
	return sb.String()
}

// GenerateNewsSitemapXML outputs specialized Google News sitemap (<news:news>) using current time
func GenerateNewsSitemapXML(posts []*model.NewsPost) string {
	return GenerateNewsSitemapXMLWithTime(posts, time.Now())
}

// GenerateNewsSitemapXMLWithTime outputs Google News sitemap with explicit reference time for boundary testing
func GenerateNewsSitemapXMLWithTime(posts []*model.NewsPost, refTime time.Time) string {
	canonicalBase := common.GetCanonicalBaseURL()
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"` + "\n")
	sb.WriteString(`        xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">` + "\n")

	refUnix := refTime.Unix()
	cutoff48h := refUnix - (48 * 3600)

	for _, p := range posts {
		// Strict Google News qualification:
		// 1. Must be published
		// 2. ContentType must be "news" (guides, changelogs, evergreen analyses are excluded)
		// 3. Must not be future-dated (p.PublishedAt <= refUnix)
		// 4. Must be within 48-hour freshness window (p.PublishedAt >= cutoff48h)
		// 5. Must NOT be seed content (p.IsSeed must be false) - bootstrap content excluded from Google News
		isNews := p.ContentType == model.ContentTypeNews || p.ContentType == ""
		if p.Status != model.NewsStatusPublished || !isNews || p.PublishedAt > refUnix || p.PublishedAt < cutoff48h || p.IsSeed {
			continue
		}

		pubDate := time.Unix(p.PublishedAt, 0).UTC().Format(time.RFC3339)
		sb.WriteString("  <url>\n")
		sb.WriteString(fmt.Sprintf("    <loc>%s/news/%s</loc>\n", canonicalBase, p.Slug))
		sb.WriteString("    <news:news>\n")
		sb.WriteString("      <news:publication>\n")
		sb.WriteString("        <news:name>Tora AI News</news:name>\n")
		sb.WriteString("        <news:language>th</news:language>\n")
		sb.WriteString("      </news:publication>\n")
		sb.WriteString(fmt.Sprintf("      <news:publication_date>%s</news:publication_date>\n", pubDate))
		sb.WriteString(fmt.Sprintf("      <news:title><![CDATA[%s]]></news:title>\n", p.Title))
		sb.WriteString("    </news:news>\n")
		sb.WriteString("  </url>\n")
	}

	sb.WriteString("</urlset>")
	return sb.String()
}

// GenerateRobotsTXT outputs search engine directives
func GenerateRobotsTXT() string {
	canonicalBase := common.GetCanonicalBaseURL()
	return fmt.Sprintf(`User-agent: *
Allow: /
Allow: /news
Allow: /news/*
Disallow: /api/
Disallow: /v1/
Disallow: /admin/
Disallow: /news-admin/
Disallow: /news-admin

Sitemap: %s/sitemap.xml
Sitemap: %s/news-sitemap.xml
`, canonicalBase, canonicalBase)
}
