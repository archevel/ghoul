# Ghoul - an undead themed lisp interpreter
Ghoul is a lisp interpreter that aims to be simple to understand while being a bit more advanced than a naive interpreter. 

#### Explicit goals:
- [ ] Easy to understand code base
- [ ] Support simple integration of Golang code
- [x] Proper tail call optimizations
- [ ] Hygenic macro support  

#### Non-goals:
- Fast code execution - It should be easy enough to drop down to golang code if something needs speed.
- Comprehensive standard library implementation written in Ghoul - it should rely on Golang implementations as much as possible.
- Special handling of datastructures in the interpreter - special syntax for e.g. maps should be handled by macromancy! 


## Notes:
TODO: Add a Foreign expression type that can be used for arbitrary structs  
TODO: Write "wraith" - tool for wrapping Go code for Ghoul use  
TODO: Wrap Golang standard library so it is callable from ghoul  
TODO: Implement module system so that code can be included when required (as opposed to including it all upfront).  
TODO: Make error messages contain line and column of failed expression. Derived expressions should as far as possible point to their original version.  
TODO: Use fn instead of lambda?  
TODO: Use do instead of begin?  
TODO: Use def instead of define?  
TODO: Implement macros in the macromancy pakage!  
TODO: Clean up tests into separate files with distinct areas  
TODO: Implement an error printer  


