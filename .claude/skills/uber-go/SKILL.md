---
name: uber-go
description: Uber Go Style Guide rules. Use when writing, reviewing or refactoring Go code.
---

# Uber Go Style Guide

The rule index below condenses the Uber Go Style Guide. The full guide, with a Bad/Good example for every rule, is [`style.md`](style.md) in this folder (upstream `uber-go/guide` commit `1d60a91`, Apache-2.0, see [`LICENSE`](LICENSE)).

## Precedence

1. The project's own conventions come first: its `ARCHITECTURE.md`, `CLAUDE.md`, linter config and the patterns in the surrounding code.
2. Then the **Modern Go adjustments** section below, which updates rules the guide wrote for older Go versions. Check the `go` line in `go.mod` before applying it.
3. Then this guide.

When a project convention overrides a rule, follow the project and mention the conflict once in your reply.

## Steps

**Writing or refactoring Go:** apply every rule in the index to the code you touch. Leave untouched code as it is, unless the user asked for a style pass. When the module has a `.golangci.yml`, the work is done when `golangci-lint run ./...` and `golangci-lint fmt --diff ./...` both pass. Fix each issue in the code; a `//nolint:<linter>` needs a comment giving the reason.

**Reviewing Go:** go through every rule in the index against the diff. Report each violation with `file:line`, the rule's heading name exactly as written in the index, and the fix. The review is done when every rule has been checked against every changed file.

**When a rule's detail is unclear:** look it up instead of guessing. Run `grep -n '^##* <Heading>' .claude/skills/uber-go/style.md`, then Read from that line up to the next heading. Every heading in the index matches a heading in `style.md`.

## Rule index

### Guidelines

- **Pointers to Interfaces**: pass interfaces by value. A pointer to an interface is almost never needed.
- **Verify Interface Compliance**: add `var _ Iface = (*Impl)(nil)` for every exported type that must satisfy an interface.
- **Receivers and Interfaces**: value-receiver methods work on values and pointers. Pointer-receiver methods work only on pointers or addressable values.
- **Zero-value Mutexes are Valid**: declare `var mu sync.Mutex` (no `new`). Keep the mutex as an unexported field, never embedded in an exported struct.
- **Copy Slices and Maps at Boundaries**: copy slices and maps you receive and store, and copy internal slices and maps before returning them.
- **Defer to Clean Up**: release locks, files and other resources with `defer` straight after acquiring them.
- **Channel Size is One or None**: channels are unbuffered or have size 1. Any larger size needs a justification of how it is bounded.
- **Start Enums at One**: start `iota` enums at `iota + 1`, unless the zero value is a meaningful default.
- **Use `"time"` to handle time**: `time.Time` for instants, `time.Duration` for periods. At external boundaries, put the unit in the field name (`IntervalMillis`) when a Duration cannot be used.
- **Error Types**: plain message, no matching needed → `errors.New`/`fmt.Errorf`. Caller must match → exported sentinel `var ErrX` or a custom type checked with `errors.Is`/`errors.As`.
- **Error Wrapping**: add context with `fmt.Errorf("get user %q: %w", id, err)`. Use `%w` when callers may match the cause, `%v` to hide it. Keep the context short with no "failed to" prefix.
- **Error Naming**: exported error vars start with `Err`, unexported with `err`. Custom error types end with `Error`.
- **Handle Errors Once**: each error is either logged or returned, never both. Wrap it and return, or handle it and degrade.
- **Handle Type Assertion Failures**: use the two-value form `t, ok := i.(T)`.
- **Don't Panic**: return errors. `panic` is only for truly impossible states or program initialisation. Tests use `t.Fatal`.
- **Use go.uber.org/atomic**: use typed atomics so every access is atomic (see Modern Go adjustments).
- **Avoid Mutable Globals**: inject dependencies (clocks, clients, functions) through struct fields instead of package-level vars.
- **Avoid Embedding Types in Public Structs**: write explicit delegate methods so the embedded type's API does not leak.
- **Avoid Using Built-In Names**: keep `error`, `string`, `len`, `copy`, `new` etc. free for their built-in meaning.
- **Avoid `init()`**: initialise with plain var declarations or explicit setup functions. If `init()` is unavoidable, keep it deterministic with no I/O, env or global-state dependence.
- **Exit in Main**: only `main` calls `os.Exit`/`log.Fatal`. Every other function returns an error.
- **Exit Once**: put the logic in `run() error`. `main` calls it and exits once on error.
- **Use field tags in marshaled structs**: every field marshalled to JSON/YAML/etc. carries an explicit tag.
- **Don't fire-and-forget goroutines**: every goroutine has a way to be stopped and a way to wait for it to finish (`sync.WaitGroup`, done channel).
- **Wait for goroutines to exit**: the owner of a goroutine waits for it before returning or shutting down.
- **No goroutines in `init()`**: start background work from an object with `Close`/`Shutdown` that stops and waits for it.

### Performance (hot paths only)

- **Prefer strconv over fmt**: `strconv.Itoa(n)` rather than `fmt.Sprint(n)`.
- **Avoid repeated string-to-byte conversions**: convert once, outside the loop.
- **Prefer Specifying Container Capacity**: `make(map[K]V, n)` and `make([]T, 0, n)` when the size is known.

### Style

- **Avoid overly long lines**: soft limit 99 characters.
- **Be Consistent**: match the style already in the package. Apply a new style package-wide or not at all.
- **Group Similar Declarations**: group related `const`, `var`, `type` and `import` blocks, and only related ones.
- **Import Group Ordering**: two groups, the standard library then everything else (the `goimports` default).
- **Package Names**: short, lowercase, singular, no underscores. Avoid `common`, `util`, `shared`, `lib`.
- **Function Names**: MixedCaps. Test names may use underscores to group cases (`TestMyFunction_WhatIsBeingTested`).
- **Import Aliasing**: alias only when the package name differs from the last element of the import path, or when names collide.
- **Function Grouping and Ordering**: order a file as type, constructor `newX`/`NewX`, exported methods, unexported methods, then plain helper functions. Sort methods roughly in call order.
- **Reduce Nesting**: handle errors and special cases first and return early.
- **Unnecessary Else**: when both branches only assign a variable, assign the default first and overwrite it in the `if`.
- **Top-level Variable Declarations**: `var x = f()` with no type unless the type differs from the expression's type.
- **Prefix Unexported Globals with _**: unexported top-level `var`/`const` start with `_` (unexported error values use the `err` prefix instead).
- **Embedding in Structs**: put embedded fields at the top of the struct, followed by a blank line. Embed only when it adds real behaviour, never just for convenience.
- **Local Variable Declarations**: use `:=` when setting a value. Use `var` when the zero value is intended (`var s []int`).
- **nil is a valid slice**: return `nil` rather than `[]T{}`, check emptiness with `len(s) == 0`, and use the zero value `var s []T`.
- **Reduce Scope of Variables**: declare variables in the narrowest scope (`if err := f(); err != nil`) unless that adds nesting.
- **Avoid Naked Parameters**: label unclear literal arguments with `/* name */` comments, or replace bools with named types.
- **Use Raw String Literals to Avoid Escaping**: use backtick strings instead of escaped quotes.
- **Use Field Names to Initialize Structs**: `User{Name: "a"}`, always (positional only in small test tables).
- **Omit Zero Value Fields in Structs**: leave out fields whose value is the zero value, unless it carries meaning.
- **Use `var` for Zero Value Structs**: `var u User`, not `u := User{}`.
- **Initializing Struct References**: `&T{Name: "a"}`, not `new(T)` followed by field assignments.
- **Initializing Maps**: `make(map[K]V)` for maps filled in code, a map literal for a fixed set of entries.
- **Format Strings outside Printf**: declare format strings used outside the call as `const`, so `go vet` can check them.
- **Naming Printf-style Functions**: end the name with `f` (`Wrapf`) so `go vet` checks the format string.

### Patterns

- **Test Tables**: use a `tests` slice of structs, `tt` as the loop variable, and `give`/`want` field prefixes.
- **Avoid Unnecessary Complexity in Table Tests**: every field is used by every case, with no conditional assertions inside the loop. Split into separate tests when cases need branching.
- **Parallel Tests**: call `t.Parallel()` inside `t.Run` (see Modern Go adjustments for loop variables).
- **Functional Options**: for constructors with optional arguments, use an `Option` interface with an unexported `apply(*options)` method plus `WithX` constructors.

### Linting

- **Linting**: baseline `errcheck`, `goimports`, `revive`, `govet`, `staticcheck`, run consistently across the codebase.
- **Lint Runners**: use `golangci-lint`.

## Modern Go adjustments

The guide targets the two latest Go releases, but some sections predate newer features:

- **Use go.uber.org/atomic** → from Go 1.19 on, use the standard library's typed `atomic.Int64`, `atomic.Bool`, `atomic.Pointer[T]`. They give the same type safety with no extra dependency.
- **Parallel Tests** → from Go 1.22 on, each loop iteration gets its own variable, so the `tt := tt` copy is unneeded and should be left out.
- **Lint Runners** → upstream's sample `.golangci.yml` is golangci-lint v1 format and enables the deprecated `golint`. Write configs in v2 format (`version: "2"`, with `goimports` under `formatters:`) and use `revive` in place of `golint`.
