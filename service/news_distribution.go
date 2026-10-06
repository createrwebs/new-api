package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// GenerateSocialDerivatives generates platform-specific content variants for multi-channel distribution
func GenerateSocialDerivatives(post *model.NewsPost) []*model.NewsDistribution {
	var dists []*model.NewsDistribution

	// 1. Facebook Thai Post
	fbContent := fmt.Sprintf(
		"🚀 [อัปเดต AI ล่าสุด] %s\n\n"+
			"📌 สรุปสาระสำคัญ:\n%s\n\n"+
			"💡 มุมมองสำหรับนักพัฒนาไทย:\n"+
			"- รองรับการเชื่อมต่อผ่าน Tora AI Managed API และ BYOK\n"+
			"- ตรวจสอบความหน่วงและราคาโทเค็นได้ทันที\n\n"+
			"📖 อ่านบทวิเคราะห์และโค้ดตัวอย่างฉบับเต็มได้ที่:\n%s\n\n"+
			"#ToraAI #AINews #TechUpdate #LLM #ThailandDevelopers",
		post.Title,
		post.Summary,
		post.CanonicalUrl,
	)
	dists = append(dists, &model.NewsDistribution{
		PostId:         post.Id,
		Platform:       model.DistPlatformFacebook,
		Status:         "pending",
		ContentPayload: fbContent,
	})

	// 2. LinkedIn Technical Post
	liContent := fmt.Sprintf(
		"Technical Analysis: %s\n\n"+
			"Key takeaways for software engineers and AI builders:\n\n"+
			"%s\n\n"+
			"Explore implementation details, benchmarks, and Thai developer impact on Tora AI News:\n%s\n\n"+
			"#ArtificialIntelligence #MachineLearning #LLM #SoftwareEngineering #CloudComputing #ToraAI",
		post.Title,
		post.Summary,
		post.CanonicalUrl,
	)
	dists = append(dists, &model.NewsDistribution{
		PostId:         post.Id,
		Platform:       model.DistPlatformLinkedIn,
		Status:         "pending",
		ContentPayload: liContent,
	})

	// 3. Twitter / X Short Thread
	twContent := fmt.Sprintf(
		"⚡ %s\n\n"+
			"• %s\n\n"+
			"อ่านบทวิเคราะห์สำหรับนักพัฒนา: %s\n\n"+
			"#ToraAI #LLM #AI",
		post.Title,
		post.Summary,
		post.CanonicalUrl,
	)
	if len(twContent) > 280 {
		twContent = fmt.Sprintf("⚡ %s\n\nอ่านฉบับเต็ม: %s\n#ToraAI #AI", post.Title, post.CanonicalUrl)
	}
	dists = append(dists, &model.NewsDistribution{
		PostId:         post.Id,
		Platform:       model.DistPlatformTwitter,
		Status:         "pending",
		ContentPayload: twContent,
	})

	// 4. DEV.to / Technical Blog Cross-Post
	devToContent := fmt.Sprintf(
		"---\n"+
			"title: \"%s\"\n"+
			"published: true\n"+
			"tags: ai, llm, technology, news\n"+
			"canonical_url: %s\n"+
			"---\n\n"+
			"%s\n\n"+
			"> บทความนี้เผยแพร่ครั้งแรกที่ [Tora AI News](%s)",
		strings.ReplaceAll(post.Title, "\"", "\\\""),
		post.CanonicalUrl,
		post.ContentMarkdown,
		post.CanonicalUrl,
	)
	dists = append(dists, &model.NewsDistribution{
		PostId:         post.Id,
		Platform:       model.DistPlatformDevTo,
		Status:         "pending",
		ContentPayload: devToContent,
	})

	return dists
}

// QueuePostDistributions saves all generated social derivatives to the database idempotently
func QueuePostDistributions(post *model.NewsPost) error {
	derivatives := GenerateSocialDerivatives(post)
	for _, d := range derivatives {
		if d.DistributionVersion <= 0 {
			d.DistributionVersion = 1
		}
		// Idempotency: skip if distribution intent already exists for this (post, platform, version)
		existing, err := model.GetDistributionByPostPlatformAndVersion(post.Id, d.Platform, d.DistributionVersion)
		if err == nil && existing != nil {
			continue
		}
		if err := model.CreateNewsDistribution(d); err != nil {
			return err
		}
	}
	return nil
}
