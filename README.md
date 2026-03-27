# Ghoul - an undead themed lisp interpreter
Ghoul is a lisp interpreter that aims to be simple to understand while being a bit more advanced than a naive interpreter.

#### Explicit goals:
- [ ] Undead themed names
- [ ] Easy to understand code base
- [ ] Support simple integration of Golang code
- [x] Proper tail call optimizations
- [x] Hygenic macro support

#### Non-goals:
- Fast code execution - It should be easy enough to drop down to golang code if something needs speed.
- Comprehensive standard library implementation written in Ghoul - it should rely on Golang implementations as much as possible.
- Special handling of datastructures in the interpreter - special syntax for e.g. maps should be handled by macromancy!


## Notes:
~~TODO: Add a Foreign expression type that can be used for arbitrary structs~~
~~TODO: Write "wraith" - tool for wrapping Go code for Ghoul use~~
TODO: Wrap Golang standard library so it is callable from ghoul
TODO: Implement module system so that code can be included when required (as opposed to including it all upfront).
~~TODO: Make error messages contain line and column of failed expression. Derived expressions should as far as possible point to their original version.~~
TODO: Use `fn` instead of `lambda`?
TODO: Use `do` instead of `begin`?
TODO: Use `def` instead of `define`?
~~TODO: Implement macros in the macromancy pakage!~~
~~TODO: Make macro elipsis associate to preceeding `identifier` so more complex code patterns can be expanded.~~
~~TODO: Add keyword literals to macro matching~~
~~TODO: Implement support for multiple elipsis in macro matching and in expansion bodies.~~
~~TODO: Ensure macros propagate source location to expanded code~~
~~TODO: Implement pathological macros (like in racket macro docs 16.1), e.g. (swap tmp other).~~
TODO: Clean up tests into separate files with distinct areas
~~TODO: Implement an error printer~~
~~TODO: Make `Pair` struct private and replace usages with a `Cons(Expr, Expr)` function returning a `*pair`.~~
~~TODO: Make `List.Tail()` return `(List, bool)` and move usages of `tail()` to that~~
~~TODO: Make `Pair` interface have a `First() Expr` and `Second() Expr` method.~~
TODO: Replace `cond` with `match` keyword and have it use pattern matching.
TODO: Implement a symbol table and use integers instead of strings to compare/find the right values
~~TODO: Add logging (to lowest log level) essentially everywhere and make sure it is disabled in `ghoul` command unless some param is given.~~

## Package Structure

The ghoul feeds in three phases: **exhume** (parse) → **reanimate** (expand macros + translate to AST) → **consume** (evaluate).

All packages follow an undead/occult naming theme:

| Package | Role |
|---------|------|
| `ghoul` | The creature itself — public API orchestrating the three phases |
| `bones` | The unified `*Node` type — AST nodes, values, and runtime data |
| `exhumer` | Digs up structure from raw text — lexer and parser |
| `reanimator` | Brings macros to life — expansion + translation to semantic AST |
| `consume` | How the ghoul feeds — CPS evaluator with tail call optimization |
| `macromancy` | The dark arts — macro pattern matching and hygienic expansion |
| `tome` | The book of spells — standard library functions |
| `engraving` | Carved records — logging |
| `mummy` | FFI type registry for wraith-generated sarcophagi |
| `wraith` | Possesses Go packages and generates sarcophagi (FFI wrappers) |
| `prelude` | Standard macros: `let`, `let*`, `when`, `unless`, `syntax-case` |
