package parser

import (
	"bufio"
	"io"
	"strings"
)

// Result represents a test result.
type Result int

// Test result constants
const (
	PASS Result = iota
	FAIL
	SKIP
)

// Report is a collection of package tests.
type Report struct {
	Packages []Package
}

// Package contains the test results of a single package.
type Package struct {
	Name        string
	Time        int
	Tests       []*Test
	CoveragePct string
}

// Test contains the results of a single test.
type Test struct {
	Name   string
	Time   int
	Result Result
	Output []string
}

type parser interface {
	IngestLine(string) error
	Report() (*Report, error)
}

// Parse parses go test output from reader r and returns a report with the
// results. An optional pkgName can be given, which is used in case a package
// result line is missing.
func Parse(r io.Reader, pkgName string) (*Report, error) {
	reader := bufio.NewReader(r)
	var p parser
	parserDetect := true

	backlog := make([]string, 0)

	clearBacklock := func() error {
		for _, l := range backlog {
			err := p.IngestLine(l)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// parse lines
	for {
		l, _, err := reader.ReadLine()
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		line := string(l)

		if parserDetect {
			if strings.HasPrefix(line, "{") {
				p = newJsonlParser(pkgName)
			} else if strings.HasPrefix(line, "=") {
				p = newTextParser(pkgName)
			} else {
				backlog = append(backlog, line)
				continue
			}
			parserDetect = false
			err = clearBacklock()
			if err != nil {
				return nil, err
			}
		}

		err = p.IngestLine(line)
		if err != nil {
			return nil, err
		}
	}

	if p == nil { // we only have a backlog, let's assume text parser
		p = newTextParser(pkgName)
		err := clearBacklock()
		if err != nil {
			return nil, err
		}
	}

	return p.Report()
}

// Failures counts the number of failed tests in this report
func (r *Report) Failures() int {
	count := 0

	for _, p := range r.Packages {
		for _, t := range p.Tests {
			if t.Result == FAIL {
				count++
			}
		}
	}

	return count
}
