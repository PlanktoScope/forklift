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
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Printf(format, a...)
}

func IndentedPrint(indent int, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Print(a...)
}

func IndentedPrintln(indent int, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Println(a...)
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
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Print(bullet)
	fmt.Printf(format, a...)
}

func BulletedPrint(indent int, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Print(bullet)
	fmt.Print(a...)
}

func BulletedPrintln(indent int, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Print(bullet)
	fmt.Println(a...)
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
