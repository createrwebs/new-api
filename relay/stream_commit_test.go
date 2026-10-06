package relay_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/relay"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitDetectingWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writer := relay.NewCommitDetectingWriter(c.Writer)
	assert.False(t, writer.IsCommitted())

	// Write status and initial chunk before commit
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.WriteHeader(http.StatusOK)
	n, err := writer.WriteString("data: {\"test\":1}\n\n")
	require.NoError(t, err)
	assert.Positive(t, n)

	// Underlying recorder should have received NOTHING yet
	assert.False(t, writer.IsCommitted())
	assert.Empty(t, rec.Body.String(), "underlying recorder must be empty before commit")
	assert.Equal(t, 200, writer.Status())

	// Simulate failure before commit -> Reset()
	err = writer.Reset()
	require.NoError(t, err)
	assert.Equal(t, 0, writer.Size())
	assert.False(t, writer.IsCommitted())

	// Now write for a successful fallback attempt
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("X-Route-Id", "2")
	writer.WriteHeader(http.StatusOK)
	_, err = writer.WriteString("data: {\"first_token\":\"hello\"}\n\n")
	require.NoError(t, err)

	// Commit the winning attempt
	err = writer.Commit()
	require.NoError(t, err)
	assert.True(t, writer.IsCommitted())

	// Verify underlying recorder has the data and headers
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, "2", rec.Header().Get("X-Route-Id"))
	assert.Equal(t, "data: {\"first_token\":\"hello\"}\n\n", rec.Body.String())

	// Subsequent write after commit goes directly to underlying
	_, err = writer.WriteString("data: {\"second_token\":\"world\"}\n\n")
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), "second_token")

	// Attempting Reset after commit MUST fail
	err = writer.Reset()
	assert.ErrorIs(t, err, relay.ErrAlreadyCommitted)
}

func TestCommitDetectingWriter_StreamCommitScenarios(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Scenario 1: WriteHeader -> upstream fails before first valid event
	t.Run("WriteHeader before data", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		w := relay.NewCommitDetectingWriter(c.Writer)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		assert.False(t, w.IsCommitted(), "WriteHeader alone must not commit response")
		assert.Empty(t, rec.Body.String())

		// Safe to reset on failure before data
		require.NoError(t, w.Reset())
		assert.False(t, w.IsCommitted())
	})

	// Scenario 2: Flush before data -> upstream fails
	t.Run("Flush before data", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		w := relay.NewCommitDetectingWriter(c.Writer)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Flush() // Flush called with 0 buffered bytes

		assert.False(t, w.IsCommitted(), "Flush without data must not commit response")
		assert.Empty(t, rec.Body.String())

		require.NoError(t, w.Reset())
		assert.False(t, w.IsCommitted())
	})

	// Scenario 3: First SSE event -> Flush -> upstream fails
	t.Run("First SSE event and Flush", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		w := relay.NewCommitDetectingWriter(c.Writer)
		w.Header().Set("Content-Type", "text/event-stream")
		_, err := w.WriteString("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n")
		require.NoError(t, err)

		w.Flush() // Flushes with data -> Commits!
		assert.True(t, w.IsCommitted(), "Flushing first data event must commit response downstream")
		assert.Contains(t, rec.Body.String(), "Hello")

		// Upstream failure mid-stream cannot reset
		err = w.Reset()
		assert.ErrorIs(t, err, relay.ErrAlreadyCommitted)
	})

	// Scenario 4: Reasoning event -> Flush -> upstream fails
	t.Run("Reasoning event and Flush", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		w := relay.NewCommitDetectingWriter(c.Writer)
		w.Header().Set("Content-Type", "text/event-stream")
		_, err := w.WriteString("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"thinking...\"}}]}\n\n")
		require.NoError(t, err)

		w.Flush()
		assert.True(t, w.IsCommitted())
		assert.Contains(t, rec.Body.String(), "thinking...")

		err = w.Reset()
		assert.ErrorIs(t, err, relay.ErrAlreadyCommitted)
	})

	// Scenario 5: Tool-call event -> Flush -> upstream fails
	t.Run("Tool-call event and Flush", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		w := relay.NewCommitDetectingWriter(c.Writer)
		w.Header().Set("Content-Type", "text/event-stream")
		_, err := w.WriteString("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"id\":\"call_1\"}]}}]}\n\n")
		require.NoError(t, err)

		w.Flush()
		assert.True(t, w.IsCommitted())
		assert.Contains(t, rec.Body.String(), "call_1")

		err = w.Reset()
		assert.ErrorIs(t, err, relay.ErrAlreadyCommitted)
	})

	// Scenario 6: Headers + metadata only
	t.Run("Headers and metadata only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		w := relay.NewCommitDetectingWriter(c.Writer)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		assert.False(t, w.IsCommitted())
		assert.Empty(t, rec.Body.String())
		require.NoError(t, w.Reset())
	})
}
