package analyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestPlimsoll_Basic(t *testing.T) {
	// Tight caps so the small fixtures trip them: 3 methods, 3 exported fields.
	cfg := Config{MaxMethods: 3, MaxExportedFields: 3}
	a := New(cfg)
	analysistest.Run(t, analysistest.TestData(), a, "basic")
}

func TestConfig_Defaults(t *testing.T) {
	c := Config{}.withDefaults()
	if c.MaxMethods != defaultMaxMethods {
		t.Errorf("MaxMethods = %d, want %d", c.MaxMethods, defaultMaxMethods)
	}
	if c.MaxExportedFields != defaultMaxExportedFields {
		t.Errorf("MaxExportedFields = %d, want %d", c.MaxExportedFields, defaultMaxExportedFields)
	}
}

func TestConfig_OverridesAndExclude(t *testing.T) {
	n := 60
	c := Config{
		MaxMethods: 40,
		Overrides:  map[string]Limit{"App": {MaxMethods: &n}, "pkg.Local": {MaxMethods: &n}},
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
	c := Config{MaxMethods: 40, Overrides: map[string]Limit{"X": {MaxMethods: &neg}}}.withDefaults()
	if _, ok := c.methodLimitFor("p", "X"); ok {
		t.Error("negative override should disable the check")
	}
}
