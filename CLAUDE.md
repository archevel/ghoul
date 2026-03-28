# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ghoul is an undead-themed Lisp interpreter written in Go that aims to be simple to understand while being more advanced than a naive interpreter. It features proper tail call optimizations, hygienic macro support with a separate expansion phase, and a standard prelude with `let`, `let*`, `when`, `unless`, and `syntax-case`.

## Architecture

Processing flows through three phases: **exhume** (parse to `*bones.Node`) -> **reanimate** (expand macros + translate to semantic nodes) -> **consume** (compile to bytecode + execute on stack VM).

Everything is `*bones.Node`. There is no `Expr` interface, no separate `Pair`/`List`/`Cons` types, no standalone `Function` type. The `Node` struct carries a `Kind` field that distinguishes syntax nodes (parser output), semantic nodes (reanimator output), and runtime nodes (values produced during evaluation).

The codebase is organized into thematically named packages:

- **`ghoul.go`**: The ghoul itself — the public API that orchestrates the three phases. Creates the exhumer, reanimator, and consumer with a shared mark counter for hygiene. Injects a `ModuleLoader` function into the consumer to handle `require` without import cycles. Returns `*bones.Node`.
- **`cmd/ghoul/main.go`**: CLI interface supporting both REPL mode and file execution.
- **`cmd/wraith/main.go`**: CLI for the wraith tool — possesses Go packages and generates sarcophagi.
- **`bones/`**: The unified type system — `node.go` defines `*Node` with all node kinds, and `location.go` defines `CodeLocation`, `SourcePosition`, and `MacroExpansionLocation`. Foundation package with no ghoul imports.
- **`exhumer/`**: Digs up structure from raw text — lexer and yacc-based parser for Lisp syntax. Produces `*bones.Node` trees (ListNode with Children) with `SourcePosition` set on parsed nodes.
- **`reanimator/`**: Brings macromancy macros to life — walks the `*Node` tree, processes `define-syntax` forms to register macros, expands macro calls, then translates the fully-expanded tree into semantic nodes (`CallNode`, `LambdaNode`, `DefineNode`, `CondNode`, `BeginNode`, `SetNode`, `RequireNode`). Uses a sub-evaluator (via `consume`) for general transformer bodies.
- **`consume/`**: How the ghoul feeds — bytecode compiler and stack VM with proper tail call optimization. Files: `evaluator.go` (wrapper, translation from syntax to semantic nodes), `compiler.go` (AST to bytecode), `bytecode.go` (opcode definitions, CodeObject), `vm.go` (stack-based VM execution loop), `cps_evaluator.go` (entry points: ConsumeNodes, EvalSubExpression), `environment.go` (scope maps storing `*Node` values), `require.go` (require form handling), `module.go` (module loading state and exports), `suggest.go` (Levenshtein-based typo suggestions).
- **`macromancy/`**: The dark arts — `macro.go` (pattern matching and transformer construction), `syntax_object.go` (hygiene via `SyntaxObject` wrapping), `marks.go` (mark types for hygienic expansion). Nested ellipsis and wildcard (`_`) patterns are supported.
- **`tome/`**: The book of spells — standard library functions (`car`, `cdr`, `cons`, `list`, `+`, `-`, `eq?`, `map`, `filter`, `syntax-match?`, `assoc`, etc.).
- **`mummy/`**: Wraith support — sarcophagus registry (`RegisterSarcophagus`, `LookupSarcophagus`) and conversion functions (`bytes`, `int-slice`, `float-slice`, `go-nil`, `string-from-bytes`). Mummy values are stored as `MummyNode` in `*bones.Node` (Kind: `MummyNode`, `ForeignVal` holds the Go value, `TypeNameV` holds the type name).
- **`wraith/`**: Code generation tool that analyzes Go packages and generates sarcophagus packages with wrapped functions, struct constructors, interface method wrappers, and callback adapters.
- **`engraving/`**: Carved records — configurable logging with TRACE/DEBUG/WARN levels. Does not import `bones` — uses `any` args with a `reprAble` interface check to format values.
- **`prelude/`**: The standard prelude (`prelude.ghl`) defining `let`, `let*`, `syntax-case`, `when`, and `unless` macros.

## Prerequisites

This project requires:
- Go 1.25+ with modules support

## Development Commands

### Building
```bash
# Generate parser + stdlib sarcophagi
go generate ./...

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

### Unified Node Type
All values are `*bones.Node`. The `Kind` field determines the node's role:

**Syntax nodes** (produced by the exhumer):
- `NilNode` — empty list / void value; singleton `bones.Nil`
- `IntegerNode`, `FloatNodeKind`, `StringNode`, `BooleanNode` — literals
- `IdentifierNode` — identifiers; `Name` holds the name, `Marks` (non-nil map) indicates a scoped/hygienic identifier
- `QuoteNode` — quoted datum in `Quoted` field
- `ListNode` — S-expression list; `Children` holds elements, `DottedTail` for improper lists

**Semantic nodes** (produced by the reanimator's translation step):
- `DefineNode`, `SetNode`, `LambdaNode`, `CondNode`, `BeginNode`, `CallNode`, `RequireNode`
- These reuse `Children` for operands, plus `Params` (ParamSpec) for lambda and `Clauses` ([]*CondClause) for cond

**Runtime nodes** (produced during evaluation):
- `FunctionNode` — closures; `FuncVal` holds `*func([]*Node, Evaluator) (*Node, error)`
- `ForeignNode` — arbitrary Go values in `ForeignVal`
- `MummyNode` — wrapped Go values with type metadata; `ForeignVal` + `TypeNameV`
- `SyntaxObjectNode` — hygiene wrapper; `Quoted` holds the wrapped datum, `Marks` holds marks

### Three-Phase Processing
1. **Exhume** (`exhumer/`): Parse source text into `*bones.Node` trees (ListNode with Children) with source positions.
2. **Reanimate** (`reanimator/`): Walk the node tree, process `define-syntax` forms to register macros, expand macro calls, strip all macro-related forms, and translate the result into semantic nodes (`CallNode`, `LambdaNode`, etc.) for the consumer. General transformer bodies are pre-expanded then evaluated through a sub-evaluator.
3. **Consume** (`consume/`): Compile semantic nodes to bytecode and execute on a stack-based VM.

### Consumer (Evaluator)
- Compiles AST to bytecode (`compiler.go`), then runs on a stack VM (`vm.go`) with pre-allocated value stack and call frames
- Two entry points:
  - `ConsumeNodes([]*bones.Node)` — main pipeline entry point, compiles and runs semantic nodes produced by the reanimator
  - `EvaluateNode(*bones.Node)` — translates a syntax node tree to semantic nodes then compiles and runs; used internally by the reanimator for macro transformer evaluation
- `EvalSubExpression(*bones.Node)` — evaluates a single node with a fresh VM
- The consumer does NOT handle `define-syntax` or macro expansion — that is the reanimator's job
- Module loading uses a `ModuleLoader` function injected by `ghoul.go` — the loader runs the full pipeline (parse -> expand -> translate -> evaluate) to avoid import cycles between `consume` and `ghoul`
- Environment scope maps use `scopeKey` (Name + canonical marks string) as keys and store `*Node` values directly
- Lookup fallback: scoped identifiers with marks fall back to name-only lookup for macro-introduced references to existing bindings

### Hygienic Macros
- Mark-based hygiene: each expansion gets a fresh mark (uint64 counter shared between reanimator and consumer)
- For `syntax-rules`: template identifiers not in pattern vars and not bound at definition site get the mark
- For general transformers: input is pre-marked, output is marked again — toggle semantics cancels marks on input-originated identifiers
- `SyntaxObjectNode` wraps leaf nodes with marks during expansion; list tree structure is preserved
- `syntax-rules` supports: multiple clauses, nested ellipsis (`(var val) ...`), wildcard (`_`), literals
- `syntax-case` is defined in the prelude as a general transformer macro

### Reanimator (Macro Expander)
- Walks the node tree top-down, processing `define-syntax` and expanding macro calls
- After expansion, translates the fully-expanded syntax node tree into semantic nodes for the consumer
- Maintains scoped macro bindings (parent-chain for inner `define-syntax` in lambda/begin)
- For `syntax-rules`: builds transformer directly via `macromancy.BuildSyntaxRulesTransformer`
- For general transformers: pre-expands the transformer expression, evaluates it to get a function node, then invokes it with mark-based hygiene during macro call expansion
- Returns original nodes unchanged when no macros are present, preserving source positions
- `containsMacroCall` check avoids unnecessary tree rebuilding

### Error Messages
- `EvaluationError` includes source location from `Node.Loc` (set by exhumer)
- `SourcePosition` carries optional `Filename` — when present, `SourceContext()` reads the file to show surrounding lines with a caret
- `MacroExpansionLocation` points back to the macro call site
- `suggestIdentifiers` provides Levenshtein-based typo suggestions for undefined identifiers
- `NodeTypeName()` gives human-readable type names in error messages — internal Go types never leak into user-facing errors

### Wraith Tool
- Generates sarcophagus packages (not code in the target package)
- Mummy values stored as `MummyNode` in `*bones.Node` — `ForeignVal` holds the Go value, `TypeNameV` holds the type name
- Methods registered as `typename-methodname` (e.g., `person-getage`)
- Struct constructors as `make-typename` (e.g., `make-person`)
- Callback adapters wrap Ghoul function nodes in Go function closures
- Variadic functions consume remaining args into a slice
- Detects unwrappable types (channels, maps) and fails unless `--skip-unwrappable` is set
- `Environment` type alias exported from `consume` for use in generated `RegisterFunctions`

### Dependency Graph
```
bones           (foundation — unified *Node type, no ghoul imports)
  |
exhumer         -> bones
macromancy      -> bones
consume         -> bones
tome            -> bones, consume
mummy           (sarcophagus registry, standalone)
reanimator      -> bones, consume
ghoul           -> all packages
```

## Conventions

- All packages use undead/occult themed names
- No reflection in generated wraith code
- TDD approach for new features
- Comments should explain *why*, not restate *what* the code does
- Error messages use `NodeTypeName()` for human-readable type names, not `%T`
- Go naming conventions: camelCase variables, no snake_case
- Breaking changes are acceptable — no backward-compat shims needed
- `newEnvWithEmptyScope` copies the slice to avoid aliasing — prevents a subtle scope corruption bug in recursive functions
