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

## Building a Ghoul

Prerequisites: Go 1.25+.

Ghoul binaries are built with the **undertaker** tool. The undertaker mummifies Go packages (wrapping them for use from Ghoul) and produces a self-contained binary.

```bash
# Install the undertaker
go install github.com/archevel/ghoul/cmd/undertaker@latest

# Create a graveyard.toml listing the Go packages you want
cat > graveyard.toml <<'EOF'
[[embalm]]
package = "github.com/my/cool-lib"
EOF

# Build a custom ghoul binary (stdlib packages are included by default)
undertaker prepare my-ghoul graveyard.toml

# Run the REPL
./my-ghoul

# Run a file
./my-ghoul examples/stdlib.ghl
```

The default stdlib packages (math, strings, fmt, os, net/http, etc.) are included automatically. Use `--no-stdlib` to include only the packages listed in your `graveyard.toml`.

### graveyard.toml format

```toml
[[embalm]]
package = "net/http"
skip_unwrappable = true   # skip functions with channel/map types

[[embalm]]
package = "github.com/foo/bar"

[[embalm]]
package = "github.com/my/local-lib"
path = "/home/me/code/local-lib"   # use a local directory (adds a replace directive)
```

### Undertaker flags

| Flag | Description |
|------|-------------|
| `--no-stdlib` | Don't auto-include default stdlib mummies |
| `--no-prelude` | Omit the standard prelude (let, let*, when, unless, syntax-case) |
| `--verbose` | Verbose output during build |
| `--keep` | Preserve the `.ghoul/build/` directory after build |
| `--work-dir` | Override the build directory |

### Running the examples

To run the examples that use Go standard library functions:

```bash
# Build a ghoul with default stdlib (from the repo root)
undertaker prepare ghoul graveyard.toml

# Pure Ghoul examples (no mummies needed)
./ghoul examples/basics.ghl
./ghoul examples/macros.ghl
./ghoul examples/modules.ghl
./ghoul examples/tail_calls.ghl
./ghoul examples/pattern_matching_macro.ghl

# Examples that require stdlib mummies
./ghoul examples/stdlib.ghl
./ghoul examples/hello_server.ghoul
```

An empty `graveyard.toml` works fine — the default stdlib packages are included automatically.

### Development

```bash
# Generate the parser (only needed after changing exhumer/parser.y)
go generate ./exhumer/

# Run all tests
go test ./...

# Build the undertaker from source (for development)
go build -o undertaker ./cmd/undertaker
```

## Testing

```bash
go test ./...
```

## TODOs
- Replace `cond` with `match` keyword and have it use pattern matching
- Implement a symbol table and use integers instead of strings to compare/find the right values

## Package Structure

The ghoul feeds in three phases: **exhume** (parse) -> **reanimate** (expand macros + translate to AST) -> **consume** (evaluate).

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
| `sarcophagus` | Registry where mummies are entombed |
| `embalmer` | Mummifies Go packages, generating FFI wrappers for Ghoul |
| `undertaker` | Build tool — assembles a ghoul binary from a `graveyard.toml` |
| `prelude` | Standard macros: `let`, `let*`, `when`, `unless`, `syntax-case` |
