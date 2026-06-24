// Package analyzer implements plimsoll, a go/analysis Analyzer that flags
// "god object" types: those whose method count, exported-method count, or
// exported-field count exceeds a configurable load line.
//
// The ecosystem has linters for interface size (interfacebloat), function
// length (funlen) and complexity (gocyclo/gocognit), but none that caps the
// method or field surface of a concrete type — the metric that actually tracks
// a struct accreting into a god-object. plimsoll fills that gap, with per-type
// overrides so existing offenders can be grandfathered and ratcheted down.
//
// plimsoll enforces three independent load lines:
//
//   - max-methods: the total method count (exported + unexported). A backstop
//     for internal sprawl — a receiver carrying dozens of methods is one struct
//     whose fields every method can reach, regardless of visibility.
//   - max-exported-methods: the exported method count. The sharper god-object
//     signal, since exported methods are the coupling surface consumers bind
//     to; its default is stricter than max-methods.
//   - max-fields: the exported struct-field count.
//
// Each line resolves independently with the same precedence (inline directive >
// config override > default), so a type can raise one without touching another.
package analyzer

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// New returns a plimsoll Analyzer bound to cfg. cfg should already have
// defaults applied (LoadConfig does this). Use New when embedding plimsoll as a
// golangci-lint module plugin, where config comes from the host. For a
// standalone binary, NewWithFlags wires a -config flag instead.
func New(cfg Config) *analysis.Analyzer {
	r := &runner{cfg: cfg.withDefaults()}
	return &analysis.Analyzer{
		Name: "plimsoll",
		Doc:  "checks that a type's method count and exported-field count stay under a configurable load line",
		Run:  r.run,
	}
}

// NewWithFlags returns a plimsoll Analyzer that reads its config from a
// -plimsoll.config file path at run time. This is the form the standalone
// command uses (via singlechecker), so the config file can be passed on the
// command line.
func NewWithFlags() *analysis.Analyzer {
	r := &runner{}
	a := &analysis.Analyzer{
		Name: "plimsoll",
		Doc:  "checks that a type's method count and exported-field count stay under a configurable load line",
		Run:  r.runWithConfigPath,
	}
	a.Flags.StringVar(&r.configPath, "config", "", "path to a plimsoll YAML config file")
	return a
}

type runner struct {
	cfg        Config
	configPath string
}

func (r *runner) runWithConfigPath(pass *analysis.Pass) (any, error) {
	cfg, err := LoadConfig(r.configPath)
	if err != nil {
		return nil, err
	}
	r.cfg = cfg
	return r.run(pass)
}

func (r *runner) run(pass *analysis.Pass) (any, error) {
	pkgName := pass.Pkg.Name()

	methodCounts := map[string]int{}     // type name -> total method count
	exportedMethods := map[string]int{}  // type name -> exported method count
	typePos := map[string]token.Pos{}    // type name -> decl position (for the report)
	structFields := map[string]int{}     // struct type name -> exported field count
	directives := map[string]directive{} // type name -> inline directive

	for _, file := range pass.Files {
		// Test files don't count toward a production type's surface — a type
		// with test-only helper methods isn't a god-object, and counting them
		// also makes the normal vs. test package variants disagree (causing a
		// spurious double report). Skipping _test.go keeps the metric about the
		// shipped API and makes the two variants produce identical results,
		// which the driver dedupes.
		if isTestFile(pass, file) {
			continue
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if name, ok := receiverTypeName(d); ok {
					methodCounts[name]++
					if d.Name.IsExported() {
						exportedMethods[name]++
					}
				}
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					typePos[ts.Name.Name] = ts.Name.Pos()
					if st, ok := ts.Type.(*ast.StructType); ok {
						structFields[ts.Name.Name] = countExportedFields(st)
					}
					// A directive may sit on the TypeSpec itself or, for a
					// single-spec `type X …` block, on the GenDecl.
					if dir := parseDirective(ts.Doc); dir.hasOverride {
						directives[ts.Name.Name] = dir
					} else if dir := parseDirective(d.Doc); dir.hasOverride {
						directives[ts.Name.Name] = dir
					}
				}
			}
		}
	}

	// Methods may be declared in files where the type is not (legal in Go), so
	// ensure every type that has methods has a reportable position even if its
	// TypeSpec was processed in a different file of the same package.
	for name := range methodCounts {
		if _, ok := typePos[name]; !ok {
			typePos[name] = token.NoPos
		}
	}

	for _, name := range sortedKeys(methodCounts) {
		limit, enabled := r.limit(pkgName, name, directives[name], dimMethods)
		if !enabled {
			continue
		}
		if got := methodCounts[name]; got > limit {
			pass.Reportf(typePos[name],
				"type %s has %d methods, over the load line of %d (split it into focused types, or annotate with //plimsoll:max-methods=N)",
				name, got, limit)
		}
	}

	for _, name := range sortedKeys(exportedMethods) {
		limit, enabled := r.limit(pkgName, name, directives[name], dimExportedMethods)
		if !enabled {
			continue
		}
		if got := exportedMethods[name]; got > limit {
			pass.Reportf(typePos[name],
				"type %s has %d exported methods, over the load line of %d (narrow its public API into focused types, or annotate with //plimsoll:max-exported-methods=N)",
				name, got, limit)
		}
	}

	for _, name := range sortedKeys(structFields) {
		limit, enabled := r.limit(pkgName, name, directives[name], dimFields)
		if !enabled {
			continue
		}
		if got := structFields[name]; got > limit {
			pass.Reportf(typePos[name],
				"struct %s has %d exported fields, over the load line of %d (group related fields into a sub-struct, or annotate with //plimsoll:max-fields=N)",
				name, got, limit)
		}
	}

	return nil, nil
}

// isTestFile reports whether file is a _test.go file, by resolving its name
// through the pass's FileSet.
func isTestFile(pass *analysis.Pass, file *ast.File) bool {
	name := pass.Fset.File(file.Pos()).Name()
	return strings.HasSuffix(name, "_test.go")
}

// dimension selects which load line to resolve.
type dimension int

const (
	dimMethods dimension = iota
	dimExportedMethods
	dimFields
)

// limit resolves the effective cap for (type, dimension), with precedence:
// inline directive > config override > config default. It returns
// (limit, enabled); enabled is false when the type is ignored (by directive or
// config exclude) or the cap is negative (disabled).
func (r *runner) limit(pkgName, typeName string, dir directive, dim dimension) (int, bool) {
	// An inline //plimsoll:ignore is the most local, most visible opt-out.
	if dir.ignore {
		return 0, false
	}

	var (
		base    int
		enabled bool
		dirVal  *int
	)
	switch dim {
	case dimMethods:
		base, enabled = r.cfg.methodLimitFor(pkgName, typeName)
		dirVal = dir.maxMethods
	case dimExportedMethods:
		base, enabled = r.cfg.exportedMethodLimitFor(pkgName, typeName)
		dirVal = dir.maxExpMethods
	case dimFields:
		base, enabled = r.cfg.fieldLimitFor(pkgName, typeName)
		dirVal = dir.maxFields
	}

	// A per-type directive overrides config — including re-enabling a check the
	// config disabled, or setting a negative (disabling) value.
	if dirVal != nil {
		if *dirVal < 0 {
			return 0, false
		}
		return *dirVal, true
	}
	return base, enabled
}

// receiverTypeName returns the named receiver type of a method declaration,
// unwrapping a pointer receiver. Returns ("", false) for free functions and for
// receivers that are not a plain named type (e.g. generic instantiations are
// reduced to their base name).
func receiverTypeName(fd *ast.FuncDecl) (string, bool) {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return "", false
	}
	expr := fd.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	// Generic receiver: Foo[T] -> base name Foo.
	if idx, ok := expr.(*ast.IndexExpr); ok {
		expr = idx.X
	}
	if idx, ok := expr.(*ast.IndexListExpr); ok {
		expr = idx.X
	}
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name, true
	}
	return "", false
}

// countExportedFields counts the exported fields of a struct literal. An
// embedded field counts as one field, named by its type; embedded exported
// types are counted (they widen the exported surface just like a named field).
func countExportedFields(st *ast.StructType) int {
	if st.Fields == nil {
		return 0
	}
	n := 0
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			// Embedded field — exported iff the embedded type name is exported.
			if name, ok := embeddedName(f.Type); ok && ast.IsExported(name) {
				n++
			}
			continue
		}
		for _, name := range f.Names {
			if name.IsExported() {
				n++
			}
		}
	}
	return n
}

func embeddedName(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name, true
	case *ast.StarExpr:
		return embeddedName(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name, true
	default:
		return "", false
	}
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
