// Command plimsoll runs the plimsoll load-line linters as a standalone tool.
//
// plimsoll enforces "load lines" — Plimsoll lines for Go — on load-bearing
// scopes: a type may not cross its method/exported-field line, and a package may
// not cross its exported-type line. Each scope is a subcommand:
//
//	plimsoll type    [-config plimsoll.yml] ./...   # god-object types
//	plimsoll package [-config plimsoll.yml] ./...   # god-object packages
//	plimsoll         [-config plimsoll.yml] ./...   # run every load line
//
// It exits non-zero when any scope exceeds its load line, so it drops straight
// into a CI step. With no -config it uses the built-in defaults.
//
// A single -config flag feeds every scope, so the bare "run all" form needs
// only one config path — unlike wiring each analyzer's own flag separately.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sourcehaven-bv/plimsoll/analyzer"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	scope, rest := parseScope(os.Args)
	if scope == "help" {
		usage(os.Stderr)
		return
	}

	// Own the -config flag here so a single value feeds every scope. We strip
	// it from the args before handing the remainder to multichecker, which has
	// its own flag set (and would otherwise reject an unknown -config).
	configPath, passthrough, err := extractConfig(rest)
	if err != nil {
		fmt.Fprintln(os.Stderr, "plimsoll:", err)
		os.Exit(2)
	}
	cfg, err := analyzer.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	var analyzers []*analysis.Analyzer
	switch scope {
	case "type":
		analyzers = []*analysis.Analyzer{analyzer.NewType(cfg.Type)}
	case "package":
		analyzers = []*analysis.Analyzer{analyzer.NewPackage(cfg.Package)}
	default: // "" — run every load line
		analyzers = []*analysis.Analyzer{
			analyzer.NewType(cfg.Type),
			analyzer.NewPackage(cfg.Package),
		}
	}

	// multichecker reads os.Args itself. passthrough already preserves argv[0]
	// (with the scope subcommand and -config flag stripped), so hand it over
	// directly.
	os.Args = passthrough
	multichecker.Main(analyzers...)
}

// parseScope splits off a leading scope subcommand ("type" or "package") if
// present. It returns the scope ("" when none, "help" for help requests) and
// the remaining args including argv[0].
func parseScope(args []string) (scope string, rest []string) {
	if len(args) < 2 {
		return "", args
	}
	switch args[1] {
	case "type", "package":
		return args[1], append([]string{args[0]}, args[2:]...)
	case "help", "-h", "--help":
		return "help", args
	default:
		return "", args
	}
}

// extractConfig pulls a -config/-config=… (or --config form) flag out of args,
// returning the path, the remaining args (argv[0] preserved), and any error. A
// single flag value applies to every scope, so the bare "run all" form needs
// only one config path. Other flags pass through to multichecker untouched.
func extractConfig(args []string) (path string, rest []string, err error) {
	rest = append(rest, args[0])
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-config" || a == "--config":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("flag needs an argument: %s", a)
			}
			path = args[i+1]
			i++
		case strings.HasPrefix(a, "-config="):
			path = strings.TrimPrefix(a, "-config=")
		case strings.HasPrefix(a, "--config="):
			path = strings.TrimPrefix(a, "--config=")
		default:
			rest = append(rest, a)
		}
	}
	return path, rest, nil
}

func usage(w *os.File) {
	fmt.Fprint(w, `plimsoll — load-line linters for Go

Usage:
  plimsoll type    [-config plimsoll.yml] ./...   check god-object types
  plimsoll package [-config plimsoll.yml] ./...   check god-object packages
  plimsoll         [-config plimsoll.yml] ./...   run every load line

Exits non-zero when any scope crosses its load line.
`)
}
