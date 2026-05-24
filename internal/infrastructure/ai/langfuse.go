package ai

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	lfcallback "github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino/callbacks"
)

// LangfuseConfig holds Langfuse connection settings.
type LangfuseConfig struct {
	Host      string // e.g. "http://localhost:3100"
	PublicKey string // pk-lf-...
	SecretKey string // sk-lf-...
}

// langfuseHandler is the concrete Langfuse callback handler.
var langfuseHandler *lfcallback.CallbackHandler

// ── Context keys (package-private) ───────────────────────────────────

type presetTraceIDKey struct{} // pre-generated trace ID
type baseTraceOptsKey struct{} // middleware's base trace options

// InitLangfuse creates and registers a Langfuse callback handler wrapped
// with an autoTraceHandler that automatically sets trace-level Input/Output
// from the root Eino component call.
//
// Returns a flush function that MUST be called during graceful shutdown.
func InitLangfuse(cfg *LangfuseConfig) (flush func(), err error) {
	if cfg.PublicKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("langfuse: public key and secret key are required")
	}

	host := cfg.Host
	if host == "" {
		host = "https://cloud.langfuse.com"
	}

	handler, flusher := lfcallback.NewLangfuseHandler(&lfcallback.Config{
		Host:      host,
		PublicKey: cfg.PublicKey,
		SecretKey: cfg.SecretKey,
	})

	langfuseHandler = handler

	// Wrap with autoTraceHandler for automatic trace-level I/O,
	// then register globally so all Eino executions are auto-traced.
	wrapped := &autoTraceHandler{inner: handler}
	callbacks.AppendGlobalHandlers(wrapped)

	return flusher, nil
}

// PrepareTraceContext creates a trace-ready context with Langfuse metadata.
// Call from HTTP middleware or manually for background jobs.
//
// It pre-generates a trace ID and stores base trace options so the
// autoTraceHandler can merge them with the AI component's input.
//
// The handler layer does NOT need any trace-related code — the
// autoTraceHandler callback wrapper handles input/output automatically.
func PrepareTraceContext(ctx context.Context, name, userID, sessionID string) context.Context {
	traceID := generateTraceID()

	traceOpts := []lfcallback.TraceOption{
		lfcallback.WithID(traceID),
		lfcallback.WithName(name),
	}
	if userID != "" {
		traceOpts = append(traceOpts, lfcallback.WithUserID(userID))
	}
	if sessionID != "" {
		traceOpts = append(traceOpts, lfcallback.WithSessionID(sessionID))
	}

	ctx = context.WithValue(ctx, presetTraceIDKey{}, traceID)
	ctx = context.WithValue(ctx, baseTraceOptsKey{}, traceOpts)
	ctx = lfcallback.SetTrace(ctx, traceOpts...)

	return ctx
}

// ── Internal helpers ──────────────────────────────────────────────────

func generateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func marshalValue(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
