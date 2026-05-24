package ai

import (
	"context"
	"log"

	lfcallback "github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

// autoTraceHandler wraps the eino-ext Langfuse CallbackHandler to
// automatically set trace-level Input/Output from the root Eino
// component call (ChatModel, Graph, Agent, Retriever, etc.).
//
// It is self-contained: if PrepareTraceContext was called (e.g. by
// HTTP middleware), the trace inherits HTTP metadata (name, user,
// session). If not, it auto-generates a trace ID and uses the Eino
// component name as the trace name.
//
// Depth tracking uses a context value that increments on OnStart and
// lets OnEnd identify the root exit.
type autoTraceHandler struct {
	inner *lfcallback.CallbackHandler
}

// ── Context key for depth tracking ───────────────────────────────────

type traceDepthKey struct{}

func getTraceDepth(ctx context.Context) int {
	if v, ok := ctx.Value(traceDepthKey{}).(int); ok {
		return v
	}
	return 0
}

// getComponentName extracts a human-readable name from RunInfo.
func getComponentName(info *callbacks.RunInfo) string {
	if info == nil {
		return "unknown"
	}
	if info.Name != "" {
		return info.Name
	}
	return info.Type + string(info.Component)
}

// ── callbacks.Handler implementation ────────────────────────────────

func (h *autoTraceHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	depth := getTraceDepth(ctx)
	ctx = context.WithValue(ctx, traceDepthKey{}, depth+1)

	// Root-level: inject input into trace options before the inner
	// handler creates the trace (so trace.Input is set at creation time).
	if depth == 0 {
		if baseOpts, ok := ctx.Value(baseTraceOptsKey{}).([]lfcallback.TraceOption); ok {
			// PrepareTraceContext was called (HTTP middleware) —
			// merge existing metadata with component input.
			merged := make([]lfcallback.TraceOption, len(baseOpts))
			copy(merged, baseOpts)
			merged = append(merged, lfcallback.WithInput(marshalValue(input)))
			ctx = lfcallback.SetTrace(ctx, merged...)
		} else {
			// No PrepareTraceContext (background job, test, etc.) —
			// auto-generate trace ID and use component name.
			traceID := generateTraceID()
			ctx = context.WithValue(ctx, presetTraceIDKey{}, traceID)
			ctx = lfcallback.SetTrace(ctx,
				lfcallback.WithID(traceID),
				lfcallback.WithName(getComponentName(info)),
				lfcallback.WithInput(marshalValue(input)),
			)
		}
	}

	return h.inner.OnStart(ctx, info, input)
}

func (h *autoTraceHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	depth := getTraceDepth(ctx)
	ctx = h.inner.OnEnd(ctx, info, output)

	// Root-level exit (depth==1 means we entered at 0, incremented to 1).
	if depth == 1 {
		if traceID, ok := ctx.Value(presetTraceIDKey{}).(string); ok && traceID != "" {
			h.inner.UpdateTraceOutput(ctx, traceID, marshalValue(output))
		}
	}

	return ctx
}

func (h *autoTraceHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	return h.inner.OnError(ctx, info, err)
}

func (h *autoTraceHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	depth := getTraceDepth(ctx)
	ctx = context.WithValue(ctx, traceDepthKey{}, depth+1)

	// For streaming input, ensure trace is still created with a proper ID.
	if depth == 0 {
		if _, ok := ctx.Value(baseTraceOptsKey{}).([]lfcallback.TraceOption); !ok {
			traceID := generateTraceID()
			ctx = context.WithValue(ctx, presetTraceIDKey{}, traceID)
			ctx = lfcallback.SetTrace(ctx,
				lfcallback.WithID(traceID),
				lfcallback.WithName(getComponentName(info)),
			)
		}
		log.Printf("[langfuse] streaming input at root level — trace input will be in observation only")
	}

	return h.inner.OnStartWithStreamInput(ctx, info, input)
}

func (h *autoTraceHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	depth := getTraceDepth(ctx)

	if depth == 1 {
		log.Printf("[langfuse] streaming output at root level — trace output will be in observation only")
	}

	return h.inner.OnEndWithStreamOutput(ctx, info, output)
}
