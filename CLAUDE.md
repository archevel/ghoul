# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ghoul is an undead-themed Lisp interpreter written in Go that aims to be simple to understand while being more advanced than a naive interpreter. It features proper tail call optimizations and hygienic macro support through the "macromancy" package.

## Architecture

The codebase is organized into several key packages:

- **`ghoul.go`**: Main entry point that orchestrates the parsing, macro transformation, and evaluation pipeline
- **`cmd/ghoul/main.go`**: CLI interface supporting both REPL mode and file execution
- **`parser/`**: Lexer and yacc-based parser for Lisp syntax (uses `parser.y` with goyacc)
- **`evaluator/`**: Core expression evaluation engine with tail call optimization
- **`expressions/`**: Expression type definitions and interfaces
- **`macromancy/`**: Macro expansion system for hygienic macros
- **`logging/`**: Configurable logging infrastructure

## Prerequisites

This project requires:
- Go 1.25+ with modules support
- `goyacc` tool for parser generation: `go install golang.org/x/tools/cmd/goyacc@latest`

## Development Commands

### Building
```bash
# Generate parser from yacc grammar (required before building)
go generate ./parser

# Build the main executable
go build -o ghoul ./cmd/ghoul

# Build all packages
go build ./...
```

### Logging Levels
```bash
# Quiet (WARN level only)
./ghoul  # Default

# Verbose (includes TRACE level for detailed evaluator steps)
# Use VerboseLogger in code: ghoul.NewLoggingGhoul(logging.VerboseLogger)
```

### Testing
```bash
# Run all tests (note: some test formatting issues exist but build succeeds)
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test ./evaluator
```

### Running
```bash
# Start REPL
./ghoul

# Execute a Ghoul file
./ghoul filename.ghoul
```

## Key Implementation Details

- **Tail Call Optimization**: The evaluator uses continuation-passing style with a continuation stack for proper tail calls
- **Macro System**: The macromancy package handles hygienic macro expansion before evaluation
- **Environment**: Uses lexical scoping with environment chains for variable lookup
- **Built-in Functions**: Core functions like `eq?`, `and`, `<`, `mod`, `+`, `println` are registered in the default environment
- **Parser Generation**: Uses goyacc to generate parser from `parser/parser.y` grammar file

## Special Notes

- The parser requires generation via `go generate ./parser` before building
- The project follows Go package conventions with clear separation of concerns
- Error handling focuses on parse-time and runtime errors with plans for better error messages with source locations
- The evaluator supports special forms: `cond`, `else`, `begin`, `lambda`, `define`, `set!`

## Modernization Notes

**✅ Completed:**
- Replaced `interface{}` with `any` (Go 1.18+) for better readability
- Migrated to structured logging with `slog` (Go 1.21+) for better observability and performance
- Added TRACE level logging for detailed evaluator execution flow (below DEBUG level)

**❌ Not Recommended:**
- **Generics for type safety**: The dynamic nature of Lisp expressions conflicts fundamentally with static generics. The architecture relies on type switches across different expression types (Boolean, Integer, String, etc.) in a single interface, which generics would break. Estimated 6-8 weeks of work with high risk and questionable benefit.
- **`slices` package migration**: Current slice operations are in performance-critical hot paths (evaluator continuation stack, environment creation). The `append()` operations in `evaluator.go:65` are called for every expression evaluation. Replacing with `slices` package functions would add function call overhead to the core evaluation loop with minimal readability benefit. Risk > reward for this performance-sensitive interpreter.

**✅ Error Wrapping Improvements Completed:**
- Enhanced parser error context with parse result codes for better debugging
- Improved environment error messages for `define`, `set!`, and undefined identifier errors
- Added comprehensive macro error context with specific failure details
- Implemented type assertion safety for all built-in functions (`and`, `<`, `mod`, `+`) with descriptive error messages
- Fixed nil pointer crashes in macro error handling
- Updated all test cases to expect improved error messages
- All tests now pass with significantly better debugging experience

**✅ Context Support Implementation Completed:**
- Added `context.Context` support for cancellation and timeout control
- Implemented `EvaluateWithContext()` and `ProcessWithContext()` methods
- Maintains full backward compatibility with existing `Evaluate()` and `Process()` methods
- Context checking integrated into the main evaluation loop for responsive cancellation
- Comprehensive test coverage including happy path, cancellation, timeout, and complex programs
- **Key Benefits:**
  - **Infinite loop protection**: Cancel runaway Lisp programs gracefully
  - **Timeout support**: Enforce evaluation time limits for server environments
  - **Production readiness**: Makes Ghoul suitable for production use with proper resource management
  - **Go ecosystem integration**: Works seamlessly with HTTP handlers, gRPC, and other Go services

**❌ String Building Optimization Analysis:**
- **Not recommended for Ghoul**: Current string concatenations are small (2-3 pieces) and already optimal
- Go compiler optimizes simple `+` concatenation for small strings better than `strings.Builder`
- `Pair.Repr()` already uses `bytes.Buffer` appropriately for complex cases
- `strings.Builder` would add overhead, not reduce it, for typical Lisp expression rendering
- **Conclusion**: Current implementation is well-optimized for Lisp interpreter use cases

**✅ Type Assertion Safety Improvements Completed:**
- **Fixed all high-risk panicking type assertions using TDD approach**
- **Priority 1**: `evaluator/environment.go:49` - Assignment with non-identifier variables now returns proper error instead of panicking
- **Priority 2**: `macromancy/macro.go` - Multiple macro processing type assertions now safely handle malformed patterns
- **Priority 3**: `evaluator/evaluator.go:151` - Analysis confirmed this assertion is already protected by logic flow
- **Priority 4**: `macromancy/macro.go:166` - `splitListAt` function now safely handles non-Pair splitPoints
- **Priority 5**: `logging/logger.go:29` - slog level replacer now uses defensive programming against type mismatches
- **Test Coverage**: Added comprehensive test suites for each fix using Test-Driven Development
- **Result**: Eliminated runtime panic risks from user input, replaced with proper error messages

**✅ Error Chaining Improvements Completed:**
- **Enhanced error context throughout the codebase using TDD approach**
- **Fixed evaluation error chaining** in `ghoul.go:48-52` - Processing errors now include proper context with underlying causes
- **Fixed macro error chaining** in `macromancy/macromancy.go` - Macro definition errors now propagate properly with context
- **Updated context tests** in `ghoul_context_test.go` - Context cancellation/timeout tests now use `errors.Is()` for proper error matching
- **Added comprehensive test coverage** in `ghoul_error_chaining_test.go` - Tests validate both processing and macro error chaining
- **Modified Macromancer interface** - `Transform` method now returns `(e.Expr, error)` instead of just `e.Expr` for proper error propagation
- **Test Coverage**: All error chaining scenarios tested with TDD methodology
- **Result**: Significantly improved debugging experience with proper error context and chain unwrapping

**✅ Wraith Tool Implementation Completed:**
- **Undead-themed package possession** - Successfully implemented the wraith command-line tool with `wraith possess <package>` interface
- **Mummy wrapper generation** - Creates `<packagename>_mummy.go` files containing wrapped functions with full undead theming
- **Possession protection** - Refuses to possess packages that already have mummy files (prevents double-possession)
- **Comprehensive type mapping system** - Maps Go primitives (`int`, `string`, `bool`, `float`) to Ghoul expressions, complex types to Foreign
- **Template-based code generation** - Generates complete wrapper functions with argument conversion, function calls, and result handling
- **Error propagation support** - Automatically handles Go errors and propagates them as Ghoul errors with proper context
- **Method and function support** - Handles both standalone functions and struct methods (value and pointer receivers)
- **Thematic code generation** - Generated mummy files include undead-themed comments and function descriptions
- **CLI interface** - `wraith possess <package-path> [-v]` command with verbose mode and proper error messages
- **Package analysis** - Uses go/packages and go/ast for robust Go package parsing and type information extraction
- **Generated registration** - Automatically creates `RegisterFunctions()` to "awaken" all mummified functions in Ghoul environments
- **Test Coverage**: Successfully tested with sample package containing various function signatures, methods, and types
- **Result**: Enables automatic possession and mummification of any Go package for use in Ghoul, fulfilling the original README requirement

**🔄 Future Considerations:**
- Additional string concatenation optimizations with `strings.Builder` (already analyzed - not recommended for small strings)
- Enhanced slice handling (convert primitive slices to Ghoul Lists)
- Support for Go interfaces and function types in wraith generation