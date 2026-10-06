package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// MockProviderMode dictates the deterministic test execution behavior.
type MockProviderMode string

const (
	MockModeInstantSuccess MockProviderMode = "instant_success"
	MockModeDelayedSuccess MockProviderMode = "delayed_success"
	MockModePermanentFail  MockProviderMode = "permanent_fail"
	MockModeAmbiguousFail  MockProviderMode = "ambiguous_fail"
)

// DeterministicMockProvider provides a controllable test double for Studio providers.
type DeterministicMockProvider struct {
	mu           sync.Mutex
	Mode         MockProviderMode
	SubmitCalls  int
	PollCalls    int
	CancelCalls  int
	JobStatusMap map[string]string
	PollCountMap map[string]int
}

func NewDeterministicMockProvider(mode MockProviderMode) *DeterministicMockProvider {
	if mode == "" {
		mode = MockModeInstantSuccess
	}
	return &DeterministicMockProvider{
		Mode:         mode,
		JobStatusMap: make(map[string]string),
		PollCountMap: make(map[string]int),
	}
}

func (m *DeterministicMockProvider) Name() string {
	return "mock"
}

func (m *DeterministicMockProvider) Submit(ctx context.Context, job *model.StudioToolJob) (*ProviderSubmitResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SubmitCalls++

	providerJobId := fmt.Sprintf("mock_job_%d_%d", time.Now().UnixNano(), m.SubmitCalls)

	switch m.Mode {
	case MockModePermanentFail:
		return nil, ErrProviderPermanent

	case MockModeAmbiguousFail:
		return nil, ErrProviderAmbiguous

	case MockModeDelayedSuccess:
		m.JobStatusMap[providerJobId] = "queued"
		return &ProviderSubmitResult{
			ProviderJobId: providerJobId,
			Status:        "queued",
			RawResponse:   `{"mock_status": "queued"}`,
		}, nil

	case MockModeInstantSuccess:
		fallthrough
	default:
		m.JobStatusMap[providerJobId] = "completed"
		return &ProviderSubmitResult{
			ProviderJobId: providerJobId,
			Status:        "completed",
			OutputURL:     fmt.Sprintf("https://cdn.toraapi.com/mock-assets/%s/output.png", job.Id),
			RawResponse:   `{"mock_status": "completed"}`,
		}, nil
	}
}

func (m *DeterministicMockProvider) Poll(ctx context.Context, providerJobId string) (*ProviderPollResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PollCalls++

	m.PollCountMap[providerJobId]++
	count := m.PollCountMap[providerJobId]

	if m.Mode == MockModeDelayedSuccess {
		if count == 1 {
			m.JobStatusMap[providerJobId] = "processing"
			return &ProviderPollResult{
				Status:   "processing",
				Progress: 50,
			}, nil
		}
		m.JobStatusMap[providerJobId] = "completed"
		return &ProviderPollResult{
			Status:    "completed",
			Progress:  100,
			OutputURL: fmt.Sprintf("https://cdn.toraapi.com/mock-assets/%s/delayed_output.png", providerJobId),
		}, nil
	}

	status, exists := m.JobStatusMap[providerJobId]
	if !exists {
		status = "completed"
	}

	return &ProviderPollResult{
		Status:    status,
		Progress:  100,
		OutputURL: fmt.Sprintf("https://cdn.toraapi.com/mock-assets/%s/output.png", providerJobId),
	}, nil
}

func (m *DeterministicMockProvider) Cancel(ctx context.Context, providerJobId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CancelCalls++
	m.JobStatusMap[providerJobId] = "cancelled"
	return nil
}
