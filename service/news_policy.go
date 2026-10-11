package service

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// Policy Decisions for Autonomous Publication (Section 13)
const (
	AutonomousDecisionCandidateAutoPublish = "CANDIDATE_FUTURE_AUTOPUBLISH"
	AutonomousDecisionReviewRequired       = "REVIEW_REQUIRED"
	AutonomousDecisionNeverAutoPublish     = "NEVER_AUTOPUBLISH"
)

// AutonomousPolicyDecision records the decision and evaluation rationale for an article
type AutonomousPolicyDecision struct {
	PostID          int      `json:"post_id"`
	Slug            string   `json:"slug"`
	Decision        string   `json:"decision"` // CANDIDATE_FUTURE_AUTOPUBLISH, REVIEW_REQUIRED, NEVER_AUTOPUBLISH
	ContentRisk     string   `json:"content_risk"`
	QualityPassed   bool     `json:"quality_passed"`
	SourceVerified  bool     `json:"source_verified"`
	IsDuplicate     bool     `json:"is_duplicate"`
	IsPublished     bool     `json:"is_published"` // canonical Tora publication prerequisite
	Reasons         []string `json:"reasons"`
	AutoPublishSafe bool     `json:"auto_publish_safe"`
}

// MassAutoPublishState describes the active policy gate status (Section 12, 13)
type MassAutoPublishState struct {
	MassAutopublishEnabled bool   `json:"mass_autopublish_enabled"` // strictly false in current stage
	PolicyVersion          string `json:"policy_version"`           // v1-controlled-staged
	ActiveRuleSummary      string `json:"active_rule_summary"`
	CandidatesCount        int    `json:"candidates_count"`
	ReviewRequiredCount    int    `json:"review_required_count"`
	NeverPublishCount      int    `json:"never_publish_count"`
}

// EvaluateAutonomousPublishEligibility inspects a post against Section 13 Autonomous Mode Policy
func EvaluateAutonomousPublishEligibility(post *model.NewsPost) AutonomousPolicyDecision {
	if post == nil {
		return AutonomousPolicyDecision{
			Decision:        AutonomousDecisionNeverAutoPublish,
			ContentRisk:     "unknown",
			Reasons:         []string{"nil post provided"},
			AutoPublishSafe: false,
		}
	}

	res := AutonomousPolicyDecision{
		PostID:      post.Id,
		Slug:        post.Slug,
		ContentRisk: post.ContentRisk,
	}

	// 1. Canonical Tora Publication Prerequisite
	// "External distribution happens only after successful canonical Tora publication."
	if post.Status == "published" {
		res.IsPublished = true
	} else {
		res.IsPublished = false
		res.Reasons = append(res.Reasons, "post not yet canonically published on Tora (status is "+post.Status+")")
	}

	// 2. Risk Evaluation
	// "HIGH risk = never auto-publish"
	if strings.ToLower(post.ContentRisk) == "high" {
		res.Decision = AutonomousDecisionNeverAutoPublish
		res.AutoPublishSafe = false
		res.Reasons = append(res.Reasons, "editorial risk is HIGH: strictly requires human intervention")
		return res
	}

	// "MEDIUM risk = review required"
	if strings.ToLower(post.ContentRisk) == "medium" {
		res.Decision = AutonomousDecisionReviewRequired
		res.AutoPublishSafe = false
		res.Reasons = append(res.Reasons, "editorial risk is MEDIUM: requires human editorial review")
		return res
	}

	// 3. LOW Risk Criteria Evaluation
	// "LOW risk + quality gate passed + source verification passed + not duplicate = candidate for future auto-publish"
	qualityPassed := true
	if len(strings.TrimSpace(post.Title)) < 10 {
		qualityPassed = false
		res.Reasons = append(res.Reasons, "title too short for quality gate")
	}
	if len(strings.TrimSpace(post.ContentMarkdown)) < 200 {
		qualityPassed = false
		res.Reasons = append(res.Reasons, "content body too short for editorial standard")
	}
	if strings.TrimSpace(post.Summary) == "" {
		qualityPassed = false
		res.Reasons = append(res.Reasons, "article summary missing")
	}
	res.QualityPassed = qualityPassed

	// Source Verification
	sourceVerified := true
	if post.FactCheckStatus == "disputed" {
		sourceVerified = false
		res.Reasons = append(res.Reasons, "source fact check is disputed")
	}
	res.SourceVerified = sourceVerified

	// Duplicate Check
	isDuplicate := false
	if post.IsSeed {
		isDuplicate = true
		res.Reasons = append(res.Reasons, "seed or duplicate content")
	}
	res.IsDuplicate = isDuplicate

	if qualityPassed && sourceVerified && !isDuplicate && res.IsPublished {
		res.Decision = AutonomousDecisionCandidateAutoPublish
		res.AutoPublishSafe = true
		res.Reasons = append(res.Reasons, "all autonomous criteria satisfied (candidate for future auto-publish)")
	} else {
		res.Decision = AutonomousDecisionReviewRequired
		res.AutoPublishSafe = false
	}

	return res
}

// GetMassAutoPublishPolicyState returns the current fleet policy status
func GetMassAutoPublishPolicyState(samplePosts []*model.NewsPost) MassAutoPublishState {
	state := MassAutoPublishState{
		MassAutopublishEnabled: false, // Section 12: MASS_AUTOPUBLISH=false strictly enforced
		PolicyVersion:          "v1-controlled-staged",
		ActiveRuleSummary:      "LOW risk + quality + source + not duplicate = candidate; MEDIUM = review; HIGH = never",
	}

	for _, p := range samplePosts {
		dec := EvaluateAutonomousPublishEligibility(p)
		switch dec.Decision {
		case AutonomousDecisionCandidateAutoPublish:
			state.CandidatesCount++
		case AutonomousDecisionReviewRequired:
			state.ReviewRequiredCount++
		case AutonomousDecisionNeverAutoPublish:
			state.NeverPublishCount++
		}
	}

	return state
}
