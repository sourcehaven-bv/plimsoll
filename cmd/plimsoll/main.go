// Command plimsoll runs the plimsoll analyzer as a standalone linter.
//
// Usage:
//
//	plimsoll [-config path/to/plimsoll.yml] ./...
//
// It exits non-zero when any type exceeds its method or exported-field load
// line, so it drops straight into a CI step. With no -config it uses the
// built-in defaults (40 methods, 20 exported fields per type).
package main

import (
	"github.com/sourcehaven-bv/plimsoll/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analyzer.NewWithFlags())
}
