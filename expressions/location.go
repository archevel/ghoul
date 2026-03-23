package expressions

import (
	"fmt"
)

func TypeName(expr Expr) string {
	switch expr.(type) {
	case Boolean:
		return "boolean"
	case Integer:
		return "integer"
	case Float:
		return "float"
	case String:
		return "string"
	case Identifier:
		return "identifier"
	case ScopedIdentifier:
		return "identifier"
	case *Quote:
		return "quoted expression"
	case *Pair:
		return "list"
	case Pair:
		return "list"
	case nilList:
		return "empty list"
	case *Foreign:
		return "foreign value"
	case Foreign:
		return "foreign value"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

type CodeLocation interface {
	Line() int
	Column() int
	String() string
}

type SourcePosition struct {
	Ln  int
	Col int
}

func (sp *SourcePosition) Line() int   { return sp.Ln }
func (sp *SourcePosition) Column() int { return sp.Col }
func (sp *SourcePosition) String() string {
	return fmt.Sprintf("%d:%d", sp.Ln, sp.Col)
}

// MacroExpansionLocation points back to where the macro was invoked,
// so errors in expanded code can be traced to the call site.
type MacroExpansionLocation struct {
	MacroName string
	CallSite  CodeLocation
}

func (mel *MacroExpansionLocation) Line() int   { return mel.CallSite.Line() }
func (mel *MacroExpansionLocation) Column() int { return mel.CallSite.Column() }
func (mel *MacroExpansionLocation) String() string {
	return fmt.Sprintf("%s in expansion of '%s'", mel.CallSite.String(), mel.MacroName)
}
