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
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jacaudi/critical-thinking/internal/thinking"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const shutdownGrace = 10 * time.Second

// runStdio serves one MCP session over stdin/stdout. The server holds no
// state, so the single process-wide *mcp.Server is the whole story.
func runStdio() error {
	srv := newMCPServer()

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

// runHTTP serves Streamable HTTP in the SDK's stateless mode (MCP protocol
// 2026-07-28): no Mcp-Session-Id, no per-connection state, every POST is a
// complete unit of work. See buildHTTPHandler.
func runHTTP(cfg httpConfig, addr string) error {
	if err := cfg.validateAuth(); err != nil {
		return err // fail fast: never bind a port with auth misconfigured
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	var verifier *oidc.IDTokenVerifier
	if cfg.OIDCIssuer != "" {
		v, err := newOIDCVerifier(ctx, cfg.OIDCIssuer, cfg.OIDCAudience)
		if err != nil {
			return err
		}
		verifier = v
		slog.Info("OIDC authentication ENABLED", "issuer", cfg.OIDCIssuer, "audience", cfg.OIDCAudience)
	} else {
		slog.Warn("OIDC authentication DISABLED (CTHINK_OIDC_ISSUER not set); /mcp is unauthenticated")
	}

	rootHandler, err := buildHTTPHandler(cfg, verifier)
	if err != nil {
		return err
	}

	// addr like ":3000" already includes the colon; combine with the configured host.
	listenAddr := cfg.HTTPHost + addr

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           otelHTTPHandler(rootHandler),
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

// buildHTTPHandler assembles the HTTP handler chain and is the single source
// of truth for the /mcp + /health wiring, used by runHTTP and the tests.
//
// One *mcp.Server serves every request. The CSRF layer is Go's
// http.CrossOriginProtection wrapping the MCP handler (the SDK's own option
// has been deprecated since v1.6.1). StreamableHTTPOptions.Stateless makes
// the SDK ignore Mcp-Session-Id entirely, answer GET and DELETE with 405, and
// accept MCP protocol 2026-07-28 (the sessionless model); older clients still
// get their initialize answered by a per-request temporary session. Request
// bodies are capped at the SDK default (4 MiB → 413).
//
// /mcp is wrapped by requireAuth iff verifier != nil; /health is always bare;
// withCORS is the outermost layer this function builds; runHTTP wraps the
// result in otelHTTPHandler.
func buildHTTPHandler(cfg httpConfig, verifier *oidc.IDTokenVerifier) (http.Handler, error) {
	// Register the configured browser origins (CTHINK_ALLOWED_ORIGINS) with
	// Go's CrossOriginProtection, the CSRF layer that wraps the MCP handler
	// below. withCORS (outermost) is a separate, stricter allow-list: it
	// rejects any Origin not in the list, including same-origin requests that
	// CrossOriginProtection alone would permit. Non-browser callers (no Origin,
	// no Sec-Fetch-Site) pass both.
	csrf := http.NewCrossOriginProtection()
	for _, o := range cfg.AllowedOrigins {
		if err := csrf.AddTrustedOrigin(o); err != nil {
			return nil, fmt.Errorf("invalid CTHINK_ALLOWED_ORIGINS entry %q: %w", o, err)
		}
	}

	// The SDK deprecated its CrossOriginProtection option in v1.6.1 in favour
	// of wrapping the handler. With the option unset the SDK applies no origin
	// check of its own, so this wrap is the whole CSRF layer. (Unless
	// MCPGODEBUG=enableoriginverification=1 is set, which makes the SDK add a
	// zero-trust check of its own; do not set it.)
	srv := newMCPServer()
	handler := csrf.Handler(mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true}))

	mcpEndpoint := handler
	if verifier != nil {
		mcpEndpoint = requireAuth(verifier, handler)
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpEndpoint)
	mux.HandleFunc("/health", healthHandler)
	return withCORS(mux, cfg.AllowedOrigins), nil
}

// otelHTTPHandler wraps the fully-composed HTTP handler (from buildHTTPHandler,
// which already applies withCORS as its outermost layer) in otelhttp, keeping
// otelhttp OUTERMOST so CORS/CSRF rejections are traced too. /health is filtered
// from telemetry — liveness probes would otherwise emit a span every few seconds
// forever.
func otelHTTPHandler(inner http.Handler) http.Handler {
	// otelhttp v0.69.0 ignores the operation arg ("mcp") for span naming — its
	// default names spans from semconv SpanName (method, or method+route, e.g.
	// "POST /mcp"). WithSpanNameFormatter forces the stable "mcp" name.
	return otelhttp.NewHandler(inner, "mcp",
		otelhttp.WithSpanNameFormatter(func(string, *http.Request) string { return "mcp" }),
		otelhttp.WithFilter(func(r *http.Request) bool { return r.URL.Path != "/health" }))
}

// newMCPServer constructs the *mcp.Server with the criticalthinking tool
// registered. The tool handler is a pure function, so one server is shared by
// every session and every request, and the ReadOnlyHint and OpenWorldHint
// annotations below are accurate (IdempotentHint and DestructiveHint are
// spec-inert once ReadOnlyHint is set).
func newMCPServer() *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "critical-thinking",
		Version: version,
	}, nil)
	srv.AddReceivingMiddleware(otelMiddleware())

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "criticalthinking",
		Description: thinking.ToolDescription,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}, handleThought)

	return srv
}

// handleThought adapts thinking.Process to the SDK's typed tool handler. The
// second return value becomes structuredContent.
func handleThought(ctx context.Context, _ *mcp.CallToolRequest, td thinking.ThoughtData) (*mcp.CallToolResult, any, error) {
	// Bounded domain attributes only — reasoning text never reaches telemetry.
	// Attributes reflect the request as sent; ct.total_thoughts is pre-clamp.
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.Int("ct.thought_number", td.ThoughtNumber),
		attribute.Int("ct.total_thoughts", td.TotalThoughts),
		attribute.Float64("ct.confidence", td.Confidence),
		attribute.Bool("ct.is_revision", td.IsRevision),
		attribute.Bool("ct.is_branch", td.BranchID != ""),
	)

	res := thinking.Process(td)
	out := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: res.Text}},
		IsError: res.IsError,
	}
	if res.IsError {
		// Return an untyped nil: with Out = any the SDK's typed-nil guard does
		// not run, and a typed nil would serialise as "structuredContent": null.
		return out, nil, nil
	}
	return out, res.Structured, nil
}

// withCORS gates browser access via the configured allowed-origins list
// (CTHINK_ALLOWED_ORIGINS). Empty means no browser origins allowed.
//
// When an origin matches:
//   - Access-Control-Allow-Origin: <origin>
//   - Access-Control-Allow-Credentials: true
//   - Vary: Origin                                   (cache-poisoning mitigation)
//
// The server is stateless, so only POST (and the OPTIONS preflight) are
// advertised. Authorization is allowed so browser clients can present OIDC
// bearer tokens; Mcp-Method and Mcp-Name are the per-request headers MCP
// protocol 2026-07-28 requires; mcp-session-id stays allowed so preflights
// from older clients that still send it succeed (the server ignores it).
//
// Non-browser callers (no Origin header) bypass the check entirely.
func withCORS(h http.Handler, allowed []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Add("Vary", "Origin")
			if !slices.Contains(allowed, origin) {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Protocol-Version, Mcp-Method, Mcp-Name, mcp-session-id")

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

// healthHandler reports liveness. The server keeps no sessions, so there is
// nothing to count beyond "up" and which build this is.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	body := struct {
		Status    string `json:"status"`
		Transport string `json:"transport"`
		Version   string `json:"version"`
	}{
		Status:    "ok",
		Transport: "streamable-http",
		Version:   version,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
