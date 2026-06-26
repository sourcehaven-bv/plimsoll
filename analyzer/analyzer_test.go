package analyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestPlimsoll_Basic(t *testing.T) {
	// Tight caps so the small fixtures trip them: 3 total methods, 3 exported
	// methods, 3 exported fields.
	cfg := TypeConfig{MaxMethods: 3, MaxExportedMethods: 3, MaxExportedFields: 3}
	a := NewType(cfg)
	analysistest.Run(t, analysistest.TestData(), a, "basic")
}

func TestPlimsollPackage(t *testing.T) {
	// Tight cap so the small fixtures trip it: 3 exported types per package.
	a := NewPackage(PackageConfig{MaxExportedTypes: 3})
	analysistest.Run(t, analysistest.TestData(), a, "widepkg", "ignoredpkg", "raisedpkg")
}

func TestConfig_Defaults(t *testing.T) {
	c := RootConfig{}.withDefaults().Type
	if c.MaxMethods != defaultMaxMethods {
		t.Errorf("MaxMethods = %d, want %d", c.MaxMethods, defaultMaxMethods)
	}
	if c.MaxExportedMethods != defaultMaxExportedMethods {
		t.Errorf("MaxExportedMethods = %d, want %d", c.MaxExportedMethods, defaultMaxExportedMethods)
	}
	if c.MaxExportedFields != defaultMaxExportedFields {
		t.Errorf("MaxExportedFields = %d, want %d", c.MaxExportedFields, defaultMaxExportedFields)
	}
	p := RootConfig{}.withDefaults().Package
	if p.MaxExportedTypes != defaultMaxExportedTypes {
		t.Errorf("MaxExportedTypes = %d, want %d", p.MaxExportedTypes, defaultMaxExportedTypes)
	}
}

func TestConfig_ExportedMethodLimit(t *testing.T) {
	n := 25
	neg := -1
	c := TypeConfig{
		MaxExportedMethods: 15,
		Overrides: map[string]TypeLimit{
			"App": {MaxExportedMethods: &n},
			"Off": {MaxExportedMethods: &neg},
		},
		Exclude: []string{"Gen"},
	}.withDefaults()

	if got, ok := c.exportedMethodLimitFor("p", "App"); !ok || got != 25 {
		t.Errorf("App override: got (%d,%v), want (25,true)", got, ok)
	}
	if got, ok := c.exportedMethodLimitFor("p", "Plain"); !ok || got != 15 {
		t.Errorf("default: got (%d,%v), want (15,true)", got, ok)
	}
	if _, ok := c.exportedMethodLimitFor("p", "Off"); ok {
		t.Error("negative override should disable the exported-method check")
	}
	if _, ok := c.exportedMethodLimitFor("p", "Gen"); ok {
		t.Error("Gen should be excluded from the exported-method check too")
	}
}

func TestConfig_OverridesAndExclude(t *testing.T) {
	n := 60
	c := TypeConfig{
		MaxMethods: 40,
		Overrides:  map[string]TypeLimit{"App": {MaxMethods: &n}, "pkg.Local": {MaxMethods: &n}},
		Exclude:    []string{"Gen", "other.Skip"},
	}.withDefaults()

	if got, ok := c.methodLimitFor("anypkg", "App"); !ok || got != 60 {
		t.Errorf("App override: got (%d,%v), want (60,true)", got, ok)
	}
	if got, ok := c.methodLimitFor("pkg", "Local"); !ok || got != 60 {
		t.Errorf("pkg-qualified override: got (%d,%v), want (60,true)", got, ok)
	}
	if got, ok := c.methodLimitFor("anypkg", "Plain"); !ok || got != 40 {
		t.Errorf("default: got (%d,%v), want (40,true)", got, ok)
	}
	if _, ok := c.methodLimitFor("anypkg", "Gen"); ok {
		t.Error("Gen should be excluded")
	}
	if _, ok := c.methodLimitFor("other", "Skip"); ok {
		t.Error("other.Skip should be excluded")
	}
	if _, ok := c.methodLimitFor("notother", "Skip"); !ok {
		t.Error("Skip in a different package should NOT be excluded (pkg-qualified)")
	}
}

func TestConfig_NegativeDisables(t *testing.T) {
	neg := -1
	c := TypeConfig{MaxMethods: 40, Overrides: map[string]TypeLimit{"X": {MaxMethods: &neg}}}.withDefaults()
	if _, ok := c.methodLimitFor("p", "X"); ok {
		t.Error("negative override should disable the check")
	}
}

func TestPackageConfig_ExportedTypeLimit(t *testing.T) {
	n := 120
	neg := -1
	c := PackageConfig{
		MaxExportedTypes: 40,
		Overrides: map[string]PackageLimit{
			"github.com/foo/bar/app": {MaxExportedTypes: &n},   // exact path
			"legacy":                 {MaxExportedTypes: &neg}, // any pkg named legacy
		},
		Exclude: []string{"github.com/foo/bar/gen", "vendored"},
	}.withDefaults()

	// Exact import-path override.
	if got, ok := c.exportedTypeLimitFor("github.com/foo/bar/app", "app"); !ok || got != 120 {
		t.Errorf("path override: got (%d,%v), want (120,true)", got, ok)
	}
	// A different package that happens to share the short name "app" is NOT
	// covered by the path-keyed override — it gets the default.
	if got, ok := c.exportedTypeLimitFor("github.com/other/app", "app"); !ok || got != 40 {
		t.Errorf("unrelated app pkg: got (%d,%v), want (40,true)", got, ok)
	}
	// Bare-name override matches any package with that short name (negative ->
	// disabled).
	if _, ok := c.exportedTypeLimitFor("github.com/x/legacy", "legacy"); ok {
		t.Error("negative bare override should disable the package check")
	}
	// Exact-path exclude.
	if _, ok := c.exportedTypeLimitFor("github.com/foo/bar/gen", "gen"); ok {
		t.Error("gen should be excluded by exact path")
	}
	// Bare-name exclude.
	if _, ok := c.exportedTypeLimitFor("github.com/any/vendored", "vendored"); ok {
		t.Error("vendored should be excluded by short name")
	}
	// Default when nothing matches.
	if got, ok := c.exportedTypeLimitFor("github.com/foo/plain", "plain"); !ok || got != 40 {
		t.Errorf("default: got (%d,%v), want (40,true)", got, ok)
	}
}
