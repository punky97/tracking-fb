package logger

import (
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
)

type hooked struct {
	zapcore.Core
	hooks []func(zapcore.Entry, map[string]interface{}) error
}

// registerCtxHooks wraps a Core and runs a collection of user-defined callback
// hooks each time a message is logged. Execution of the callbacks is blocking.
//
// This offers users an easy way to register simple callbacks (e.g., metrics
// collection) without implementing the full Core interface.
func registerCtxHooks(core zapcore.Core, hooks ...func(zapcore.Entry, map[string]interface{}) error) zapcore.Core {
	funcs := append([]func(zapcore.Entry, map[string]interface{}) error{}, hooks...)
	return &hooked{
		Core:  core,
		hooks: funcs,
	}
}

func (h *hooked) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// Let the wrapped Core decide whether to log this message or not. This
	// also gives the downstream a chance to register itself directly with the
	// CheckedEntry.
	if downstream := h.Core.Check(ent, ce); downstream != nil {
		return downstream.AddCore(ent, h)
	}
	return ce
}

func (h *hooked) With(fs []zapcore.Field) zapcore.Core {
	return &hooked{
		Core:  h.Core.With(fs),
		hooks: h.hooks,
	}
}

func (h *hooked) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	clone := h.cloneFields(fields)
	// Since our downstream had a chance to register itself directly with the
	// CheckedMessage, we don't need to call it here.
	var err error
	for i := range h.hooks {
		err = multierr.Append(err, h.hooks[i](ent, clone))
	}
	return err
}

func (h *hooked) cloneFields(fs []zapcore.Field) map[string]interface{} {
	m := make(map[string]interface{}, 0)
	// Add fields to an in-memory encoder.
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fs {
		f.AddTo(enc)
	}

	for k, v := range enc.Fields {
		m[k] = v
	}

	return m
}
