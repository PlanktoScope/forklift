// Package cli has shared utilities and application logic for the forklift CLI
package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	indentation = "  "
	bullet      = "- "
)

// Indented

func IndentedPrintf(indent int, format string, a ...any) {
	IndentedFprintf(indent, os.Stdout, format, a...)
}

func IndentedFprintf(indent int, w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, "%s%s", makeIndentation(indent), fmt.Sprintf(format, a...))
}

func makeIndentation(indent int) string {
	return strings.Repeat(indentation, indent)
}

func IndentedPrint(indent int, a ...any) {
	IndentedFprint(indent, os.Stdout, a...)
}

func IndentedFprint(indent int, w io.Writer, a ...any) {
	_, _ = fmt.Fprintf(w, "%s%s", makeIndentation(indent), fmt.Sprint(a...))
}

func IndentedPrintln(indent int, a ...any) {
	IndentedFprintln(indent, os.Stdout, a...)
}

func IndentedFprintln(indent int, w io.Writer, a ...any) {
	_, _ = fmt.Fprintf(w, "%s%s\n", makeIndentation(indent), fmt.Sprint(a...))
}

func IndentedPrintYaml(indent int, a any) error {
	return IndentedFprintYaml(indent, os.Stdout, a)
}

func IndentedFprintYaml(indent int, w io.Writer, a any) error {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(len(indentation))
	if err := encoder.Encode(a); err != nil {
		return errors.Wrapf(err, "couldn't serialize %T as yaml document", a)
	}
	if err := encoder.Close(); err != nil {
		return errors.Wrapf(
			err, "couldn't close yaml encoder after serializing %T as  yaml document", a,
		)
	}
	lines := strings.Split(buf.String(), "\n")
	for _, line := range lines[:len(lines)-1] { // last line follows last "\n" and is empty
		IndentedFprintln(indent, w, line)
	}
	return nil
}

// Bulleted

func BulletedPrintf(indent int, format string, a ...any) {
	BulletedFprintf(indent, os.Stdout, format, a...)
}

func BulletedFprintf(indent int, w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, "%s%s", makeBullet(indent), fmt.Sprintf(format, a...))
}

func makeBullet(indent int) string {
	return strings.Repeat(indentation, indent) + bullet
}

func BulletedPrint(indent int, a ...any) {
	BulletedFprint(indent, os.Stdout, a...)
}

func BulletedFprint(indent int, w io.Writer, a ...any) {
	_, _ = fmt.Fprintf(w, "%s%s", makeBullet(indent), fmt.Sprint(a...))
}

func BulletedPrintln(indent int, a ...any) {
	BulletedFprintln(indent, os.Stdout, a...)
}

func BulletedFprintln(indent int, w io.Writer, a ...any) {
	_, _ = fmt.Fprintf(w, "%s%s\n", makeBullet(indent), fmt.Sprint(a...))
}

func BulletedPrintYaml(indent int, a any) error {
	return BulletedFprintYaml(indent, os.Stdout, a)
}

func BulletedFprintYaml(indent int, w io.Writer, a any) error {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(len(indentation))
	if err := encoder.Encode(a); err != nil {
		return errors.Wrapf(err, "couldn't serialize %T as yaml document", a)
	}
	if err := encoder.Close(); err != nil {
		return errors.Wrapf(err, "couldn't close yaml encloder for %T", a)
	}
	lines := strings.Split(buf.String(), "\n")
	for i, line := range lines[:len(lines)-1] { // last line follows last "\n" and is empty
		if i == 0 {
			BulletedFprintln(indent, w, line)
		} else {
			IndentedFprintln(indent+1, w, line)
		}
	}
	return nil
}

// Markdown files

func PrintMarkdown(indent int, text []byte, widthLimit, lengthLimit int) {
	FprintMarkdown(indent, os.Stdout, text, widthLimit, lengthLimit)
}

func FprintMarkdown(indent int, w io.Writer, text []byte, widthLimit, lengthLimit int) {
	lines := strings.Split(string(text), "\n")
	for i, line := range lines {
		if lengthLimit > 0 && i >= lengthLimit {
			break
		}
		line = strings.TrimRight(line, "\r")
		if line == "" {
			IndentedFprintln(indent, w)
		}
		for len(line) > 0 {
			if len(line) < widthLimit { // we've printed everything!
				IndentedFprintln(indent, w, line)
				break
			}
			IndentedFprintln(indent, w, line[:widthLimit])
			line = line[widthLimit:]
		}
	}
	if lengthLimit > 0 && len(lines) > lengthLimit {
		IndentedFprintln(indent, w, "[remainder of file truncated]")
	}
}
