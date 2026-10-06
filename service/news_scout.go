package service

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// RawStoryItem represents a single item discovered from an external feed or API
type RawStoryItem struct {
	Title       string
	URL         string
	Summary     string
	Author      string
	PublishedAt time.Time
	GUID        string
	SourceName  string
	SourceId    int
	TrustTier   string
	Category    string
	Tags        []string
}

// RSS 2.0 XML structures
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Author      string `xml:"author"`
	Creator     string `xml:"creator"` // Dublin Core dc:creator
	GUID        string `xml:"guid"`
}

// Atom XML structures
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	ID        string     `xml:"id"`
	Author    atomAuthor `xml:"author"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

var scoutHttpClient = &http.Client{
	Timeout: 15 * time.Second,
}

// ParseFeedXML parses raw XML bytes into normalized RawStoryItems, detecting RSS or Atom automatically
func ParseFeedXML(data []byte, src *model.NewsSource) ([]*RawStoryItem, error) {
	if len(data) == 0 {
		return nil, errors.New("empty feed content")
	}

	// Try RSS 2.0
	var rss rssFeed
	if err := xml.Unmarshal(data, &rss); err == nil && len(rss.Channel.Items) > 0 {
		var items []*RawStoryItem
		for _, it := range rss.Channel.Items {
			title := strings.TrimSpace(it.Title)
			link := strings.TrimSpace(it.Link)
			if title == "" || link == "" {
				continue
			}
			author := strings.TrimSpace(it.Creator)
			if author == "" {
				author = strings.TrimSpace(it.Author)
			}
			pubTime := parseFlexDate(it.PubDate)
			guid := strings.TrimSpace(it.GUID)
			if guid == "" {
				guid = link
			}

			items = append(items, &RawStoryItem{
				Title:       title,
				URL:         link,
				Summary:     cleanSnippet(it.Description),
				Author:      author,
				PublishedAt: pubTime,
				GUID:        guid,
				SourceName:  src.Name,
				SourceId:    src.Id,
				TrustTier:   src.TrustTier,
			})
		}
		return items, nil
	}

	// Try Atom
	var atom atomFeed
	if err := xml.Unmarshal(data, &atom); err == nil && len(atom.Entries) > 0 {
		var items []*RawStoryItem
		for _, entry := range atom.Entries {
			title := strings.TrimSpace(entry.Title)
			link := ""
			for _, l := range entry.Links {
				if l.Rel == "alternate" || l.Rel == "" {
					link = strings.TrimSpace(l.Href)
					break
				}
			}
			if title == "" || link == "" {
				continue
			}
			summary := entry.Summary
			if summary == "" {
				summary = entry.Content
			}
			dateStr := entry.Published
			if dateStr == "" {
				dateStr = entry.Updated
			}
			pubTime := parseFlexDate(dateStr)
			guid := strings.TrimSpace(entry.ID)
			if guid == "" {
				guid = link
			}

			items = append(items, &RawStoryItem{
				Title:       title,
				URL:         link,
				Summary:     cleanSnippet(summary),
				Author:      strings.TrimSpace(entry.Author.Name),
				PublishedAt: pubTime,
				GUID:        guid,
				SourceName:  src.Name,
				SourceId:    src.Id,
				TrustTier:   src.TrustTier,
			})
		}
		return items, nil
	}

	return nil, errors.New("unrecognized feed format (neither RSS 2.0 nor Atom)")
}

// FetchSourceFeed downloads and parses the feed for a given NewsSource
func FetchSourceFeed(ctx context.Context, src *model.NewsSource) ([]*RawStoryItem, error) {
	if src == nil || src.FeedUrl == "" {
		return nil, errors.New("invalid news source")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.FeedUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ToraNewsScout/1.0 (+https://www.toraapi.com)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9, */*;q=0.8")

	resp, err := scoutHttpClient.Do(req)
	if err != nil {
		src.FetchErrorCount++
		src.LastError = err.Error()
		_ = model.UpdateNewsSource(src)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errStr := fmt.Sprintf("HTTP %d from feed URL", resp.StatusCode)
		src.FetchErrorCount++
		src.LastError = errStr
		_ = model.UpdateNewsSource(src)
		return nil, errors.New(errStr)
	}

	// Limit to max 5MB
	limitReader := io.LimitReader(resp.Body, 5*1024*1024)
	bodyBytes, err := io.ReadAll(limitReader)
	if err != nil {
		src.FetchErrorCount++
		src.LastError = err.Error()
		_ = model.UpdateNewsSource(src)
		return nil, err
	}

	items, err := ParseFeedXML(bodyBytes, src)
	if err != nil {
		src.FetchErrorCount++
		src.LastError = err.Error()
		_ = model.UpdateNewsSource(src)
		return nil, err
	}

	// Update source telemetry on success
	src.LastFetchedAt = common.GetTimestamp()
	src.FetchErrorCount = 0
	src.LastError = ""
	_ = model.UpdateNewsSource(src)

	// Enrich scoring
	for _, it := range items {
		it.Category = classifyCategory(it.Title, it.Summary)
		it.Tags = extractTags(it.Title, it.Summary)
	}

	return items, nil
}

// ScoreStory calculates technical relevance for Tora AI audience
func ScoreStory(item *RawStoryItem) (relevance float64, dev float64, thai float64) {
	content := strings.ToLower(item.Title + " " + item.Summary)

	// Developer / API keywords
	devKeywords := []string{
		"api", "sdk", "token", "context window", "latency", "inference",
		"endpoint", "gpt-4", "claude-3", "gemini-2", "deepseek", "pricing",
		"python", "rest", "function calling", "prompt caching", "open source",
		"benchmark", "weights", "fine-tuning", "rag", "embeddings",
	}

	for _, kw := range devKeywords {
		if strings.Contains(content, kw) {
			dev += 1.5
		}
	}
	if dev > 10.0 {
		dev = 10.0
	}

	// Thailand / Southeast Asia relevance
	thaiKeywords := []string{"thailand", "thai", "southeast asia", "asean", "bangkok", "บาท", "ภาษาไทย"}
	for _, kw := range thaiKeywords {
		if strings.Contains(content, kw) {
			thai += 4.0
		}
	}
	if thai > 10.0 {
		thai = 10.0
	}
	if thai == 0 {
		// Default base score for universal global tech news
		thai = 5.0
	}

	// Overall relevance is weighted combination
	relevance = (dev*0.7 + thai*0.3)
	if relevance > 10.0 {
		relevance = 10.0
	}

	return relevance, dev, thai
}

func parseFlexDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Now()
	}

	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05-0700",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}
	return time.Now()
}

func cleanSnippet(raw string) string {
	raw = strings.TrimSpace(raw)
	// Strip HTML tags for summary snippet
	var buf bytes.Buffer
	inTag := false
	for _, r := range raw {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			buf.WriteRune(r)
		}
	}
	clean := strings.Join(strings.Fields(buf.String()), " ")
	if len(clean) > 300 {
		return clean[:297] + "..."
	}
	return clean
}

func classifyCategory(title, summary string) string {
	combined := strings.ToLower(title + " " + summary)
	if strings.Contains(combined, "price") || strings.Contains(combined, "cost") || strings.Contains(combined, "discount") {
		return "pricing"
	}
	if strings.Contains(combined, "release") || strings.Contains(combined, "launch") || strings.Contains(combined, "announce") {
		return "model_release"
	}
	if strings.Contains(combined, "benchmark") || strings.Contains(combined, "research") || strings.Contains(combined, "paper") {
		return "research"
	}
	if strings.Contains(combined, "sdk") || strings.Contains(combined, "library") || strings.Contains(combined, "client") {
		return "sdk"
	}
	return "api_update"
}

func extractTags(title, summary string) []string {
	combined := strings.ToLower(title + " " + summary)
	tagCandidates := []string{
		"openai", "gpt-4", "gpt-4.5", "chatgpt",
		"anthropic", "claude", "claude-3.7",
		"google", "gemini", "gemma", "deepmind",
		"deepseek", "deepseek-v3", "deepseek-r1",
		"openrouter", "new-api", "api", "pricing",
	}

	var found []string
	for _, cand := range tagCandidates {
		if strings.Contains(combined, cand) {
			found = append(found, cand)
		}
	}
	return found
}
