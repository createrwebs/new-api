package service

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
)

func TestBangkokDateRange(t *testing.T) {
	refTime := time.Date(2026, 10, 6, 15, 30, 0, 0, time.UTC)
	start, end, dateStr := model.GetBangkokDateRange(refTime)

	// UTC 15:30 on 2026-10-06 is 22:30 in Bangkok (UTC+7) on 2026-10-06
	if dateStr != "2026-10-06" {
		t.Fatalf("expected dateStr 2026-10-06, got %s", dateStr)
	}
	if end-start != 86400 {
		t.Fatalf("expected 86400 seconds in date range, got %d", end-start)
	}

	// UTC 18:00 on 2026-10-06 is 01:00 in Bangkok on 2026-10-07
	refNextDay := time.Date(2026, 10, 6, 18, 0, 0, 0, time.UTC)
	_, _, dateNextDay := model.GetBangkokDateRange(refNextDay)
	if dateNextDay != "2026-10-07" {
		t.Fatalf("expected dateNextDay 2026-10-07, got %s", dateNextDay)
	}
}

func TestAutopilotCaps(t *testing.T) {
	if MaxPublishedPerDay != 20 {
		t.Fatalf("expected MaxPublishedPerDay 20, got %d", MaxPublishedPerDay)
	}
	if MaxPerBatch != 5 {
		t.Fatalf("expected MaxPerBatch 5, got %d", MaxPerBatch)
	}
	if MaxPerSourcePerDay != 3 {
		t.Fatalf("expected MaxPerSourcePerDay 3, got %d", MaxPerSourcePerDay)
	}
}

func TestEditorialThaiStructure(t *testing.T) {
	cluster := &model.StoryCluster{
		Id:             99,
		Title:          "Claude 3.7 Sonnet Hybrid Reasoning Engine Launch",
		Summary:        "Anthropic เปิดตัวโมเดล Claude 3.7 Sonnet ผสาน Hybrid Reasoning และ Function Calling",
		Category:       "model_release",
		DeveloperScore: 9.5,
		RelevanceScore: 9.0,
		PrimaryUrl:     "https://www.anthropic.com/news/claude-3-7-sonnet",
		Tags:           "anthropic,claude,reasoning",
	}
	src := &model.NewsSource{
		Id:        1,
		Name:      "Anthropic Official",
		Slug:      "anthropic-official",
		TrustTier: "tier_1_official",
	}

	post := GenerateEditorialPost(cluster, src)
	if post == nil {
		t.Fatal("expected non-nil NewsPost")
	}

	// Verify all 9 required sections from Section 9
	requiredSections := []string{
		"## บทนำและบริบทภาพรวม",
		"## รายละเอียดการประกาศและการเปิดตัว",
		"## สาระสำคัญทางเทคนิคและสเปกโมเดล",
		"## ทำไมการพัฒนานี้ถึงสำคัญ",
		"## ผลกระทบต่อนักพัฒนาและธุรกิจซอฟต์แวร์ไทย",
		"## แนวทางการนำไปประยุกต์ใช้งานจริง",
		"## มุมมองเชิงสถาปัตยกรรม & Tora API Integration",
		"## ข้อสรุปสำคัญ",
		"## แหล่งข้อมูลอ้างอิงต้นฉบับ",
	}

	for _, sec := range requiredSections {
		if !strings.Contains(post.ContentMarkdown, sec) {
			t.Errorf("editorial markdown missing required section: %q", sec)
		}
	}

	// Verify SEO attributes (Section 13)
	if post.CanonicalUrl == "" || !strings.Contains(post.CanonicalUrl, "/news/") {
		t.Errorf("invalid canonical URL: %s", post.CanonicalUrl)
	}
	if post.OgImageUrl == "" || !strings.Contains(post.OgImageUrl, "/og.png") {
		t.Errorf("invalid OgImageUrl: %s", post.OgImageUrl)
	}
	if post.SeoTitle == "" {
		t.Errorf("empty SeoTitle")
	}
	if post.SeoDescription == "" {
		t.Errorf("empty SeoDescription")
	}

	// Verify Tora Integration Mention (Section 10)
	if !strings.Contains(post.ContentMarkdown, "Tora Managed API") {
		t.Errorf("editorial content missing Tora Managed API reference")
	}
}

func TestHuggingFaceGUIDQuirk(t *testing.T) {
	rssWithMissingLink := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Hugging Face Blog</title>
    <item>
      <title>SmolLM2 Release</title>
      <link></link>
      <guid>https://huggingface.co/blog/smollm2</guid>
      <description>Compact language model</description>
      <pubDate>Mon, 06 Oct 2026 10:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`)

	src := &model.NewsSource{
		Id:        10,
		Name:      "Hugging Face Blog",
		Slug:      "hugging-face-blog",
		TrustTier: "tier_1_official",
	}

	items, err := ParseFeedXML(rssWithMissingLink, src)
	if err != nil {
		t.Fatalf("unexpected error parsing HF feed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].URL != "https://huggingface.co/blog/smollm2" {
		t.Fatalf("expected URL to fall back to GUID, got %s", items[0].URL)
	}
}

func TestAnthropicRegexParsing(t *testing.T) {
	mockHtml := `<div><a href="/news/claude-3-7-sonnet">Claude 3.7</a><a href="/news/enterprise-safeguards">Safeguards</a></div>`
	matches := anthropicNewsRegex.FindAllStringSubmatch(mockHtml, -1)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0][1] != "claude-3-7-sonnet" {
		t.Fatalf("expected slug claude-3-7-sonnet, got %s", matches[0][1])
	}
	if matches[1][1] != "enterprise-safeguards" {
		t.Fatalf("expected slug enterprise-safeguards, got %s", matches[1][1])
	}
}
