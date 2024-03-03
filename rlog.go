package rlog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type RawTextHandler struct {
	mu     *sync.Mutex
	writer io.Writer
	attrs  []slog.Attr
	opts   slog.HandlerOptions
}

type Parenthesis int

const (
	None  Parenthesis = 0b00
	Left  Parenthesis = 0b01
	Right Parenthesis = 0b10
	Both  Parenthesis = 0b11
)

func NewRawTextHandler(w io.Writer, opts *slog.HandlerOptions) *RawTextHandler {
	h := &RawTextHandler{
		mu:     &sync.Mutex{},
		writer: w,
	}

	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}

	return h
}

func (h *RawTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *RawTextHandler) printAttr(buf io.Writer, attr slog.Attr, paren Parenthesis) error {
	if paren&Left != 0 {
		_, err := fmt.Fprint(buf, "(")
		if err != nil {
			return err
		}
	}

	switch attr.Value.Kind() {
	case slog.KindGroup:
		attrs := attr.Value.Group()
		_, err := fmt.Fprintf(buf, "%v=", attr.Key)
		if err != nil {
			return err
		}
		for i, a := range attrs {
			p := None
			if i == 0 {
				p |= Left
			}
			if i == len(attrs)-1 {
				p |= Right
			}
			err = h.printAttr(buf, a, p)
			if err != nil {
				break
			}
		}
	default:
		fmtString := ", %v=%v"
		if paren&Left != 0 {
			fmtString = fmtString[2:]
		}
		_, err := fmt.Fprintf(buf, fmtString, attr.Key, attr.Value)
		if err != nil {
			break
		}
	}

	if paren&Right != 0 {
		_, err := fmt.Fprint(buf, ")")
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *RawTextHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	if h.opts.AddSource {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := frames.Next()
		_, err := fmt.Fprintf(buf, "%s %s %s:%d %s",
			r.Time.Format("2006-01-02T15:04:05.999Z07:00"), r.Level, filepath.Base(frame.File), frame.Line, r.Message)
		if err != nil {
			return err
		}
	} else {
		_, err := fmt.Fprintf(buf, "%s %s %s",
			r.Time.Format("2006-01-02T15:04:05.999Z07:00"), r.Level, r.Message)
		if err != nil {
			return err
		}
	}

	r.AddAttrs(h.attrs...)

	count := 0
	r.Attrs(func(a slog.Attr) bool {
		paren := None
		if count == 0 {
			paren |= Left
			_, err := fmt.Fprint(buf, " ")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return false
			}
		}
		if count == r.NumAttrs()-1 {
			paren |= Right
		}

		err := h.printAttr(buf, a, paren)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}
		count += 1

		return true
	})
	_, err := fmt.Fprint(buf, "\n")
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	n, err := h.writer.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if n != len(buf.Bytes()) {
		return fmt.Errorf("incomplete log write. (expected=%d, actual=%d)", len(buf.Bytes()), n)
	}

	return nil
}

func (h *RawTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewRawTextHandler(h.writer, nil)
}

func (h *RawTextHandler) WithGroup(name string) slog.Handler {
	return h
}
