# Ghoul - an undead themed lisp interpreter
Ghoul is a lisp interpreter that aims to be simple to understand while being a bit more advanced than a naive interpreter.

#### Explicit goals:
- [x] Undead themed names
- [ ] Easy to understand code base
- [x] Support simple integration of Golang code
- [x] Proper tail call optimizations
- [x] Hygenic macro support

#### Non-goals:
- Fast code execution - It should be easy enough to drop down to golang code if something needs speed.
- Comprehensive standard library implementation written in Ghoul - it should rely on Golang implementations as much as possible.
- Special handling of datastructures in the interpreter - special syntax for e.g. maps should be handled by macromancy!

## Building

Prerequisites: Go 1.25+.

```bash
# Generate parser + stdlib mummies + stdlib.go
go generate ./...

# Build the ghoul binary
go build -o ghoul ./cmd/ghoul/

# Run the REPL
./ghoul

# Run a file
./ghoul examples/hello_server.ghoul
```

The `go generate` step runs two generators:
1. `goyacc` to generate the parser from `exhumer/parser.y`
2. `scripts/possess.sh` to mummify Go packages listed in `sarcophagus.txt`

To add a Go package to the stdlib, add its import path to `sarcophagus.txt` and re-run `go generate ./...`. Third-party packages work too — just `go get` them first.

## Testing

```bash
go test ./...
```

## TODOs
- Replace `cond` with `match` keyword and have it use pattern matching
- Implement a symbol table and use integers instead of strings to compare/find the right values

## Package Structure

The ghoul feeds in three phases: **exhume** (parse) → **reanimate** (expand macros + translate to AST) → **consume** (evaluate).

All packages follow an undead/occult naming theme:

| Package | Role |
|---------|------|
| `ghoul` | The creature itself — public API orchestrating the three phases |
| `bones` | The unified `*Node` type — AST nodes, values, and runtime data |
| `exhumer` | Digs up structure from raw text — lexer and parser |
| `reanimator` | Brings macros to life — expansion + translation to semantic AST |
| `consume` | How the ghoul feeds — bytecode compiler and stack VM with tail call optimization |
| `macromancy` | The dark arts — macro pattern matching and hygienic expansion |
| `tome` | The book of spells — standard library functions |
| `engraving` | Carved records — logging |
| `sarcophagus` | Registry (sarcophagus) for wraith-generated mummies |
| `wraith` | Possesses Go packages and generates mummies (FFI wrappers) |
| `prelude` | Standard macros: `let`, `let*`, `when`, `unless`, `syntax-case` |
