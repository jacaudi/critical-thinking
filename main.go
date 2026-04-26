package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

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
	// HTTP mode wired in Task 13.
	log.Fatal("HTTP mode not yet implemented")
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
