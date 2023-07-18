package main

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

func indentedPrintf(indent int, format string, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Printf(format, a...)
}

func indentedPrint(indent int, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Print(a...)
}

func indentedPrintln(indent int, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Println(a...)
}

func indentedPrintYaml(indent int, a any) error {
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
		indentedPrintln(indent, line)
	}
	return nil
}

// Bulleted

func bulletedPrintf(indent int, format string, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Print(bullet)
	fmt.Printf(format, a...)
}

/*
func bulletedPrint(indent int, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Printf(bullet)
	fmt.Print(a...)
}

func bulletedPrintln(indent int, a ...any) {
	for i := 0; i < indent; i++ {
		fmt.Print(indentation)
	}
	fmt.Printf(bullet)
	fmt.Println(a...)
}

func bulletedPrintYaml(indent int, a any) error {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(len(indentation))
	if err := encoder.Encode(a); err != nil {
		return errors.Wrapf(err, "couldn't serialize %T as yaml document", a)
	}
	encoder.Close()
	lines := strings.Split(buf.String(), "\n")
	for i, line := range lines[:len(lines)-1] { // last line follows last "\n" and is empty
		if i == 0 {
			bulletedPrintln(indent, line)
		} else {
			indentedPrintln(indent+1, line)
		}
	}
	return nil
}
*/
