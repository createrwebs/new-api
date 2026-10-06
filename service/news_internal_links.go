package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type InternalLinkRecommendation struct {
	SourcePostID int    `json:"source_post_id"`
	TargetPostID int    `json:"target_post_id,omitempty"`
	TargetURL    string `json:"target_url"`
	AnchorText   string `json:"anchor_text"`
	Relationship string `json:"relationship"` // cluster_peer, core_page, evergreen_guide
	Reason       string `json:"reason"`
}

// RecommendInternalLinks builds internal linking recommendations from topic clusters, entities, and core Tora pages
// Adheres strictly to Section 11: Avoid excessive exact-match anchors, never create orphan news articles
func RecommendInternalLinks(postId int) ([]InternalLinkRecommendation, error) {
	post, err := model.GetNewsPostById(postId)
	if err != nil || post == nil {
		return nil, fmt.Errorf("post %d not found", postId)
	}

	var recs []InternalLinkRecommendation
	baseURL := common.GetCanonicalBaseURL()

	// 1. Mandatory core platform links (ensuring commercial discovery without exact match spam)
	recs = append(recs, InternalLinkRecommendation{
		SourcePostID: postId,
		TargetURL:    baseURL,
		AnchorText:   "Tora Managed API",
		Relationship: "core_page",
		Reason:       "Connects technical news readers to server-managed API Gateway",
	})
	recs = append(recs, InternalLinkRecommendation{
		SourcePostID: postId,
		TargetURL:    fmt.Sprintf("%s/pricing", baseURL),
		AnchorText:   "Tora Pricing & Plans",
		Relationship: "core_page",
		Reason:       "Exposes model pricing tables, token multipliers, and subscription tiers",
	})
	recs = append(recs, InternalLinkRecommendation{
		SourcePostID: postId,
		TargetURL:    fmt.Sprintf("%s/docs", baseURL),
		AnchorText:   "Tora Technical Documentation",
		Relationship: "core_page",
		Reason:       "Guides developers on API client integration and SDK usage",
	})

	// 2. Peer cluster articles (prevents orphan news articles)
	published, _, _ := model.GetPublishedNewsPosts(1, 20, "", "", "")
	for _, p := range published {
		if p.Id == postId {
			continue
		}
		isRelated := false
		relType := "related_article"
		if post.ClusterId > 0 && p.ClusterId == post.ClusterId {
			isRelated = true
			relType = "cluster_peer"
		} else if p.ContentType == model.ContentTypeGuide {
			isRelated = true
			relType = "evergreen_guide"
		}

		if isRelated {
			targetURL := p.CanonicalUrl
			if targetURL == "" {
				targetURL = fmt.Sprintf("%s/news/%s", baseURL, p.Slug)
			}
			anchor := common.TruncateRunesWithEllipsis(p.Title, 50)
			recs = append(recs, InternalLinkRecommendation{
				SourcePostID: postId,
				TargetPostID: p.Id,
				TargetURL:    targetURL,
				AnchorText:   anchor,
				Relationship: relType,
				Reason:       fmt.Sprintf("Related %s providing topical depth and preventing orphan crawl graph", p.ContentType),
			})
			if len(recs) >= 6 {
				break
			}
		}
	}

	return recs, nil
}
