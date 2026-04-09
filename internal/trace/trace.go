// Package trace provides lightweight HTTP tracing for the gateway client.
//
// Wrap any http.RoundTripper with NewRoundTripper to automatically record
// a Span for every outgoing request. Spans are emitted to a SpanSink;
// the default LogSink writes JSON to stderr, gated by DOJO_TRACE=1.
package trace

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// traceEnabled is checked once at package init. When false the RoundTripper
// still propagates headers but skips span recording entirely — zero overhead.
var traceEnabled bool

func init() {
	traceEnabled = os.Getenv("DOJO_TRACE") == "1"
}

// headerTraceID is the HTTP header used for trace correlation.
const headerTraceID = "X-Trace-ID"

// Span captures timing and metadata for a single HTTP round-trip.
type Span struct {
	TraceID    string `json:"trace_id"`
	SpanID     string `json:"span_id"`
	Method     string `json:"method"`
	URL        string `json:"url"`
	StatusCode int    `json:"status"`
	StartTime  time.Time `json:"-"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// SpanSink receives completed spans. Implementations must be safe for
// concurrent use. The interface is intentionally minimal so an OTEL
// adapter can be added without changing the transport layer.
type SpanSink interface {
	Emit(Span)
}

// LogSink writes JSON-encoded spans to stderr.
type LogSink struct{}

// Emit writes s as a single JSON line to stderr.
func (LogSink) Emit(s Span) {
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%s\n", data)
}

// roundTripper wraps an http.RoundTripper, creating a Span for every request.
type roundTripper struct {
	inner http.RoundTripper
	sink  SpanSink
}

// NewRoundTripper returns an http.RoundTripper that instruments every request.
// If inner is nil, http.DefaultTransport is used. If sink is nil and tracing
// is enabled, a LogSink is used.
func NewRoundTripper(inner http.RoundTripper, sink SpanSink) http.RoundTripper {
	if inner == nil {
		inner = http.DefaultTransport
	}
	if sink == nil {
		sink = LogSink{}
	}
	return &roundTripper{inner: inner, sink: sink}
}

// RoundTrip implements http.RoundTripper.
func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Always propagate a trace ID header, even when tracing is off.
	traceID := req.Header.Get(headerTraceID)
	if traceID == "" {
		traceID = randomHex(8)
		req = req.Clone(req.Context())
		req.Header.Set(headerTraceID, traceID)
	}

	if !traceEnabled {
		return rt.inner.RoundTrip(req)
	}

	span := Span{
		TraceID:   traceID,
		SpanID:    randomHex(4),
		Method:    req.Method,
		URL:       req.URL.String(),
		StartTime: time.Now(),
	}

	resp, err := rt.inner.RoundTrip(req)

	span.DurationMS = time.Since(span.StartTime).Milliseconds()
	if err != nil {
		span.Error = err.Error()
	}
	if resp != nil {
		span.StatusCode = resp.StatusCode
	}

	rt.sink.Emit(span)

	return resp, err
}

// randomHex returns n random bytes encoded as a hex string (2n chars).
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
