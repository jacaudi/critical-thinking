package main

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TestSetupOTelInstallsProviders: exporter construction is lazy (no network
// at setup time), so this passes with no collector running.
func TestSetupOTelInstallsProviders(t *testing.T) {
	prevTP := otel.GetTracerProvider()
	prevMP := otel.GetMeterProvider()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetMeterProvider(prevMP)
	})

	shutdown, err := setupOTel(context.Background())
	if err != nil {
		t.Fatalf("setupOTel: %v", err)
	}
	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Errorf("global tracer provider = %T, want *sdktrace.TracerProvider", otel.GetTracerProvider())
	}

	// Shutdown flushes against the (absent) default collector endpoint; a
	// flush error is expected and tolerated — it must return, not hang.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}

func TestServeSkipsOTelWhenDisabled(t *testing.T) {
	prev := otel.GetTracerProvider()
	c := newServeCmd(newConfigViper())
	c.stdioRun = func() error { return nil }
	c.httpRun = func(httpConfig, string) error { return nil }
	c.SetArgs([]string{})
	if err := c.Execute(); err != nil {
		t.Fatalf("serve: %v", err)
	}
	if otel.GetTracerProvider() != prev {
		t.Errorf("serve installed a tracer provider despite CTHINK_OTEL_ENABLED being unset")
	}
}

func TestServeInstallsOTelWhenEnabled(t *testing.T) {
	prevTP := otel.GetTracerProvider()
	prevMP := otel.GetMeterProvider()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetMeterProvider(prevMP)
	})
	t.Setenv("CTHINK_OTEL_ENABLED", "true")

	var sawProvider bool
	c := newServeCmd(newConfigViper())
	c.stdioRun = func() error {
		_, sawProvider = otel.GetTracerProvider().(*sdktrace.TracerProvider)
		return nil
	}
	c.httpRun = func(httpConfig, string) error { return nil }
	c.SetArgs([]string{})
	if err := c.Execute(); err != nil {
		t.Fatalf("serve: %v", err)
	}
	if !sawProvider {
		t.Error("serve did not install the SDK tracer provider before running the transport")
	}
}
