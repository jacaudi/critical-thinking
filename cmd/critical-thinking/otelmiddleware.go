package main

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// instrumentationScope names the tracer/meter for all telemetry this package
// emits. Single source of truth for the scope name.
const instrumentationScope = "github.com/jacaudi/critical-thinking"

// otelMiddleware returns an mcp.Middleware that wraps every received JSON-RPC
// method in a server span and records ct.mcp.calls / ct.mcp.duration. It is
// attached unconditionally in newMCPServer: when CTHINK_OTEL_ENABLED is false
// the otel globals are no-op providers and this is near-free.
//
// Privacy hard rule (#76): tool arguments are never inspected — only the tool
// NAME is read from CallToolParamsRaw. Reasoning content (thought, critique,
// counterArgument, assumptions, nextStepRationale) must never reach telemetry.
//
// Tracer, meter, and instruments are resolved here (per server construction,
// i.e. per session in HTTP mode) rather than at package init so they bind to
// whatever provider is installed at the time — the real one in serve, or a
// test's in-memory one. Duplicate instrument registration with an identical
// identity is defined by the OTel spec to return the same instrument.
func otelMiddleware() mcp.Middleware {
	tracer := otel.Tracer(instrumentationScope)
	meter := otel.Meter(instrumentationScope)
	calls, _ := meter.Int64Counter("ct.mcp.calls",
		metric.WithDescription("MCP JSON-RPC methods received, by method and outcome"))
	duration, _ := meter.Float64Histogram("ct.mcp.duration",
		metric.WithUnit("s"),
		metric.WithDescription("MCP JSON-RPC method handling duration"))

	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			attrs := []attribute.KeyValue{attribute.String("mcp.method", method)}
			if p, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok {
				attrs = append(attrs, attribute.String("mcp.tool.name", p.Name))
			}
			ctx, span := tracer.Start(ctx, "mcp."+method,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(attrs...))
			defer span.End()

			start := time.Now()
			res, err := next(ctx, method, req)

			outcome := "ok"
			switch {
			case err != nil:
				outcome = "error"
				span.SetStatus(codes.Error, err.Error())
			default:
				if ctr, ok := res.(*mcp.CallToolResult); ok && ctr.IsError {
					outcome = "tool_error"
				}
			}
			span.SetAttributes(attribute.String("ct.outcome", outcome))

			calls.Add(ctx, 1, metric.WithAttributes(
				attribute.String("mcp.method", method),
				attribute.String("ct.outcome", outcome)))
			duration.Record(ctx, time.Since(start).Seconds(),
				metric.WithAttributes(attribute.String("mcp.method", method)))
			return res, err
		}
	}
}
