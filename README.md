# plimsoll

A Go linter that flags **god-object types** — those whose method count or
exported-field count has grown past a configurable *load line*.

The name comes from the [Plimsoll line](https://en.wikipedia.org/wiki/Waterline#Plimsoll_line):
the marking on a ship's hull showing the maximum safe load. plimsoll is the
same idea for a Go type — a line it may not cross.

## Why

Go's linter ecosystem caps lots of things, but not this one:

| Linter | Caps |
| --- | --- |
| `interfacebloat` | methods per **interface** |
| `funlen` | lines per **function** |
| `gocyclo` / `gocognit` / `maintidx` | per-**function** complexity |
| `fieldalignment` | struct field **memory layout** (not count) |
| `revive: max-public-structs` | struct count per **file** |

None of them caps the **method or exported-field surface of a concrete type** —
the metric that actually tracks a struct accreting into a god-object. Adding the
228th method to an existing struct is frictionless; spinning up a focused new
type is work, so the path of least resistance always points back at the
god-object. plimsoll adds the missing brake: the 228th method fails CI, forcing
the "should this be its own type?" conversation at the moment of growth instead
of hundreds of methods too late.

## Install

```sh
go install github.com/sourcehaven-bv/plimsoll/cmd/plimsoll@latest
```

## Usage

```sh
plimsoll ./...                      # default caps: 40 methods, 20 exported fields
plimsoll -config plimsoll.yml ./... # project caps + overrides
```

Exits non-zero when any type is over its load line, so it drops straight into a
CI step.

## Configuration

Two layers, by design:

### 1. Global caps + grandfathering — a config file

```yaml
# plimsoll.yml
max_methods: 40
max_exported_fields: 20

# Grandfather existing offenders with an EXPLICIT number, not a blanket skip:
# a number still fails CI if the type grows past it, so it ratchets down over
# time. A bare name matches any package; "pkg.Type" scopes to one package.
overrides:
  App:
    max_methods: 230        # TODO(TKT-xxxx): decompose toward 40
  dataentry.Server:
    max_methods: 60

# Prefer an override with a number over exclude — exclude is a blind spot.
exclude:
  - GeneratedThing
```

### 2. Per-type exceptions — inline directives

Exceptions live next to the code they excuse, so they travel with the type and
vanish when it's split up (unlike a central list, which rots):

```go
//plimsoll:ignore
type LegacyGod struct { /* ... */ }

//plimsoll:max-methods=60
type BusyButBounded struct { /* ... */ }

//plimsoll:max-fields=30
type WideConfig struct { /* ... */ }
```

Precedence (most local wins): **inline directive → config override → default**.
A directive can also *raise* a cap the config lowered, or disable a check with a
negative value.

## What counts

- **Methods**: every method declared on a named type, pointer and value
  receivers together. Methods in `_test.go` files do **not** count (test helpers
  aren't part of a type's shipped surface).
- **Exported fields**: exported fields of a struct, including exported embedded
  types. Unexported fields are ignored.

## golangci-lint

plimsoll is a standard `go/analysis` Analyzer (`analyzer.New` / `analyzer.NewWithFlags`),
so it can be wired in as a golangci-lint
[module plugin](https://golangci-lint.run/plugins/module-plugins/). Until then,
run the standalone binary as its own CI step.

## License

MIT — see [LICENSE](LICENSE).
