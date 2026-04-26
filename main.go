package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jacaudi/rubber-ducky-mcp/internal/thinking"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version is set at build time via -ldflags "-X main.version=...". The
// fallback "dev" identifies non-release builds.
var version = "dev"

var httpAddr = flag.String("http", "", "if set (e.g., \":3000\"), serve Streamable HTTP at this address; otherwise use stdio")

func main() {
	flag.Parse()

	if *httpAddr == "" {
		runStdio()
		return
	}
	runHTTP(*httpAddr)
}

// runStdio runs the server with one global SequentialThinkingServer instance.
// One process = one session, no cross-stream risk by definition.
func runStdio() {
	state := thinking.NewServer()
	srv := newMCPServer(state)

	transport := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
	if err := srv.Run(context.Background(), transport); err != nil {
		log.Printf("server failed: %v", err)
		os.Exit(1)
	}
}

// runHTTP starts a Streamable HTTP server. Each session gets its own
// *mcp.Server with its own SequentialThinkingServer, constructed inside the
// factory closure. There is no map keyed by session-id anywhere in this
// process — the closure scope is the cross-session isolation invariant.
//
// We do, however, keep a small in-process registry of *active* per-session
// states so the idle-cleanup goroutine (Task 16) can iterate them. The
// registry stores only the state pointers; cross-session access is impossible
// because the tool handler captures one state and never reads from the
// registry.
func runHTTP(addr string) {
	registry := newSessionRegistry()

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		state := thinking.NewServer()
		registry.add(state)
		return newMCPServer(state)
	}, nil)

	host := "127.0.0.1"
	if os.Getenv("DOCKER") == "true" {
		host = "0.0.0.0"
	}
	// addr like ":3000" already includes the colon; combine with host.
	listenAddr := host + addr

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.HandleFunc("/health", makeHealthHandler(registry))

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("rubber-ducky-thinking %s listening on http://%s", version, listenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

// newMCPServer constructs a configured *mcp.Server with the criticalthinking
// tool registered. The state argument is captured by the tool handler — this
// is how per-session isolation works in HTTP mode (each call to this function
// inside the StreamableHTTP factory closure produces a server bound to a fresh
// state). Stdio mode calls it once with a single global state.
func newMCPServer(state *thinking.SequentialThinkingServer) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "rubber-ducky-thinking",
		Version: version,
	}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "criticalthinking",
		Description: thinking.ToolDescription,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: ptrFalse(),
			IdempotentHint:  true,
			OpenWorldHint:   ptrFalse(),
		},
	}, makeToolHandler(state))

	return srv
}

func ptrFalse() *bool { f := false; return &f }

// makeToolHandler closes over a per-session state and returns the Go SDK's
// expected handler signature. The second return value (any) becomes the
// CallToolResult's structuredContent — we send the parsed ThoughtResponse.
func makeToolHandler(state *thinking.SequentialThinkingServer) func(context.Context, *mcp.CallToolRequest, thinking.ThoughtData) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args thinking.ThoughtData) (*mcp.CallToolResult, any, error) {
		res, err := state.ProcessThought(args)
		if err != nil {
			return nil, nil, err
		}

		callResult := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: res.Text}},
			IsError: res.IsError,
		}

		if res.IsError {
			return callResult, nil, nil
		}

		var structured thinking.ThoughtResponse
		if jsonErr := json.Unmarshal([]byte(res.StructuredJSON), &structured); jsonErr != nil {
			// Should not happen — ProcessThought just produced this JSON.
			return callResult, nil, nil
		}
		return callResult, structured, nil
	}
}

// sessionRegistry tracks active per-session states for idle cleanup. It does
// NOT mediate access to states — only the factory closure that created a state
// holds the reference used by the tool handler. The registry is read-only
// from the tool path's perspective.
type sessionRegistry struct {
	mu     sync.Mutex
	states []*thinking.SequentialThinkingServer
}

func newSessionRegistry() *sessionRegistry { return &sessionRegistry{} }

func (r *sessionRegistry) add(s *thinking.SequentialThinkingServer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states = append(r.states, s)
}

func (r *sessionRegistry) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.states)
}

// withCORS gates browser access via the ALLOWED_ORIGINS env var (comma-
// separated list). Default is empty — no browser origins allowed.
//
// When an origin matches:
//   - Access-Control-Allow-Origin: <origin>
//   - Access-Control-Allow-Credentials: true
//   - Access-Control-Expose-Headers: mcp-session-id  (so JS clients can read it)
//   - Vary: Origin                                   (cache-poisoning mitigation)
//
// Non-browser callers (no Origin header) bypass the check entirely.
func withCORS(h http.Handler) http.Handler {
	allowed := parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !contains(allowed, origin) {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "mcp-session-id")
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, mcp-session-id")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func parseAllowedOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func makeHealthHandler(r *sessionRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		body := struct {
			Status         string `json:"status"`
			Transport      string `json:"transport"`
			ActiveSessions int    `json:"activeSessions"`
			Version        string `json:"version"`
		}{
			Status:         "ok",
			Transport:      "streamable-http",
			ActiveSessions: r.count(),
			Version:        version,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	}
}
