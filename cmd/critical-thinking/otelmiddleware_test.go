package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/jacaudi/critical-thinking/internal/thinking"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTestTelemetry installs in-memory global tracer/meter providers and
// restores the previous globals on cleanup. Callers must construct the
// server under test AFTER calling this (instruments bind at creation), and
// must not use t.Parallel().
func setupTestTelemetry(t *testing.T) (*tracetest.InMemoryExporter, *sdkmetric.ManualReader) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	prevTP := otel.GetTracerProvider()
	prevMP := otel.GetMeterProvider()
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetMeterProvider(prevMP)
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	return exp, reader
}

// newTelemetryTestServer builds the same handler stack runHTTP uses (minus
// otelhttp, which Task 7 adds) around fresh per-session state — the same
// construction as the existing TestCrossSessionIsolation setup
// (mcpserver_test.go:260-273).
func newTelemetryTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return newMCPServer(thinking.NewServer())
	}, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	ts := httptest.NewServer(withCORS(mux, nil))
	t.Cleanup(ts.Close)
	return ts
}

func TestMiddlewareRecordsToolCallSpan(t *testing.T) {
	exp, _ := setupTestTelemetry(t)
	ts := newTelemetryTestServer(t)

	client := newHTTPClient(t, ts.URL)
	client.callTool(t, validInputN(1, "otel"))

	var toolSpan *tracetest.SpanStub
	for _, s := range exp.GetSpans() {
		if s.Name == "mcp.tools/call" {
			toolSpan = &s
			break
		}
	}
	if toolSpan == nil {
		t.Fatalf("no mcp.tools/call span recorded; got %d spans", len(exp.GetSpans()))
	}
	attrs := make(map[attribute.Key]attribute.Value, len(toolSpan.Attributes))
	for _, kv := range toolSpan.Attributes {
		attrs[kv.Key] = kv.Value
	}
	if got := attrs["mcp.tool.name"].AsString(); got != "criticalthinking" {
		t.Errorf("mcp.tool.name = %q, want criticalthinking", got)
	}
	if got := attrs["mcp.method"].AsString(); got != "tools/call" {
		t.Errorf("mcp.method = %q, want tools/call", got)
	}
	if got := attrs["ct.outcome"].AsString(); got != "ok" {
		t.Errorf("ct.outcome = %q, want ok", got)
	}
}

func TestMiddlewareRecordsCallMetrics(t *testing.T) {
	_, reader := setupTestTelemetry(t)
	ts := newTelemetryTestServer(t)

	client := newHTTPClient(t, ts.URL)
	client.callTool(t, validInputN(1, "otel"))

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	calls := findMetric(t, rm, "ct.mcp.calls")
	sum, ok := calls.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("ct.mcp.calls data type = %T, want Sum[int64]", calls.Data)
	}
	foundOK := false
	for _, dp := range sum.DataPoints {
		method, _ := dp.Attributes.Value("mcp.method")
		outcome, _ := dp.Attributes.Value("ct.outcome")
		if method.AsString() == "tools/call" && outcome.AsString() == "ok" && dp.Value >= 1 {
			foundOK = true
		}
	}
	if !foundOK {
		t.Errorf("no ct.mcp.calls data point for tools/call outcome=ok")
	}

	dur := findMetric(t, rm, "ct.mcp.duration")
	hist, ok := dur.Data.(metricdata.Histogram[float64])
	if !ok || len(hist.DataPoints) == 0 {
		t.Errorf("ct.mcp.duration missing histogram data points (type %T)", dur.Data)
	}
}

func findMetric(t *testing.T, rm metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("metric %q not found", name)
	return metricdata.Metrics{}
}

func TestToolSpanCarriesDomainAttributes(t *testing.T) {
	exp, _ := setupTestTelemetry(t)
	ts := newTelemetryTestServer(t)

	client := newHTTPClient(t, ts.URL)
	client.callTool(t, validInputN(3, "otel-domain"))

	var toolSpan *tracetest.SpanStub
	for _, s := range exp.GetSpans() {
		if s.Name == "mcp.tools/call" {
			toolSpan = &s
			break
		}
	}
	if toolSpan == nil {
		t.Fatal("no mcp.tools/call span recorded")
	}
	attrs := make(map[attribute.Key]attribute.Value, len(toolSpan.Attributes))
	for _, kv := range toolSpan.Attributes {
		attrs[kv.Key] = kv.Value
	}
	if got := attrs["ct.thought_number"].AsInt64(); got != 3 {
		t.Errorf("ct.thought_number = %d, want 3", got)
	}
	if got := attrs["ct.total_thoughts"].AsInt64(); got != 20 {
		t.Errorf("ct.total_thoughts = %d, want 20", got)
	}
	if got := attrs["ct.confidence"].AsFloat64(); got != 0.5 {
		t.Errorf("ct.confidence = %v, want 0.5", got)
	}
	if attrs["ct.is_revision"].AsBool() || attrs["ct.is_branch"].AsBool() {
		t.Errorf("is_revision/is_branch should be false for a plain trunk thought")
	}
	if got := attrs["ct.history_length"].AsInt64(); got != 1 {
		t.Errorf("ct.history_length = %d, want 1", got)
	}
	if got := attrs["ct.episode_id"].AsString(); got != "default" {
		t.Errorf("ct.episode_id = %q, want default", got)
	}
}

// TestSessionAndEvictionCounters drives 65 distinct episodes directly through
// the state machine (bypassing HTTP) to trigger exactly one LRU eviction
// (defaultMaxEpisodes is 64), then asserts both unattributed counters wired
// in newMCPServer.
func TestSessionAndEvictionCounters(t *testing.T) {
	_, reader := setupTestTelemetry(t)

	state := thinking.NewServer()
	_ = newMCPServer(state) // wires OnEvict and counts one session

	// Drive 65 distinct episodes straight through the state machine: the
	// 65th exceeds defaultMaxEpisodes(64) and evicts one.
	yes := true
	for i := 1; i <= 65; i++ {
		_, err := state.ProcessThought(thinking.ThoughtData{
			Thought:           "t",
			ThoughtNumber:     1,
			TotalThoughts:     1,
			NextThoughtNeeded: &yes,
			Confidence:        0.5,
			Assumptions:       []string{},
			Critique:          "c",
			CounterArgument:   "ca",
			// Required because NextThoughtNeeded is true (schema.go Validate);
			// omitted in the task-5 brief's snippet, which made every call fail
			// validation silently (ProcessThought's Go-level err stays nil for
			// validation failures) and never created an episode to evict.
			NextStepRationale: "n",
			EpisodeID:         "ep-" + strconv.Itoa(i),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}

	created := findMetric(t, rm, "ct.sessions.created").Data.(metricdata.Sum[int64])
	if got := created.DataPoints[0].Value; got != 1 {
		t.Errorf("ct.sessions.created = %d, want 1", got)
	}
	evicted := findMetric(t, rm, "ct.episodes.evicted").Data.(metricdata.Sum[int64])
	if got := evicted.DataPoints[0].Value; got != 1 {
		t.Errorf("ct.episodes.evicted = %d, want 1", got)
	}
}

// TestSpansNeverContainReasoningContent enforces the issue #76 hard rule:
// thought/critique/counterArgument/assumptions/nextStepRationale text must
// never reach telemetry.
func TestSpansNeverContainReasoningContent(t *testing.T) {
	exp, _ := setupTestTelemetry(t)
	ts := newTelemetryTestServer(t)

	const sentinel = "SECRET-REASONING-CONTENT-9c4f"
	yes := true
	client := newHTTPClient(t, ts.URL)
	client.callTool(t, thinking.ThoughtData{
		Thought:           sentinel + " thought",
		ThoughtNumber:     1,
		TotalThoughts:     2,
		NextThoughtNeeded: &yes,
		Confidence:        0.5,
		Assumptions:       []string{sentinel + " assumption"},
		Critique:          sentinel + " critique",
		CounterArgument:   sentinel + " counter",
		NextStepRationale: sentinel + " rationale",
	})

	for _, s := range exp.GetSpans() {
		for _, kv := range s.Attributes {
			if strings.Contains(string(kv.Key), sentinel) || strings.Contains(kv.Value.String(), sentinel) {
				t.Errorf("span %q attribute %q leaks reasoning content: %s", s.Name, kv.Key, kv.Value.String())
			}
		}
		for _, ev := range s.Events {
			for _, kv := range ev.Attributes {
				if strings.Contains(kv.Value.String(), sentinel) {
					t.Errorf("span %q event attribute %q leaks reasoning content", s.Name, kv.Key)
				}
			}
		}
	}
}
