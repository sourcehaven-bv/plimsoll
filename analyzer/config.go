package analyzer

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config controls the load lines plimsoll enforces. A zero Config uses the
// built-in defaults (see DefaultConfig); a partially-filled Config fills any
// unset top-level limit from the defaults so a config file only needs to state
// what it changes.
type Config struct {
	// MaxMethods is the default cap on the number of methods declared on a
	// single named type (pointer and value receivers counted together).
	// Zero means "use the default"; a negative value disables the check.
	MaxMethods int `yaml:"max_methods" json:"max_methods"`

	// MaxExportedFields is the default cap on the number of exported fields in
	// a single struct type. Zero means "use the default"; negative disables.
	MaxExportedFields int `yaml:"max_exported_fields" json:"max_exported_fields"`

	// Overrides set a different limit for specific types, keyed by the type
	// name. A bare name ("App") matches that type in any package; a
	// package-qualified name ("dataentry.App", matched against the package's
	// short name) scopes the override to one package. Use overrides to
	// grandfather existing offenders — keep the number honest so it ratchets
	// down over time rather than hiding growth.
	Overrides map[string]Limit `yaml:"overrides" json:"overrides"`

	// Exclude lists type names (same matching rules as Overrides keys) that are
	// skipped entirely. Prefer an Override with an explicit number over an
	// Exclude: a number still fails CI if the type grows past it, while an
	// exclude is a blind spot.
	Exclude []string `yaml:"exclude" json:"exclude"`
}

// Limit is a per-type override. A nil pointer field means "inherit the default
// for that dimension"; a set pointer (including a negative value, which
// disables) takes precedence.
type Limit struct {
	MaxMethods        *int `yaml:"max_methods" json:"max_methods"`
	MaxExportedFields *int `yaml:"max_exported_fields" json:"max_exported_fields"`
}

// Default limits. Deliberately generous — the goal is to catch god-objects
// (dozens of methods), not to nag well-factored types. Tighten per-project.
const (
	defaultMaxMethods        = 40
	defaultMaxExportedFields = 20
)

// DefaultConfig returns the built-in configuration used when no config file is
// supplied.
func DefaultConfig() Config {
	return Config{
		MaxMethods:        defaultMaxMethods,
		MaxExportedFields: defaultMaxExportedFields,
		Overrides:         map[string]Limit{},
	}
}

// withDefaults returns a copy of c with any unset top-level limit filled from
// DefaultConfig. A negative limit is preserved (it means "disabled"); only the
// zero value is treated as "unset".
func (c Config) withDefaults() Config {
	d := DefaultConfig()
	if c.MaxMethods == 0 {
		c.MaxMethods = d.MaxMethods
	}
	if c.MaxExportedFields == 0 {
		c.MaxExportedFields = d.MaxExportedFields
	}
	if c.Overrides == nil {
		c.Overrides = map[string]Limit{}
	}
	return c
}

// LoadConfig reads a YAML config file. An empty path returns DefaultConfig.
// The returned Config already has defaults applied.
func LoadConfig(path string) (Config, error) {
	if path == "" {
		return DefaultConfig(), nil
	}
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Config{}, fmt.Errorf("plimsoll: read config %q: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("plimsoll: parse config %q: %w", path, err)
	}
	return c.withDefaults(), nil
}

// methodLimitFor resolves the effective method cap for a type, applying any
// override. It returns (limit, enabled): enabled is false when the type is
// excluded or the limit is negative (disabled).
func (c Config) methodLimitFor(pkgName, typeName string) (int, bool) {
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

// fieldLimitFor resolves the effective exported-field cap for a struct type.
func (c Config) fieldLimitFor(pkgName, typeName string) (int, bool) {
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

func (c Config) overrideFor(pkgName, typeName string) (Limit, bool) {
	if ov, ok := c.Overrides[pkgName+"."+typeName]; ok {
		return ov, true
	}
	ov, ok := c.Overrides[typeName]
	return ov, ok
}

func (c Config) isExcluded(pkgName, typeName string) bool {
	qualified := pkgName + "." + typeName
	for _, e := range c.Exclude {
		if e == typeName || e == qualified {
			return true
		}
	}
	return false
}
