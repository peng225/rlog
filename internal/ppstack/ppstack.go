package ppstack

import (
	"fmt"
	"io"
)

type ParenPrintStack struct {
	writer io.Writer
	depth  int
}

func NewParenPrintStack(buf io.Writer) *ParenPrintStack {
	return &ParenPrintStack{
		writer: buf,
		depth:  0,
	}
}

func (p *ParenPrintStack) Empty() bool {
	return p.depth == 0
}

func (p *ParenPrintStack) Push() error {
	_, err := fmt.Fprint(p.writer, "(")
	if err != nil {
		return err
	}
	p.depth += 1
	return nil
}

func (p *ParenPrintStack) Pop() error {
	if p.depth == 0 {
		return nil
	}
	_, err := fmt.Fprint(p.writer, ")")
	if err != nil {
		return err
	}
	p.depth -= 1
	return nil
}
