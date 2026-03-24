# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ghoul is an undead-themed Lisp interpreter written in Go that aims to be simple to understand while being more advanced than a naive interpreter. It features proper tail call optimizations and hygienic macro support.

## Architecture

The codebase is organized into several key packages:

- **`ghoul.go`**: Main entry point that orchestrates parsing and evaluation. Registers built-in functions. Uses `intBinOp` helper for binary integer operations.
- **`cmd/ghoul/main.go`**: CLI interface supporting both REPL mode and file execution via `ProcessFile`.
- **`cmd/wraith/main.go`**: CLI for the wraith tool — possesses Go packages and generates sarcophagi.
- **`parser/`**: Lexer and yacc-based parser for Lisp syntax (uses `parser.y` with goyacc). Sets `SourcePosition` with optional `Filename` on each parsed `Pair`.
- **`evaluator/`**: Core expression evaluation engine with tail call optimization. Handles `define-syntax` as a special form for interleaved macro expansion.
- **`expressions/`**: Expression type definitions, `CodeLocation` interface, `SourcePosition`/`MacroExpansionLocation` types, `TypeName()` helper, and `ScopedIdentifier` for hygiene marks.
- **`macromancy/`**: Macro pattern matching, hygienic expansion via mark-based scoping, and `SyntaxObject` type. The `Macromancer` type is removed — macro expansion is handled inline by the evaluator.
- **`mummy/`**: `Mummy` type for wraith-generated wrappers. Conversion functions (`bytes`, `int-slice`, `float-slice`, `go-nil`, `string-from-bytes`).
- **`wraith/`**: Code generation tool that analyzes Go packages and generates sarcophagus packages with wrapped functions, struct constructors, interface method wrappers, and callback adapters.
- **`logging/`**: Configurable logging with TRACE/DEBUG/WARN levels via shared `log` method.

## Prerequisites

This project requires:
- Go 1.25+ with modules support
- `goyacc` tool for parser generation: `go install golang.org/x/tools/cmd/goyacc@latest`

## Development Commands

### Building
```bash
# Generate parser from yacc grammar (required after changing parser.y)
go generate ./parser

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
go test -v ./evaluator
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

### Evaluator
- Uses continuation-passing style with a continuation stack for proper tail calls
- `chooseEvaluation` uses a type switch for expression dispatch and a nested switch for special forms
- Special forms: `cond`, `else`, `begin`, `lambda`, `define`, `set!`, `define-syntax`
- `define-syntax` is a special form — macro expansion happens interleaved with evaluation, not as a separate pass
- `SyntaxTransformer` (pattern-based) and `GeneralSyntaxTransformer` (lambda-based) are stored in the environment
- Macro calls are intercepted before argument evaluation in `functionCallContinuationFor`
- Lookup fallback: `ScopedIdentifier` with marks falls back to name-only lookup for macro-introduced references to existing bindings
- `specialFormName` strips marks so macro-introduced special form keywords are still recognized

### Hygienic Macros
- Mark-based hygiene: each expansion gets a fresh mark (uint64 counter)
- For `syntax-rules`: template identifiers not in pattern vars and not bound at definition site get the mark
- For general transformers: input is pre-marked, output is marked again — toggle semantics cancels marks on input-originated identifiers
- `SyntaxObject` wraps leaf expressions with marks; `Pair` tree structure is preserved for `List` interface compatibility
- `ResolveExpr` strips `SyntaxObject` wrappers after general transformer expansion
- `syntax-rules` supports: multiple clauses, ellipsis splicing, literals

### Error Messages
- `EvaluationError` includes source location from `Pair.Loc` (set by parser)
- `SourcePosition` carries optional `Filename` — when present, `SourceContext()` reads the file to show surrounding lines with a caret
- `MacroExpansionLocation` points back to the macro call site
- `suggestIdentifiers` provides Levenshtein-based typo suggestions for undefined identifiers
- `TypeName()` gives human-readable type names in error messages

### Wraith Tool
- Generates sarcophagus packages (not code in the target package)
- Uses `mummy.Mummy` for complex type wrapping, `mummy.Entomb`/`Unwrap` for creation/access
- Single type assertion for primitives; `*mummy.Mummy` + `Unwrap()` for complex types
- Methods registered as `typename-methodname` (e.g., `person-getage`)
- Struct constructors as `make-typename` (e.g., `make-person`)
- Callback adapters wrap Ghoul `Function` in Go function closures
- Variadic functions consume remaining args into a slice
- Detects unwrappable types (channels, maps) and fails unless `--skip-unwrappable` is set
- `Environment` type alias exported from evaluator for use in generated `RegisterFunctions`

### Expression Types
- `Pair` has `H`, `T`, and `Loc CodeLocation` fields
- `Foreign` wraps arbitrary Go values (internal use). `mummy.Mummy` wraps with type metadata (wraith-generated code).
- `ScopedIdentifier` carries `Name` + `Marks` for hygiene. Equiv to plain `Identifier` only when marks are empty.
- `Cons` creates Pairs with nil `Loc` by default; parser sets `Loc` during parsing.

## Conventions

- No reflection in generated wraith code
- TDD approach for new features
- Comments should explain *why*, not restate *what* the code does
- Error messages use `TypeName()` for human-readable type names, not `%T`
- Go naming conventions: camelCase variables, no snake_case
- Breaking changes are acceptable — no backward-compat shims needed
- `splitListAt` builds fresh lists to avoid mutating shared expression trees
