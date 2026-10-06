package relay

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

var (
	ErrAlreadyCommitted = errors.New("cannot reset response writer: response headers or payload already committed downstream")
)

// CommitDetectingWriter wraps gin.ResponseWriter to provide a deferred response boundary.
// Upstream headers and initial output chunks are staged in a private buffer until the attempt
// is confirmed valid (e.g. first valid SSE token or complete non-streaming response body).
// If failure occurs before commitment, Reset() safely discards the staged data so fallback can proceed cleanly.
// Once Commit() is invoked, all writes pass directly to the client socket and further fallback is strictly forbidden.
type CommitDetectingWriter struct {
	underlying     gin.ResponseWriter
	privateHeader  http.Header
	committed      atomic.Bool
	commitOnce     sync.Once
	pendingStatus  int
	bufferedOutput bytes.Buffer
	mu             sync.Mutex
}

func NewCommitDetectingWriter(underlying gin.ResponseWriter) *CommitDetectingWriter {
	return &CommitDetectingWriter{
		underlying:    underlying,
		privateHeader: underlying.Header().Clone(),
	}
}

func (w *CommitDetectingWriter) IsCommitted() bool {
	return w.committed.Load()
}

// Commit flushes staged headers, status code, and buffered payload to the underlying client connection.
func (w *CommitDetectingWriter) Commit() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var writeErr error
	w.commitOnce.Do(func() {
		target := w.underlying.Header()
		for key := range target {
			target.Del(key)
		}
		for key, values := range w.privateHeader {
			target[key] = append([]string(nil), values...)
		}
		w.committed.Store(true)

		status := w.pendingStatus
		if status == 0 {
			status = http.StatusOK
		}
		w.underlying.WriteHeader(status)

		if w.bufferedOutput.Len() > 0 {
			_, writeErr = w.underlying.Write(w.bufferedOutput.Bytes())
			w.bufferedOutput.Reset()
		}
		w.underlying.Flush()
	})
	return writeErr
}

// Reset safely clears staged headers and buffered output if the attempt failed before commitment.
func (w *CommitDetectingWriter) Reset() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.committed.Load() {
		return ErrAlreadyCommitted
	}

	w.pendingStatus = 0
	w.bufferedOutput.Reset()
	w.privateHeader = w.underlying.Header().Clone()
	w.commitOnce = sync.Once{}
	return nil
}

func (w *CommitDetectingWriter) Header() http.Header {
	if w.committed.Load() {
		return w.underlying.Header()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.privateHeader
}

func (w *CommitDetectingWriter) WriteHeader(statusCode int) {
	if w.committed.Load() {
		w.underlying.WriteHeader(statusCode)
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pendingStatus == 0 {
		w.pendingStatus = statusCode
	}
}

func (w *CommitDetectingWriter) Write(data []byte) (int, error) {
	if w.committed.Load() {
		return w.underlying.Write(data)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.bufferedOutput.Write(data)
}

func (w *CommitDetectingWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

var _ gin.ResponseWriter = (*CommitDetectingWriter)(nil)

func (w *CommitDetectingWriter) Flush() {
	if w.committed.Load() {
		w.underlying.Flush()
		return
	}
	w.mu.Lock()
	hasData := w.bufferedOutput.Len() > 0
	w.mu.Unlock()
	if hasData {
		_ = w.Commit()
	}
}

func (w *CommitDetectingWriter) Status() int {
	if w.committed.Load() {
		return w.underlying.Status()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pendingStatus != 0 {
		return w.pendingStatus
	}
	return w.underlying.Status()
}

func (w *CommitDetectingWriter) Size() int {
	if w.committed.Load() {
		return w.underlying.Size()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.bufferedOutput.Len()
}

func (w *CommitDetectingWriter) Written() bool {
	return w.committed.Load() || w.underlying.Written()
}

func (w *CommitDetectingWriter) WriteHeaderNow() {
	if w.committed.Load() {
		w.underlying.WriteHeaderNow()
	}
}

func (w *CommitDetectingWriter) Pusher() http.Pusher {
	if !w.committed.Load() {
		return nil
	}
	return w.underlying.Pusher()
}

func (w *CommitDetectingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if !w.committed.Load() {
		return nil, nil, errors.New("cannot hijack uncommitted route attempt")
	}
	return w.underlying.Hijack()
}

func (w *CommitDetectingWriter) CloseNotify() <-chan bool {
	return w.underlying.CloseNotify()
}
