package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jacaudi/critical-thinking/internal/thinking"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// legacyInitialize is a protocol-2025-11-25 initialize request: what a client
// that predates the sessionless model sends first.
const legacyInitialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`

// newTestServerWith serves exactly what runHTTP serves for the given config
// and verifier: the buildHTTPHandler chain wrapped in otelHTTPHandler.
func newTestServerWith(t *testing.T, cfg httpConfig, verifier *oidc.IDTokenVerifier) *httptest.Server {
	t.Helper()
	h, err := buildHTTPHandler(cfg, verifier)
	if err != nil {
		t.Fatalf("buildHTTPHandler: %v", err)
	}
	ts := httptest.NewServer(otelHTTPHandler(h))
	t.Cleanup(ts.Close)
	return ts
}

// newTestServer is newTestServerWith for the default config, auth disabled.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newTestServerWith(t, httpConfig{}, nil)
}

// newTestClient connects with the SDK client — the same code real hosts embed.
// Against a stateless v1.7.0 server it negotiates the sessionless protocol
// (server/discover, no initialize) automatically.
func newTestClient(t *testing.T, base string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: base + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// callToolErr issues one tools/call and decodes structuredContent. It returns
// an error instead of failing the test so it is safe from goroutines.
func callToolErr(cs *mcp.ClientSession, td thinking.ThoughtData) (thinking.ThoughtResponse, error) {
	var out thinking.ThoughtResponse
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "criticalthinking", Arguments: td})
	if err != nil {
		return out, err
	}
	if res.IsError {
		raw, _ := json.Marshal(res.Content)
		return out, fmt.Errorf("tool returned isError=true: %s", raw)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

func callTool(t *testing.T, cs *mcp.ClientSession, td thinking.ThoughtData) thinking.ThoughtResponse {
	t.Helper()
	out, err := callToolErr(cs, td)
	if err != nil {
		t.Fatalf("tools/call: %v", err)
	}
	return out
}

// validInputN is the integration-test analog of the package's validInput;
// defined here to avoid importing test-only helpers across packages.
func validInputN(num int, tag string) thinking.ThoughtData {
	return thinking.ThoughtData{
		Thought:           tag + " thought " + strconv.Itoa(num),
		ThoughtNumber:     num,
		TotalThoughts:     20,
		NextThoughtNeeded: new(true),
		Confidence:        0.5,
		Assumptions:       []string{},
		Critique:          "narrow",
		CounterArgument:   "alternative",
		NextStepRationale: "next",
	}
}

func TestCORSDefaultRejectsBrowser(t *testing.T) {
	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), []string{"https://app.example", "https://other.example"})

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Origin", "https://app.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Errorf("Allow-Origin = %q, want https://app.example", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "" {
		t.Errorf("Expose-Headers = %q, want none (no session id to expose)", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "POST, OPTIONS" {
		t.Errorf("Allow-Methods = %q, want POST, OPTIONS", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") || !strings.Contains(got, "Mcp-Method") || !strings.Contains(got, "Mcp-Name") {
		t.Errorf("Allow-Headers = %q, must include Authorization for browser OIDC clients", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin", got)
	}
}

func TestCORSAllowsNoOrigin(t *testing.T) {
	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for no-Origin request, got %d", rec.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["status"] != "ok" || body["transport"] != "streamable-http" {
		t.Errorf("body = %v", body)
	}
	if v, _ := body["version"].(string); v == "" {
		t.Errorf("version is empty")
	}
	if _, stale := body["sessionsCreated"]; stale {
		t.Errorf("sessionsCreated must be gone from /health: %v", body)
	}
	if len(body) != 3 {
		t.Errorf("/health must carry exactly status, transport, version; got %v", body)
	}
}

// TestToolAnnotationsAreTruthful pins what every MCP host sees in tools/list:
// a pure function is read-only, idempotent, non-destructive, and closed-world.
func TestToolAnnotationsAreTruthful(t *testing.T) {
	ts := newTestServer(t)
	cs := newTestClient(t, ts.URL)
	res, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name != "criticalthinking" {
			continue
		}
		a := tool.Annotations
		if a == nil || !a.ReadOnlyHint || !a.IdempotentHint ||
			a.DestructiveHint == nil || *a.DestructiveHint ||
			a.OpenWorldHint == nil || *a.OpenWorldHint {
			t.Errorf("annotations = %+v, want read-only, idempotent, non-destructive, closed-world", a)
		}
		return
	}
	t.Fatal("criticalthinking not listed")
}

// TestStatelessInterleavedClients proves two clients sharing one server see
// only their own calls, and that the SDK client negotiates the sessionless
// protocol (2026-07-28) against this server.
func TestStatelessInterleavedClients(t *testing.T) {
	ts := newTestServer(t)
	a := newTestClient(t, ts.URL)
	b := newTestClient(t, ts.URL)

	for _, c := range []struct {
		name string
		cs   *mcp.ClientSession
	}{{"A", a}, {"B", b}} {
		if got := c.cs.InitializeResult().ProtocolVersion; got != "2026-07-28" {
			t.Errorf("client %s negotiated protocol %q, want 2026-07-28", c.name, got)
		}
	}

	const N = 10
	errs := make(chan error, 2*N)
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 1; i <= N; i++ {
			td := validInputN(i, "A")
			td.Confidence = 0.2
			resp, err := callToolErr(a, td)
			if err != nil {
				errs <- err
				continue
			}
			if resp.ThoughtNumber != i || resp.Confidence != 0.2 || resp.TotalThoughts != 20 {
				errs <- fmt.Errorf("A: got %+v", resp)
			}
		}
	})
	wg.Go(func() {
		for i := 1; i <= N; i++ {
			td := validInputN(i, "B")
			td.Confidence = 0.8
			td.TotalThoughts = 5 // exercises the clamp on B only
			resp, err := callToolErr(b, td)
			if err != nil {
				errs <- err
				continue
			}
			wantTotal := max(5, i)
			if resp.ThoughtNumber != i || resp.Confidence != 0.8 || resp.TotalThoughts != wantTotal {
				errs <- fmt.Errorf("B: got %+v", resp)
			}
		}
	})
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// TestStatelessWire pins the HTTP facts of go-sdk v1.7.0 stateless mode that
// the docs promise: only POST is served, session ids are neither issued nor
// required, and a stale one is ignored rather than rejected.
func TestStatelessWire(t *testing.T) {
	ts := newTestServer(t)

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		req, _ := http.NewRequest(method, ts.URL+"/mcp", nil)
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("mcp-session-id", "stale")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s /mcp status = %d, want 405", method, resp.StatusCode)
		}
		if got := resp.Header.Get("Allow"); got != "POST" {
			t.Errorf("%s /mcp Allow = %q, want POST", method, got)
		}
	}

	// A legacy initialize with a stale session id must still be answered, and
	// no session id may come back.
	resp, body := rawPost(t, ts.URL, legacyInitialize, "stale")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("initialize with stale session id: status %d, body %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("mcp-session-id"); got != "" {
		t.Errorf("stateless server issued Mcp-Session-Id %q", got)
	}
	if !strings.Contains(extractFirstJSON(body), `"serverInfo"`) {
		t.Errorf("initialize response missing serverInfo: %s", body)
	}
}

// TestCSRFLayerRejectsCrossSiteWithoutOrigin proves the http.CrossOriginProtection
// wrap inside buildHTTPHandler is live: a cross-site POST that carries no Origin
// header slips past withCORS (which only inspects Origin) and must be stopped by
// the CSRF layer with the stdlib's own 403.
func TestCSRFLayerRejectsCrossSiteWithoutOrigin(t *testing.T) {
	ts := newTestServer(t)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site POST without Origin: status %d, want 403; body %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "cross-origin request detected") {
		t.Errorf("expected the CrossOriginProtection rejection body, got %s", body)
	}
}

// TestErrorResultOmitsStructuredContent guards against a typed-nil
// structuredContent: an error result must carry no structuredContent key at
// all (not "structuredContent": null).
func TestErrorResultOmitsStructuredContent(t *testing.T) {
	ts := newTestServer(t)
	bad := validInputN(1, "bad")
	bad.Critique = ""
	args, _ := json.Marshal(bad)
	call := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"criticalthinking","arguments":` + string(args) + `}}`
	resp, body := rawPost(t, ts.URL, call, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tools/call status = %d, body %s", resp.StatusCode, body)
	}
	env := extractFirstJSON(body)
	if !strings.Contains(env, `"isError":true`) {
		t.Fatalf("expected an isError result: %s", env)
	}
	if strings.Contains(env, `"structuredContent"`) {
		t.Errorf("error result must not carry structuredContent: %s", env)
	}
	// The SDK re-encodes the tool's text content as a quoted JSON string, so
	// the hint key's quotes come back escaped (\"hint\":) rather than as the
	// literal bytes "hint": — match on the word plus the checklist's known
	// lead-in instead of the exact JSON-key bytes.
	if !strings.Contains(env, `\"hint\":`) || !strings.Contains(env, "Every call requires") {
		t.Errorf("error result must carry the hint checklist: %s", env)
	}
}

// rawPost sends one JSON-RPC body to /mcp; sessionID, when non-empty, is sent
// as a (stale) mcp-session-id header.
func rawPost(t *testing.T, base, payload, sessionID string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, base+"/mcp", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("mcp-session-id", sessionID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

// extractFirstJSON returns the first JSON-RPC envelope from a server response,
// handling both plain JSON bodies and Server-Sent Event framings.
func extractFirstJSON(body string) string {
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "{") {
		return trimmed
	}
	for line := range strings.SplitSeq(body, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimRight(line, "\r"), "data: "); ok {
			return rest
		}
	}
	return body
}

func TestHTTPAuthEnabledGatesMCP(t *testing.T) {
	idp := newFakeIdP(t)
	verifier, err := newOIDCVerifier(t.Context(), idp.issuer(), "critical-thinking")
	if err != nil {
		t.Fatalf("newOIDCVerifier: %v", err)
	}
	ts := newTestServerWith(t, httpConfig{}, verifier)

	// /mcp without a token → 401
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/mcp", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("/mcp unauthenticated status = %d, want 401", resp.StatusCode)
	}

	// /health without a token → 200 (probes must never be gated)
	hresp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	_ = hresp.Body.Close()
	if hresp.StatusCode != http.StatusOK {
		t.Errorf("/health status = %d, want 200 (must stay unauthenticated)", hresp.StatusCode)
	}
}

func TestHTTPAuthDisabledLeavesMCPOpen(t *testing.T) {
	// verifier nil = auth disabled: a full tools/call must succeed without a token.
	ts := newTestServer(t)
	cs := newTestClient(t, ts.URL) // no Authorization header anywhere
	if resp := callTool(t, cs, validInputN(1, "open")); resp.ThoughtNumber != 1 {
		t.Errorf("unauthenticated tools/call returned %+v", resp)
	}
}

func TestHTTPAuthOptionsBypass(t *testing.T) {
	// OPTIONS preflight must short-circuit in withCORS before auth, returning 200 without a token.
	idp := newFakeIdP(t)
	verifier, err := newOIDCVerifier(t.Context(), idp.issuer(), "critical-thinking")
	if err != nil {
		t.Fatalf("newOIDCVerifier: %v", err)
	}
	ts := newTestServerWith(t, httpConfig{}, verifier)

	req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/mcp", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("OPTIONS /mcp status = %d, want 200 (preflight must bypass auth)", resp.StatusCode)
	}
}

// browserPost sends a cross-site browser POST (Origin + Sec-Fetch-Site) so
// both the CORS layer and the CSRF layer get to vote.
func browserPost(t *testing.T, base, origin, payload string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, base+"/mcp", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Origin", origin)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp
}

// TestBuildHTTPHandlerRejectsInvalidOrigin covers buildHTTPHandler's error
// return: an AllowedOrigins entry that csrf.AddTrustedOrigin itself rejects
// (missing scheme) must fail fast with that reason wrapped in, rather than
// binding a port with the origin silently dropped.
func TestBuildHTTPHandlerRejectsInvalidOrigin(t *testing.T) {
	_, err := buildHTTPHandler(httpConfig{AllowedOrigins: []string{"not-a-url"}}, nil)
	if err == nil {
		t.Fatal("expected an error for an invalid AllowedOrigins entry, got nil")
	}
	if !strings.Contains(err.Error(), "not-a-url") {
		t.Errorf("error = %q, want it to name the offending origin", err)
	}
}

// TestTrustedOriginPassesBothLayers proves CTHINK_ALLOWED_ORIGINS reaches the
// CSRF layer (csrf.AddTrustedOrigin): a browser POST from a configured origin
// passes withCORS and http.CrossOriginProtection, while an unconfigured one
// is stopped. Without AddTrustedOrigin the trusted case would 403.
func TestTrustedOriginPassesBothLayers(t *testing.T) {
	ts := newTestServerWith(t, httpConfig{AllowedOrigins: []string{"https://app.example"}}, nil)

	if resp := browserPost(t, ts.URL, "https://app.example", legacyInitialize); resp.StatusCode != http.StatusOK {
		t.Errorf("trusted origin: status %d, want 200", resp.StatusCode)
	}
	if resp := browserPost(t, ts.URL, "https://evil.example", legacyInitialize); resp.StatusCode != http.StatusForbidden {
		t.Errorf("untrusted origin: status %d, want 403", resp.StatusCode)
	}
}

// TestPreflightWithOrigin exercises real browser preflights (OPTIONS with an
// Origin), which the auth bypass test does not send.
func TestPreflightWithOrigin(t *testing.T) {
	ts := newTestServerWith(t, httpConfig{AllowedOrigins: []string{"https://app.example"}}, nil)

	for _, tc := range []struct {
		origin string
		want   int
	}{
		{"https://app.example", http.StatusOK},
		{"https://evil.example", http.StatusForbidden},
	} {
		req, err := http.NewRequest(http.MethodOptions, ts.URL+"/mcp", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Errorf("preflight from %s: status %d, want %d", tc.origin, resp.StatusCode, tc.want)
			continue
		}
		if tc.want != http.StatusOK {
			continue
		}
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != tc.origin {
			t.Errorf("Allow-Origin = %q, want %q", got, tc.origin)
		}
		if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "POST, OPTIONS" {
			t.Errorf("Allow-Methods = %q", got)
		}
		if got := resp.Header.Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") || !strings.Contains(got, "Mcp-Method") || !strings.Contains(got, "Mcp-Name") {
			t.Errorf("Allow-Headers = %q, must include Authorization", got)
		}
	}
}

// TestOversizedBodyRejected pins the SDK's default 4 MiB request-body cap that
// configuration.md promises.
func TestOversizedBodyRejected(t *testing.T) {
	ts := newTestServer(t)
	body := bytes.Repeat([]byte("x"), 4<<20+1)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body: status %d, want 413", resp.StatusCode)
	}
}

// TestLegacyClientFullFlow: a protocol-2025-11-25 client's initialize,
// tools/list, and tools/call (still sending the deprecated inputs) all succeed
// against the stateless handler, each POST standing on its own.
func TestLegacyClientFullFlow(t *testing.T) {
	ts := newTestServer(t)

	if resp, body := rawPost(t, ts.URL, legacyInitialize, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("initialize: status %d, body %s", resp.StatusCode, body)
	}
	_, body := rawPost(t, ts.URL, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, "")
	if !strings.Contains(extractFirstJSON(body), `"name":"criticalthinking"`) {
		t.Fatalf("tools/list missing the tool: %s", body)
	}
	args, err := json.Marshal(validInputN(1, "legacy"))
	if err != nil {
		t.Fatal(err)
	}
	// A v1.15 client still sends the two fields this release deprecates.
	legacyArgs := strings.TrimSuffix(string(args), "}") + `,"episodeId":"old-client","needsMoreThoughts":true}`
	_, body = rawPost(t, ts.URL, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"criticalthinking","arguments":`+legacyArgs+`}}`, "")
	env := extractFirstJSON(body)
	if strings.Contains(env, `"isError":true`) {
		t.Fatalf("legacy call with deprecated inputs was rejected: %s", env)
	}
	if !strings.Contains(env, `"structuredContent"`) || strings.Contains(env, `"episodeId"`) {
		t.Errorf("expected a structured response without an episodeId echo: %s", env)
	}
}
