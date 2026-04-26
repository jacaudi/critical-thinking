package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
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

// withCORS and makeHealthHandler are stubs filled in by Tasks 14 and 15.
func withCORS(h http.Handler) http.Handler { return h }

func makeHealthHandler(r *sessionRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}
