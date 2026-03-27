# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ghoul is an undead-themed Lisp interpreter written in Go that aims to be simple to understand while being more advanced than a naive interpreter. It features proper tail call optimizations, hygienic macro support with a separate expansion phase, and a standard prelude with `let`, `let*`, `when`, `unless`, and `syntax-case`.

## Architecture

Processing flows through three phases: **exhume** (parse) -> **reanimate** (expand macros) -> **consume** (evaluate).

The codebase is organized into thematically named packages:

- **`ghoul.go`**: The ghoul itself — the public API that orchestrates the three phases. Creates the exhumer, reanimator, and consumer with a shared mark counter for hygiene.
- **`cmd/ghoul/main.go`**: CLI interface supporting both REPL mode and file execution.
- **`cmd/wraith/main.go`**: CLI for the wraith tool — possesses Go packages and generates sarcophagi.
- **`bones/`**: The skeletal structure — expression type definitions (`Pair`, `Identifier`, `Integer`, `Float`, `String`, `Boolean`, `Quote`, `Foreign`, `ScopedIdentifier`), the `CodeLocation` interface, `SourcePosition`/`MacroExpansionLocation` types, and the `TypeName()` helper.
- **`exhumer/`**: Digs up structure from raw text — lexer and yacc-based parser for Lisp syntax. Sets `SourcePosition` with optional `Filename` on each parsed `Pair`.
- **`reanimator/`**: Brings macromancy macros to life — runs as a separate phase before evaluation. After reanimation, the expression tree contains no `define-syntax` forms or macro calls. Maintains its own macro scope and uses a sub-evaluator for general transformer bodies.
- **`consume/`**: How the ghoul feeds — continuation-passing style (CPS) expression evaluator with proper tail call optimization. After the reanimator phase, the consumer only handles core forms (`cond`, `begin`, `lambda`, `define`, `set!`, `quote`, `require`) and function calls.
- **`macromancy/`**: The dark arts — macro pattern matching, hygienic expansion via mark-based scoping, `SyntaxObject`/`SyntaxTransformer` types, and `BuildSyntaxRulesTransformer`. Nested ellipsis and wildcard (`_`) patterns are supported.
- **`tome/`**: The book of spells — standard library functions (`car`, `cdr`, `cons`, `list`, `+`, `-`, `eq?`, `map`, `filter`, `syntax-match?`, `assoc`, etc.).
- **`mummy/`**: Wraps Go values for use in Ghoul — `Mummy` type for wraith-generated wrappers, conversion functions (`bytes`, `int-slice`, `float-slice`, `go-nil`, `string-from-bytes`).
- **`wraith/`**: Code generation tool that analyzes Go packages and generates sarcophagus packages with wrapped functions, struct constructors, interface method wrappers, and callback adapters.
- **`engraving/`**: Carved records — configurable logging with TRACE/DEBUG/WARN levels.
- **`prelude/`**: The standard prelude (`prelude.ghl`) defining `let`, `let*`, `syntax-case`, `when`, and `unless` macros.

## Prerequisites

This project requires:
- Go 1.25+ with modules support
- `goyacc` tool for parser generation: `go install golang.org/x/tools/cmd/goyacc@latest`

## Development Commands

### Building
```bash
# Generate parser from yacc grammar (required after changing parser.y)
go generate ./exhumer

# Build all packages
go build ./...
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Run tests for specific package
go test -v ./consume
```

### Running
```bash
# Start REPL
./ghoul

# Execute a Ghoul file
./ghoul filename.ghoul

# Possess a Go package (generate sarcophagus)
wraith possess ./path/to/package
wraith -skip-unwrappable possess ./path/to/package
```

## Key Implementation Details

### Three-Phase Processing
1. **Exhume** (`exhumer/`): Parse source text into a `Pair`-based expression tree with source positions.
2. **Reanimate** (`reanimator/`): Walk the expression tree, process `define-syntax` forms to register macros, expand macro calls, and strip all macro-related forms. General transformer bodies are pre-expanded then evaluated through a sub-evaluator.
3. **Consume** (`consume/`): Evaluate the fully-expanded expression tree using CPS with a continuation stack trampoline.

### Consumer (Evaluator)
- Uses continuation-passing style with a continuation stack for proper tail calls
- `chooseEvaluation` uses a type switch for expression dispatch and a nested switch for special forms
- Special forms: `cond`, `else`, `begin`, `lambda`, `define`, `set!`, `quote`, `require`
- The consumer does NOT handle `define-syntax` or macro expansion — that is the reanimator's job
- Lookup fallback: `ScopedIdentifier` with marks falls back to name-only lookup for macro-introduced references to existing bindings
- `specialFormName` strips marks so macro-introduced special form keywords are still recognized

### Hygienic Macros
- Mark-based hygiene: each expansion gets a fresh mark (uint64 counter shared between reanimator and consumer)
- For `syntax-rules`: template identifiers not in pattern vars and not bound at definition site get the mark
- For general transformers: input is pre-marked, output is marked again — toggle semantics cancels marks on input-originated identifiers
- `SyntaxObject` wraps leaf expressions with marks during expansion; `Pair` tree structure is preserved
- `ResolveExpr` strips `SyntaxObject` wrappers after general transformer expansion
- `syntax-rules` supports: multiple clauses, nested ellipsis (`(var val) ...`), wildcard (`_`), literals
- `syntax-case` is defined in the prelude as a general transformer macro

### Reanimator (Macro Expander)
- Walks the expression tree top-down, processing `define-syntax` and expanding macro calls
- Maintains scoped macro bindings (parent-chain for inner `define-syntax` in lambda/begin)
- For `syntax-rules`: builds transformer directly via `macromancy.BuildSyntaxRulesTransformer`
- For general transformers: pre-expands the transformer expression, evaluates it to get a `Function`, then invokes it with mark-based hygiene during macro call expansion
- Returns original expressions unchanged when no macros are present, preserving source positions
- `containsMacroCall` check avoids unnecessary tree rebuilding

### Error Messages
- `EvaluationError` includes source location from `Pair.Loc` (set by exhumer)
- `SourcePosition` carries optional `Filename` — when present, `SourceContext()` reads the file to show surrounding lines with a caret
- `MacroExpansionLocation` points back to the macro call site
- `suggestIdentifiers` provides Levenshtein-based typo suggestions for undefined identifiers
- `TypeName()` gives human-readable type names in error messages — internal Go types never leak into user-facing errors

### Wraith Tool
- Generates sarcophagus packages (not code in the target package)
- Uses `mummy.Mummy` for complex type wrapping, `mummy.Entomb`/`Unwrap` for creation/access
- Single type assertion for primitives; `*mummy.Mummy` + `Unwrap()` for complex types
- Methods registered as `typename-methodname` (e.g., `person-getage`)
- Struct constructors as `make-typename` (e.g., `make-person`)
- Callback adapters wrap Ghoul `Function` in Go function closures
- Variadic functions consume remaining args into a slice
- Detects unwrappable types (channels, maps) and fails unless `--skip-unwrappable` is set
- `Environment` type alias exported from `consume` for use in generated `RegisterFunctions`

### Expression Types (Bones)
- `Pair` has `H`, `T`, and `Loc CodeLocation` fields
- `Foreign` wraps arbitrary Go values (internal use). `mummy.Mummy` wraps with type metadata (wraith-generated code).
- `ScopedIdentifier` carries `Name` + `Marks` for hygiene. Equiv to plain `Identifier` only when marks are empty.
- `Cons` creates Pairs with nil `Loc` by default; exhumer sets `Loc` during parsing.

## Conventions

- All packages use undead/occult themed names
- No reflection in generated wraith code
- TDD approach for new features
- Comments should explain *why*, not restate *what* the code does
- Error messages use `TypeName()` for human-readable type names, not `%T`
- Go naming conventions: camelCase variables, no snake_case
- Breaking changes are acceptable — no backward-compat shims needed
- `splitListAt` builds fresh lists to avoid mutating shared expression trees
- `newEnvWithEmptyScope` copies the slice to avoid aliasing — prevents a subtle scope corruption bug in recursive functions
