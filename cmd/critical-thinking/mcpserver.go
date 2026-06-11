package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jacaudi/critical-thinking/internal/thinking"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	idleTimeout   = 60 * time.Minute
	shutdownGrace = 10 * time.Second
)

// runStdio runs the server with one global SequentialThinkingServer instance.
// One process = one session, no cross-stream risk by definition.
func runStdio() error {
	state := thinking.NewServer()
	srv := newMCPServer(state)

	var transport mcp.Transport = &mcp.StdioTransport{}
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		// --verbose: trace every JSON-RPC frame to stderr (stdout is the protocol).
		transport = &mcp.LoggingTransport{Transport: transport, Writer: os.Stderr}
	}

	if err := srv.Run(context.Background(), transport); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// runHTTP starts a Streamable HTTP server. Each session gets its own
// *mcp.Server with its own SequentialThinkingServer, constructed inside the
// factory closure. There is no map keyed by session-id anywhere in this
// process — the closure scope is the cross-session isolation invariant.
//
// Idle-session lifecycle is delegated to the SDK via
// StreamableHTTPOptions.SessionTimeout: the SDK closes its own per-session
// state after idleTimeout of inactivity, releasing the bound *mcp.Server (and
// the *SequentialThinkingServer it captures) for GC.
//
// We keep a small in-process registry that counts every session ever created.
// The registry is NOT synchronized with the SDK's view of live sessions — once
// the SDK closes a session we have no callback, so the count drifts upward.
// /health exposes it as `sessionsCreated` to make the semantics explicit.
func runHTTP(cfg httpConfig, addr string) error {
	// Wire the configured allowed origins (CTHINK_ALLOWED_ORIGINS) into the SDK's
	// CSRF protection so browser clients from those origins aren't rejected by the
	// SDK's default same-origin policy. Non-browser callers (no Origin /
	// no Sec-Fetch-Site) are still allowed regardless.
	csrf := http.NewCrossOriginProtection()
	for _, o := range cfg.AllowedOrigins {
		if err := csrf.AddTrustedOrigin(o); err != nil {
			return fmt.Errorf("invalid CTHINK_ALLOWED_ORIGINS entry %q: %w", o, err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	registry := newSessionRegistry()

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		state := thinking.NewServer()
		registry.add(state)
		slog.Debug("http session created", "sessionsCreated", registry.count())
		return newMCPServer(state)
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout:        idleTimeout,
		CrossOriginProtection: csrf,
	})

	// addr like ":3000" already includes the colon; combine with the configured host.
	listenAddr := cfg.HTTPHost + addr

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.HandleFunc("/health", makeHealthHandler(registry))

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           withCORS(mux, cfg.AllowedOrigins),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
		}
	}()

	slog.Info("listening", "url", "http://"+listenAddr, "version", version)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

// newMCPServer constructs a configured *mcp.Server with the criticalthinking
// tool registered. The state argument is captured by the tool handler — this
// is how per-session isolation works in HTTP mode (each call to this function
// inside the StreamableHTTP factory closure produces a server bound to a fresh
// state). Stdio mode calls it once with a single global state.
func newMCPServer(state *thinking.SequentialThinkingServer) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "critical-thinking",
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

	srv.AddResource(&mcp.Resource{
		Name:        "thinking_current",
		Description: "Full thought history for the current session, including all critical-thinking fields (confidence, assumptions, critique, counterArgument).",
		URI:         "thinking://current",
		MIMEType:    "application/json",
	}, makeResourceHandler(state))

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

// makeResourceHandler closes over a per-session state and returns a
// ResourceHandler that always returns this session's snapshot, regardless of
// the requested URI. We deliberately do NOT support a thinking://sessions
// listing or thinking://{id} lookup — that would expose the existence of
// other sessions and violate the cross-session isolation invariant.
func makeResourceHandler(state *thinking.SequentialThinkingServer) func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		snap := state.Snapshot()
		body, err := json.MarshalIndent(snap, "", "  ")
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(body),
			}},
		}, nil
	}
}

// sessionRegistry counts every session ever created in this process. It holds
// only a monotonic counter — never the session states themselves — so closed
// sessions are not pinned: the factory closure that created a state holds the
// only live reference, and the SDK releases it on idle timeout for GC. Treat
// the count as a lifetime "sessions created" counter, not an "active right
// now" gauge.
type sessionRegistry struct {
	created atomic.Int64
}

func newSessionRegistry() *sessionRegistry { return &sessionRegistry{} }

// add records that a session was created. The *SequentialThinkingServer
// argument is intentionally not retained (that would pin closed-session state);
// it is kept in the signature so the call site reads naturally.
func (r *sessionRegistry) add(*thinking.SequentialThinkingServer) { r.created.Add(1) }

func (r *sessionRegistry) count() int { return int(r.created.Load()) }

// withCORS gates browser access via the configured allowed-origins list
// (CTHINK_ALLOWED_ORIGINS). Empty means no browser origins allowed.
//
// When an origin matches:
//   - Access-Control-Allow-Origin: <origin>
//   - Access-Control-Allow-Credentials: true
//   - Access-Control-Expose-Headers: mcp-session-id  (so JS clients can read it)
//   - Vary: Origin                                   (cache-poisoning mitigation)
//
// Non-browser callers (no Origin header) bypass the check entirely.
func withCORS(h http.Handler, allowed []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !slices.Contains(allowed, origin) {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "mcp-session-id")
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, mcp-session-id, MCP-Protocol-Version")

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

func makeHealthHandler(r *sessionRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		body := struct {
			Status          string `json:"status"`
			Transport       string `json:"transport"`
			SessionsCreated int    `json:"sessionsCreated"`
			Version         string `json:"version"`
		}{
			Status:          "ok",
			Transport:       "streamable-http",
			SessionsCreated: r.count(),
			Version:         version,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	}
}
