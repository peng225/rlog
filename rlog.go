package rlog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type rawTextHandler struct {
	mu    sync.Mutex
	w     io.Writer
	attrs []slog.Attr
}

type Parenthesis int

const (
	None  Parenthesis = 0b00
	Left  Parenthesis = 0b01
	Right Parenthesis = 0b10
	Both  Parenthesis = 0b11
)

type Option func(*rawTextHandler)

func NewRawTextHandler(attrs []slog.Attr, setters ...Option) slog.Handler {
	rth := &rawTextHandler{
		w:     os.Stdout,
		attrs: attrs,
	}
	for _, setter := range setters {
		setter(rth)
	}
	return rth
}

func WithWriter(w io.Writer) Option {
	return func(rth *rawTextHandler) {
		rth.w = w
	}
}

func (h *rawTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *rawTextHandler) printAttr(attr slog.Attr, paren Parenthesis) error {
	if paren&Left != 0 {
		_, err := fmt.Fprint(h.w, "(")
		if err != nil {
			return err
		}
	}

	switch v := attr.Value.Any().(type) {
	case []slog.Attr:
		_, err := fmt.Fprintf(h.w, "%v=", attr.Key)
		if err != nil {
			return err
		}
		for i, a := range v {
			p := None
			if i == 0 {
				p |= Left
			}
			if i == len(v)-1 {
				p |= Right
			}
			err = h.printAttr(a, p)
			if err != nil {
				break
			}
		}
	default:
		fmtString := ", %v=%v"
		if paren&Left != 0 {
			fmtString = fmtString[2:]
		}
		_, err := fmt.Fprintf(h.w, fmtString, attr.Key, attr.Value)
		if err != nil {
			break
		}
	}

	if paren&Right != 0 {
		_, err := fmt.Fprint(h.w, ")")
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *rawTextHandler) Handle(ctx context.Context, r slog.Record) error {
	frames := runtime.CallersFrames([]uintptr{r.PC})
	frame, _ := frames.Next()
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := fmt.Fprintf(h.w, "%s %s %s:%d %s",
		r.Time.Format("2006-01-02T15:04:05.999Z07:00"), r.Level, filepath.Base(frame.File), frame.Line, r.Message)
	if err != nil {
		return err
	}

	r.AddAttrs(h.attrs...)

	count := 0
	r.Attrs(func(a slog.Attr) bool {
		paren := None
		if count == 0 {
			paren |= Left
			_, err := fmt.Fprint(h.w, " ")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return false
			}
		}
		if count == r.NumAttrs()-1 {
			paren |= Right
		}

		err = h.printAttr(a, paren)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}
		count += 1

		return true
	})
	_, err = fmt.Fprintf(h.w, "\n")
	if err != nil {
		return err
	}
	return nil
}

func (h *rawTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewRawTextHandler(attrs)
}

func (h *rawTextHandler) WithGroup(name string) slog.Handler {
	return h
}
