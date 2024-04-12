// Package cli has shared utilities and application logic for the forklift CLI
package cli

import (
	"bytes"
	"fmt"
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
	fmt.Printf("%s%s", makeIndentation(indent), fmt.Sprintf(format, a...))
}

func makeIndentation(indent int) string {
	return strings.Repeat(indentation, indent)
}

func IndentedPrint(indent int, a ...any) {
	fmt.Printf("%s%s", makeIndentation(indent), fmt.Sprint(a...))
}

func IndentedPrintln(indent int, a ...any) {
	fmt.Printf("%s%s\n", makeIndentation(indent), fmt.Sprint(a...))
}

func IndentedPrintYaml(indent int, a any) error {
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
		IndentedPrintln(indent, line)
	}
	return nil
}

// Bulleted

func BulletedPrintf(indent int, format string, a ...any) {
	fmt.Printf("%s%s", makeBullet(indent), fmt.Sprintf(format, a...))
}

func makeBullet(indent int) string {
	return strings.Repeat(indentation, indent) + bullet
}

func BulletedPrint(indent int, a ...any) {
	fmt.Printf("%s%s", makeBullet(indent), fmt.Sprint(a...))
}

func BulletedPrintln(indent int, a ...any) {
	fmt.Printf("%s%s\n", makeBullet(indent), fmt.Sprint(a...))
}

func BulletedPrintYaml(indent int, a any) error {
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
			BulletedPrintln(indent, line)
		} else {
			IndentedPrintln(indent+1, line)
		}
	}
	return nil
}

// Markdown files

func PrintReadme(indent int, readme []byte, widthLimit int) {
	lines := strings.Split(string(readme), "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			IndentedPrintln(indent)
		}
		for len(line) > 0 {
			if len(line) < widthLimit { // we've printed everything!
				IndentedPrintln(indent, line)
				break
			}
			IndentedPrintln(indent, line[:widthLimit])
			line = line[widthLimit:]
		}
	}
}
