package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
)

type IndexNowClient struct {
	APIKey      string
	Host        string
	KeyLocation string
	Client      *http.Client
}

func NewIndexNowClient() *IndexNowClient {
	key := os.Getenv("INDEXNOW_KEY")
	if key == "" {
		key = "tora-ai-indexnow-2026"
	}
	host := "www.toraapi.com"
	keyLocation := fmt.Sprintf("https://%s/%s.txt", host, key)
	return &IndexNowClient{
		APIKey:      key,
		Host:        host,
		KeyLocation: keyLocation,
		Client:      &http.Client{Timeout: 10 * time.Second},
	}
}

// SubmitURLs submits production URLs to IndexNow for Bing/Yandex/Seznam
// Decoupled from publication success: failures are logged as warnings and never block publication (Section 13)
func (c *IndexNowClient) SubmitURLs(ctx context.Context, urls []string) error {
	if len(urls) == 0 {
		return nil
	}

	// Strictly validate production URLs
	var validURLs []string
	for _, u := range urls {
		if strings.HasPrefix(u, "https://www.toraapi.com/news/") {
			validURLs = append(validURLs, u)
		}
	}
	if len(validURLs) == 0 {
		return nil
	}

	payload := map[string]interface{}{
		"host":        c.Host,
		"key":         c.APIKey,
		"keyLocation": c.KeyLocation,
		"urlList":     validURLs,
	}

	reqBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.indexnow.org/indexnow", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.Client.Do(req)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("[IndexNow] Submission failed (non-blocking): %v", err))
		return nil // Non-blocking
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.LogWarn(ctx, fmt.Sprintf("[IndexNow] HTTP error %d (non-blocking)", resp.StatusCode))
	} else {
		logger.LogInfo(ctx, fmt.Sprintf("[IndexNow] Submitted %d URLs successfully (HTTP %d)", len(validURLs), resp.StatusCode))
	}
	return nil
}
