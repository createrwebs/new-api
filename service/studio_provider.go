package service

import (
	"context"
	"errors"

	"github.com/QuantumNous/new-api/model"
)

var (
	ErrProviderTransient = errors.New("temporary upstream provider failure (retryable)")
	ErrProviderPermanent = errors.New("permanent upstream provider rejection")
	ErrProviderAmbiguous = errors.New("ambiguous submission state (reconciliation required)")
)

// ProviderSubmitResult contains the response after initiating a generation job.
type ProviderSubmitResult struct {
	ProviderJobId string `json:"provider_job_id"`
	Status        string `json:"status"` // "queued", "processing", "completed", "failed"
	OutputURL     string `json:"output_url,omitempty"`
	RawResponse   string `json:"raw_response,omitempty"`
}

// ProviderPollResult contains the status of a queried asynchronous job.
type ProviderPollResult struct {
	Status       string `json:"status"` // "queued", "processing", "completed", "failed"
	OutputURL    string `json:"output_url,omitempty"`
	Progress     int    `json:"progress,omitempty"` // 0 - 100
	ErrorMessage string `json:"error_message,omitempty"`
}

// StudioProvider defines the contract implemented by media generation backends.
type StudioProvider interface {
	Name() string
	Submit(ctx context.Context, job *model.StudioToolJob) (*ProviderSubmitResult, error)
	Poll(ctx context.Context, providerJobId string) (*ProviderPollResult, error)
	Cancel(ctx context.Context, providerJobId string) error
}
