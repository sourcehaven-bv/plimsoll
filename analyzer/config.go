package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RootConfig is the top-level plimsoll configuration. It groups one section per
// scope plimsoll checks: a type section (god-object types) and a package section
// (god-object packages). Each scope resolves independently with its own load
// lines, overrides and exclude list.
type RootConfig struct {
	Type    TypeConfig    `yaml:"type" json:"type"`
	Package PackageConfig `yaml:"package" json:"package"`
}

// TypeConfig controls the load lines plimsoll enforces on individual types. A
// zero TypeConfig uses the built-in defaults; a partially-filled one fills any
// unset top-level limit from the defaults so a config file only needs to state
// what it changes.
type TypeConfig struct {
	// MaxMethods is the default cap on the total number of methods declared on
	// a single named type — exported and unexported, pointer and value
	// receivers counted together. This is the backstop for internal sprawl:
	// any receiver carrying dozens of methods (even private helpers) is one
	// struct whose fields every method can reach. Zero means "use the
	// default"; a negative value disables the check.
	MaxMethods int `yaml:"max_methods" json:"max_methods"`

	// MaxExportedMethods is the default cap on the number of *exported* methods
	// on a single named type. This is the sharper god-object signal: exported
	// methods are the coupling surface consumers bind to, so the limit is
	// deliberately stricter than MaxMethods. A type may carry many private
	// helpers without being a god-object, but a wide public API is one by
	// definition. Zero means "use the default"; negative disables.
	MaxExportedMethods int `yaml:"max_exported_methods" json:"max_exported_methods"`

	// MaxExportedFields is the default cap on the number of exported fields in
	// a single struct type. Zero means "use the default"; negative disables.
	MaxExportedFields int `yaml:"max_exported_fields" json:"max_exported_fields"`

	// Overrides set a different limit for specific types, keyed by the type
	// name. A bare name ("App") matches that type in any package; a
	// package-qualified name ("dataentry.App", matched against the package's
	// short name) scopes the override to one package. Use overrides to
	// grandfather existing offenders — keep the number honest so it ratchets
	// down over time rather than hiding growth.
	Overrides map[string]TypeLimit `yaml:"overrides" json:"overrides"`

	// Exclude lists type names (same matching rules as Overrides keys) that are
	// skipped entirely. Prefer an Override with an explicit number over an
	// Exclude: a number still fails CI if the type grows past it, while an
	// exclude is a blind spot.
	Exclude []string `yaml:"exclude" json:"exclude"`
}

// PackageConfig controls the load line plimsoll enforces on whole packages: the
// number of exported types a single package may declare. This is the same
// god-object idea one scope up — a package accreting dozens of exported types is
// the package-scale grab-bag, and "spin up a focused new package" is the
// friction the load line forces.
//
// Overrides and Exclude are keyed by package, with the same qualified-vs-bare
// matching the type scope uses: a key containing a slash (or otherwise equal to
// the full import path) matches that exact package; a bare key with no slash
// matches any package whose short name equals it. Prefer the full import path —
// short package names are not unique (every internal/foo, every v1).
type PackageConfig struct {
	// MaxExportedTypes is the default cap on the number of exported named types
	// declared in a single package. Zero means "use the default"; a negative
	// value disables the check.
	MaxExportedTypes int `yaml:"max_exported_types" json:"max_exported_types"`

	// Overrides set a different limit for specific packages. See PackageConfig.
	Overrides map[string]PackageLimit `yaml:"overrides" json:"overrides"`

	// Exclude lists packages skipped entirely (same matching rules as Overrides
	// keys). Prefer an Override with an explicit number.
	Exclude []string `yaml:"exclude" json:"exclude"`
}

// TypeLimit is a per-type override. A nil pointer field means "inherit the
// default for that dimension"; a set pointer (including a negative value, which
// disables) takes precedence.
type TypeLimit struct {
	MaxMethods         *int `yaml:"max_methods" json:"max_methods"`
	MaxExportedMethods *int `yaml:"max_exported_methods" json:"max_exported_methods"`
	MaxExportedFields  *int `yaml:"max_exported_fields" json:"max_exported_fields"`
}

// PackageLimit is a per-package override. A nil field inherits the default; a
// set pointer (including a negative value, which disables) takes precedence.
type PackageLimit struct {
	MaxExportedTypes *int `yaml:"max_exported_types" json:"max_exported_types"`
}

// Default limits. Deliberately generous — the goal is to catch god-objects, not
// to nag well-factored code. Tighten per-project.
const (
	defaultMaxMethods         = 40
	defaultMaxExportedMethods = 20
	defaultMaxExportedFields  = 20
	defaultMaxExportedTypes   = 40
)

// DefaultConfig returns the built-in configuration used when no config file is
// supplied.
func DefaultConfig() RootConfig {
	return RootConfig{
		Type: TypeConfig{
			MaxMethods:         defaultMaxMethods,
			MaxExportedMethods: defaultMaxExportedMethods,
			MaxExportedFields:  defaultMaxExportedFields,
			Overrides:          map[string]TypeLimit{},
		},
		Package: PackageConfig{
			MaxExportedTypes: defaultMaxExportedTypes,
			Overrides:        map[string]PackageLimit{},
		},
	}
}

// withDefaults returns a copy of c with each section's unset limits filled from
// DefaultConfig. A negative limit is preserved (it means "disabled"); only the
// zero value is treated as "unset".
func (c RootConfig) withDefaults() RootConfig {
	c.Type = c.Type.withDefaults()
	c.Package = c.Package.withDefaults()
	return c
}

// withDefaults fills any unset type-scope limit from the built-in defaults.
func (c TypeConfig) withDefaults() TypeConfig {
	d := DefaultConfig().Type
	if c.MaxMethods == 0 {
		c.MaxMethods = d.MaxMethods
	}
	if c.MaxExportedMethods == 0 {
		c.MaxExportedMethods = d.MaxExportedMethods
	}
	if c.MaxExportedFields == 0 {
		c.MaxExportedFields = d.MaxExportedFields
	}
	if c.Overrides == nil {
		c.Overrides = map[string]TypeLimit{}
	}
	return c
}

// withDefaults fills any unset package-scope limit from the built-in defaults.
func (c PackageConfig) withDefaults() PackageConfig {
	d := DefaultConfig().Package
	if c.MaxExportedTypes == 0 {
		c.MaxExportedTypes = d.MaxExportedTypes
	}
	if c.Overrides == nil {
		c.Overrides = map[string]PackageLimit{}
	}
	return c
}

// LoadConfig reads a YAML config file. An empty path returns DefaultConfig.
// The returned config already has defaults applied.
func LoadConfig(path string) (RootConfig, error) {
	if path == "" {
		return DefaultConfig(), nil
	}
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return RootConfig{}, fmt.Errorf("plimsoll: read config %q: %w", path, err)
	}
	var c RootConfig
	if err := yaml.Unmarshal(b, &c); err != nil {
		return RootConfig{}, fmt.Errorf("plimsoll: parse config %q: %w", path, err)
	}
	return c.withDefaults(), nil
}

// methodLimitFor resolves the effective method cap for a type, applying any
// override. It returns (limit, enabled): enabled is false when the type is
// excluded or the limit is negative (disabled).
func (c TypeConfig) methodLimitFor(pkgName, typeName string) (int, bool) {
	if c.isExcluded(pkgName, typeName) {
		return 0, false
	}
	limit := c.MaxMethods
	if ov, ok := c.overrideFor(pkgName, typeName); ok && ov.MaxMethods != nil {
		limit = *ov.MaxMethods
	}
	if limit < 0 {
		return 0, false
	}
	return limit, true
}

// exportedMethodLimitFor resolves the effective exported-method cap for a type.
func (c TypeConfig) exportedMethodLimitFor(pkgName, typeName string) (int, bool) {
	if c.isExcluded(pkgName, typeName) {
		return 0, false
	}
	limit := c.MaxExportedMethods
	if ov, ok := c.overrideFor(pkgName, typeName); ok && ov.MaxExportedMethods != nil {
		limit = *ov.MaxExportedMethods
	}
	if limit < 0 {
		return 0, false
	}
	return limit, true
}

// fieldLimitFor resolves the effective exported-field cap for a struct type.
func (c TypeConfig) fieldLimitFor(pkgName, typeName string) (int, bool) {
	if c.isExcluded(pkgName, typeName) {
		return 0, false
	}
	limit := c.MaxExportedFields
	if ov, ok := c.overrideFor(pkgName, typeName); ok && ov.MaxExportedFields != nil {
		limit = *ov.MaxExportedFields
	}
	if limit < 0 {
		return 0, false
	}
	return limit, true
}

func (c TypeConfig) overrideFor(pkgName, typeName string) (TypeLimit, bool) {
	if ov, ok := c.Overrides[pkgName+"."+typeName]; ok {
		return ov, true
	}
	ov, ok := c.Overrides[typeName]
	return ov, ok
}

func (c TypeConfig) isExcluded(pkgName, typeName string) bool {
	qualified := pkgName + "." + typeName
	for _, e := range c.Exclude {
		if e == typeName || e == qualified {
			return true
		}
	}
	return false
}

// exportedTypeLimitFor resolves the effective exported-type cap for a package.
// pkgPath is the full import path; pkgName is the short package name. A key with
// a slash (or equal to the full path) matches the exact package; a bare key
// matches by short name. It returns (limit, enabled): enabled is false when the
// package is excluded or the limit is negative.
func (c PackageConfig) exportedTypeLimitFor(pkgPath, pkgName string) (int, bool) {
	if c.isExcluded(pkgPath, pkgName) {
		return 0, false
	}
	limit := c.MaxExportedTypes
	if ov, ok := c.overrideFor(pkgPath, pkgName); ok && ov.MaxExportedTypes != nil {
		limit = *ov.MaxExportedTypes
	}
	if limit < 0 {
		return 0, false
	}
	return limit, true
}

func (c PackageConfig) overrideFor(pkgPath, pkgName string) (PackageLimit, bool) {
	if ov, ok := c.Overrides[pkgPath]; ok {
		return ov, true
	}
	// A bare key (no slash) matches any package with that short name.
	if ov, ok := c.Overrides[pkgName]; ok && !strings.Contains(pkgName, "/") {
		return ov, true
	}
	return PackageLimit{}, false
}

func (c PackageConfig) isExcluded(pkgPath, pkgName string) bool {
	for _, e := range c.Exclude {
		if e == pkgPath {
			return true
		}
		if e == pkgName && !strings.Contains(e, "/") {
			return true
		}
	}
	return false
}
