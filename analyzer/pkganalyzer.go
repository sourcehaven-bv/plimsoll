package analyzer

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// NewPackage returns the plimsoll package-scope Analyzer bound to cfg. It flags
// a package whose exported-type count has grown past its load line — the
// god-object idea one scope up from a single type. cfg should already have
// defaults applied (LoadConfig does this).
func NewPackage(cfg PackageConfig) *analysis.Analyzer {
	r := &pkgRunner{cfg: cfg.withDefaults()}
	return &analysis.Analyzer{
		Name: "plimsollpackage",
		Doc:  "checks that a package's exported-type count stays under a configurable load line",
		Run:  r.run,
	}
}

// NewPackageWithFlags returns the package-scope Analyzer that reads its config
// from a -config file path at run time, for the standalone command.
func NewPackageWithFlags() *analysis.Analyzer {
	r := &pkgRunner{}
	a := &analysis.Analyzer{
		Name: "plimsollpackage",
		Doc:  "checks that a package's exported-type count stays under a configurable load line",
		Run:  r.runWithConfigPath,
	}
	a.Flags.StringVar(&r.configPath, "config", "", "path to a plimsoll YAML config file")
	return a
}

type pkgRunner struct {
	cfg        PackageConfig
	configPath string
}

func (r *pkgRunner) runWithConfigPath(pass *analysis.Pass) (any, error) {
	cfg, err := LoadConfig(r.configPath)
	if err != nil {
		return nil, err
	}
	r.cfg = cfg.Package
	return r.run(pass)
}

func (r *pkgRunner) run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	pkgName := pass.Pkg.Name()

	exported := 0     // exported named types declared in this package
	var dir directive // package directive, from the package doc comment
	reportPos := token.Pos(token.NoPos)

	for _, file := range pass.Files {
		// Test files don't count toward a package's shipped surface — test
		// helper types aren't part of the public API, and counting them makes
		// the normal vs. test package variants disagree (a spurious double
		// report). Skipping _test.go keeps the two variants identical, which
		// the driver dedupes.
		if isTestFile(pass, file) {
			continue
		}
		// The package directive lives on the package doc comment. Take the
		// first one we find (the file carrying the doc comment for the package).
		if file.Doc != nil {
			if fd := parseDirective(file.Doc); fd.hasOverride {
				dir = fd
			}
		}
		// Report against the package clause of the first non-test file.
		if !reportPos.IsValid() {
			reportPos = file.Package
		}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if ts.Name.IsExported() {
					exported++
				}
			}
		}
	}

	limit, enabled := r.limit(pkgPath, pkgName, dir)
	if !enabled {
		return nil, nil
	}
	if exported > limit {
		pass.Reportf(reportPos,
			"package %s has %d exported types, over the load line of %d (split it into focused packages, or annotate with //plimsoll:max-exported-types=N)",
			pkgName, exported, limit)
	}
	return nil, nil
}

// limit resolves the effective exported-type cap for the package, with
// precedence: package directive > config override > config default. It returns
// (limit, enabled); enabled is false when the package is ignored or the cap is
// negative (disabled).
func (r *pkgRunner) limit(pkgPath, pkgName string, dir directive) (int, bool) {
	if dir.ignore {
		return 0, false
	}
	if dir.maxExpTypes != nil {
		if *dir.maxExpTypes < 0 {
			return 0, false
		}
		return *dir.maxExpTypes, true
	}
	return r.cfg.exportedTypeLimitFor(pkgPath, pkgName)
}
