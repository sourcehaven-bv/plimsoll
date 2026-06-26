# plimsoll

Load-line linters for Go. plimsoll flags **god-objects** — code that has grown
past a configurable *load line* — at two scopes:

- **types** whose method, exported-method, or exported-field count is too high;
- **packages** whose exported-type count is too high.

The name comes from the [Plimsoll line](https://en.wikipedia.org/wiki/Waterline#Plimsoll_line):
the marking on a ship's hull showing the maximum safe load. plimsoll is the same
idea for a load-bearing Go scope — a line it may not cross.

## Why

Go's linter ecosystem caps lots of things, but not these two:

| Linter | Caps |
| --- | --- |
| `interfacebloat` | methods per **interface** |
| `funlen` | lines per **function** |
| `gocyclo` / `gocognit` / `maintidx` | per-**function** complexity |
| `fieldalignment` | struct field **memory layout** (not count) |
| `revive: max-public-structs` | struct count per **file** |

None of them caps the **method/exported-field surface of a concrete type**, nor
the **exported-type surface of a package** — the metrics that actually track a
struct accreting into a god-object, or a package accreting into a grab-bag.
Adding the 228th method to an existing struct, or the 80th exported type to an
existing package, is frictionless; spinning up a focused new type or package is
work, so the path of least resistance always points back at the god-object.
plimsoll adds the missing brake: the 228th method (or the 80th exported type)
fails CI, forcing the "should this be its own type/package?" conversation at the
moment of growth instead of hundreds of declarations too late.

## Install

```sh
go install github.com/sourcehaven-bv/plimsoll/cmd/plimsoll@latest
```

## Usage

Each scope is a subcommand; the bare form runs every load line — the CI default:

```sh
plimsoll type    ./...   # god-object types
plimsoll package ./...   # god-object packages
plimsoll         ./...   # run every load line

plimsoll -config plimsoll.yml ./...   # project caps + overrides (one config feeds every scope)
```

Default caps: per type, 40 methods / 20 exported methods / 20 exported fields;
per package, 40 exported types. Exits non-zero when any scope is over its load
line, so it drops straight into a CI step.

## Configuration

Two layers, by design.

### 1. Global caps + grandfathering — a config file

One config, one section per scope. A single `-config` flag feeds every scope.

```yaml
# plimsoll.yml
type:
  max_methods: 40           # total methods (exported + unexported)
  max_exported_methods: 20  # exported methods — the public-API surface
  max_exported_fields: 20

  # Grandfather existing offenders with an EXPLICIT number, not a blanket skip:
  # a number still fails CI if the type grows past it, so it ratchets down over
  # time. A bare name matches any package; "pkg.Type" scopes to one package.
  overrides:
    App:
      max_methods: 230          # TODO(TKT-xxxx): decompose toward 40
      max_exported_methods: 30  # raise just the public line for this one type
    dataentry.Server:
      max_methods: 60

  # Prefer an override with a number over exclude — exclude is a blind spot.
  exclude:
    - GeneratedThing

package:
  max_exported_types: 40

  # Keyed by package. Prefer the full import path — it is unique; a bare key
  # with no slash matches any package with that short name.
  overrides:
    github.com/you/app/internal/store:
      max_exported_types: 90    # TODO: split the store package
  exclude:
    - github.com/you/app/gen    # generated package
```

### 2. Per-scope exceptions — inline directives

Exceptions live next to the code they excuse, so they travel with it and vanish
when it's split up (unlike a central list, which rots). All use the `//plimsoll:`
prefix; the scope is implied by where the directive sits.

On a type:

```go
//plimsoll:ignore
type LegacyGod struct { /* ... */ }

//plimsoll:max-methods=60
type BusyButBounded struct { /* ... */ }

//plimsoll:max-exported-methods=25
type WidePublicAPI struct { /* ... */ }

//plimsoll:max-fields=30
type WideConfig struct { /* ... */ }
```

On the package doc comment:

```go
//plimsoll:max-exported-types=80
package widebutbounded

//plimsoll:ignore
package legacy
```

Precedence (most local wins): **inline directive → config override → default**.
A directive can also *raise* a cap the config lowered, or disable a check with a
negative value.

## What counts

### Type scope (`plimsoll type`)

- **Methods** (`max_methods`): every method declared on a named type, exported
  and unexported, pointer and value receivers together. This is the backstop for
  internal sprawl — a receiver carrying dozens of private helpers is still one
  struct whose fields they can all reach. Methods in `_test.go` files do **not**
  count (test helpers aren't part of a type's shipped surface).
- **Exported methods** (`max_exported_methods`): just the exported methods — the
  coupling surface consumers bind to, and the sharper god-object signal. Its
  default (20) is stricter than `max_methods` (40): a type may carry many private
  helpers without being a god-object, but a wide *public* API is one by
  definition. The two lines resolve independently, so a type can sit under the
  exported line while a separate directive grandfathers its total count.
- **Exported fields** (`max_exported_fields`): exported fields of a struct,
  including exported embedded types. Unexported fields are ignored.

### Package scope (`plimsoll package`)

- **Exported types** (`max_exported_types`): the number of exported named types
  declared in a package — structs, interfaces, aliases, and other named types
  alike, since each widens the package's public surface. Types in `_test.go`
  files do **not** count. Overrides and exclude are keyed by package: prefer the
  full import path (unique); a bare key with no slash matches any package with
  that short name.

## golangci-lint

plimsoll ships two standard `go/analysis` Analyzers — `analyzer.NewType` /
`analyzer.NewTypeWithFlags` and `analyzer.NewPackage` / `analyzer.NewPackageWithFlags`
— so each can be wired in as a golangci-lint
[module plugin](https://golangci-lint.run/plugins/module-plugins/). Until then,
run the standalone binary as its own CI step.

## License

MIT — see [LICENSE](LICENSE).
