//go:build darwin

package logging

/*
#include <stdlib.h>

extern void MicDetectorLogInit(const char *subsystem, const char *category);
extern void MicDetectorLogDebug(const char *msg);
extern void MicDetectorLogDefault(const char *msg);
extern void MicDetectorLogError(const char *msg);
*/
import "C"

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unsafe"
)

// Handler is a slog.Handler that writes to Apple's unified logging system (os_log).
//
// Level mapping:
//
//	slog.Debug → OS_LOG_TYPE_DEBUG   (only captured when streaming with --level debug)
//	slog.Info  → OS_LOG_TYPE_DEFAULT (always stored and visible)
//	slog.Warn  → OS_LOG_TYPE_ERROR   (persisted longer)
//	slog.Error → OS_LOG_TYPE_ERROR   (persisted longer)
type Handler struct {
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

// NewHandler creates a Handler that logs to the given os_log subsystem and category.
// Messages below level are suppressed.
func NewHandler(subsystem, category string, level slog.Level) *Handler {
	cs := C.CString(subsystem)
	cc := C.CString(category)
	C.MicDetectorLogInit(cs, cc)
	C.free(unsafe.Pointer(cs))
	C.free(unsafe.Pointer(cc))
	return &Handler{level: level}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)

	prefix := groupPrefix(h.groups)

	for _, a := range h.attrs {
		writeAttr(&b, prefix, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&b, prefix, a)
		return true
	})

	msg := b.String()
	cm := C.CString(msg)
	defer C.free(unsafe.Pointer(cm))

	switch {
	case r.Level >= slog.LevelWarn:
		C.MicDetectorLogError(cm)
	case r.Level >= slog.LevelInfo:
		C.MicDetectorLogDefault(cm)
	default:
		C.MicDetectorLogDebug(cm)
	}

	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)
	return &Handler{level: h.level, attrs: newAttrs, groups: h.groups}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)
	return &Handler{level: h.level, attrs: h.attrs, groups: newGroups}
}

func groupPrefix(groups []string) string {
	if len(groups) == 0 {
		return ""
	}
	return strings.Join(groups, ".") + "."
}

func writeAttr(b *strings.Builder, prefix string, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}
	b.WriteByte(' ')
	b.WriteString(prefix)
	b.WriteString(a.Key)
	b.WriteByte('=')
	b.WriteString(fmt.Sprintf("%v", a.Value.Any()))
}
