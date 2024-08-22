// Package cli provides utilities for nicer CLI output
package cli

import (
	"io"
	"strings"

	"github.com/muesli/reflow/ansi"
)

type IndentedWriter struct {
	indent     int
	ansiWriter *ansi.Writer
	skipIndent bool
	ansi       bool
}

func NewIndentedWriter(indent int, forward io.Writer) *IndentedWriter {
	return &IndentedWriter{
		indent: indent,
		ansiWriter: &ansi.Writer{
			Forward: forward,
		},
	}
}

// indentedWriter: io.Writer

func (w *IndentedWriter) Write(b []byte) (n int, err error) {
	// This method was adapted from the Writer.Write method in the indent package of the MIT-licensed
	// github.com/muesli/reflow project maintained by Christian Muehlhaeuser
	// (see https://github.com/muesli/reflow/blob/83f6379/indent/indent.go#L60). The method was
	// modified to properly indent after `\r` sequences.
	for _, c := range string(b) {
		switch {
		case c == '\x1B': // ANSI escape sequence
			w.ansi = true
		case w.ansi:
			if (c >= 0x41 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a) {
				// ANSI sequence terminated
				w.ansi = false
			}
		default:
			if !w.skipIndent {
				w.ansiWriter.ResetAnsi()
				_, err := w.ansiWriter.Write([]byte(makeIndentation(w.indent)))
				if err != nil {
					return 0, err
				}

				w.skipIndent = true
				w.ansiWriter.RestoreAnsi()
			}

			if c == '\n' || c == '\r' {
				// end of current line
				w.skipIndent = false
			}
		}

		_, err := w.ansiWriter.Write([]byte(string(c)))
		if err != nil {
			return 0, err
		}
	}

	return len(b), nil
}

func makeIndentation(indent int) string {
	const indentation = "  "
	return strings.Repeat(indentation, indent)
}
