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

	"github.com/peng225/rlog/internal/ppstack"
)

type groupOrAttrs struct {
	group string
	attrs []slog.Attr
}

type HandlerOptions struct {
	AddSource bool
	Level     slog.Leveler
}

type RawTextHandler struct {
	mu     *sync.Mutex
	writer io.Writer
	opts   HandlerOptions
	goas   []groupOrAttrs
}

func NewRawTextHandler(w io.Writer, opts *HandlerOptions) *RawTextHandler {
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

func (h *RawTextHandler) printAttr(buf io.Writer, attr slog.Attr, pps *ppstack.ParenPrintStack, leftMost bool) error {
	// Resolve the Attr's value before doing anything else.
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return nil
	}

	switch attr.Value.Kind() {
	case slog.KindGroup:
		if !leftMost {
			_, err := fmt.Fprint(buf, ", ")
			if err != nil {
				return err
			}
		}
		attrs := attr.Value.Group()
		_, err := fmt.Fprintf(buf, "%v=", attr.Key)
		if err != nil {
			return err
		}
		err = pps.Push()
		if err != nil {
			return err
		}
		for i, a := range attrs {
			err = h.printAttr(buf, a, pps, i == 0)
			if err != nil {
				break
			}
		}
		err = pps.Pop()
		if err != nil {
			return err
		}
	default:
		fmtString := ", %v=%v"
		if leftMost {
			fmtString = fmtString[2:]
		}
		_, err := fmt.Fprintf(buf, fmtString, attr.Key, attr.Value)
		if err != nil {
			break
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

	goas := h.goas
	if r.NumAttrs() == 0 {
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}

	// Print the separator between the message and attributes.
	if len(goas) != 0 || r.NumAttrs() != 0 {
		_, err := fmt.Fprint(buf, " ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
	}

	pps := ppstack.NewParenPrintStack(buf)
	leftMost := true
	for _, goa := range goas {
		if goa.group != "" && len(goa.attrs) != 0 {
			err := fmt.Errorf("invalid group and attribute settings")
			fmt.Fprintln(os.Stderr, err)
			return err
		} else if goa.group != "" {
			prefix := ", "
			if leftMost {
				err := pps.Push()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return err
				}
				prefix = ""
			}
			_, err := buf.WriteString(prefix + goa.group + "=")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return err
			}
			leftMost = true
		} else if len(goa.attrs) != 0 {
			if leftMost {
				err := pps.Push()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return err
				}
			}
			for _, a := range goa.attrs {
				err := h.printAttr(buf, a, pps, leftMost)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return err
				}
				leftMost = false
			}
		}
	}

	r.Attrs(func(a slog.Attr) bool {
		if leftMost {
			err := pps.Push()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return false
			}
		}
		err := h.printAttr(buf, a, pps, leftMost)
		leftMost = false
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}
		return true
	})

	for !pps.Empty() {
		err := pps.Pop()
		if err != nil {
			return err
		}
	}

	_, err := fmt.Fprint(buf, "\n")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	n, err := h.writer.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if n != len(buf.Bytes()) {
		err := fmt.Errorf("incomplete log write. (expected=%d, actual=%d)", len(buf.Bytes()), n)
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	return nil
}

func (h *RawTextHandler) withGroupOrAttrs(goa groupOrAttrs) *RawTextHandler {
	newHandler := *h
	newHandler.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(newHandler.goas, h.goas)
	newHandler.goas[len(newHandler.goas)-1] = goa
	return &newHandler
}

func (h *RawTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

func (h *RawTextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}
