package expressions

import (
	"fmt"
	"os"
	"strings"
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
	SourceContext() string
}

type SourcePosition struct {
	Ln       int
	Col      int
	Filename *string
}

func (sp *SourcePosition) Line() int   { return sp.Ln }
func (sp *SourcePosition) Column() int { return sp.Col }
func (sp *SourcePosition) String() string {
	if sp.Filename != nil {
		return fmt.Sprintf("%s:%d:%d", *sp.Filename, sp.Ln, sp.Col)
	}
	return fmt.Sprintf("%d:%d", sp.Ln, sp.Col)
}
func (sp *SourcePosition) SourceContext() string {
	if sp.Filename == nil {
		return ""
	}
	lines := readLinesFromFile(*sp.Filename)
	if lines == nil || sp.Ln > len(lines) {
		return ""
	}
	if strings.TrimSpace(lines[sp.Ln-1]) == "" {
		return ""
	}
	return sourceContext(lines, sp.Ln, sp.Col)
}

// sourceContext builds a source snippet showing the error line with a caret,
// the enclosing expression (found by scanning backwards for unmatched parens),
// and up to 2 lines after the error.
func sourceContext(lines []string, errorLine int, errorCol int) string {
	if len(lines) == 0 || errorLine < 1 || errorLine > len(lines) {
		return ""
	}

	startLine := findEnclosingExprStart(lines, errorLine)
	endLine := errorLine + 2
	if endLine > len(lines) {
		endLine = len(lines)
	}

	lineNumWidth := len(fmt.Sprintf("%d", endLine))

	var b strings.Builder
	for i := startLine; i <= endLine; i++ {
		prefix := fmt.Sprintf("  %*d | ", lineNumWidth, i)
		b.WriteString(prefix)
		b.WriteString(lines[i-1])
		b.WriteString("\n")
		if i == errorLine {
			padding := len(prefix) + errorCol - 1
			b.WriteString(strings.Repeat(" ", padding))
			b.WriteString("^")
			b.WriteString("\n")
		}
	}
	return b.String()
}

// findEnclosingExprStart scans backwards from the error line, counting
// parens to find where an enclosing expression starts. Continues up
// to 2 nesting levels to give broader context.
func findEnclosingExprStart(lines []string, errorLine int) int {
	depth := 0
	enclosingsFound := 0
	start := errorLine

	for i := errorLine - 1; i >= 0; i-- {
		line := lines[i]
		for j := len(line) - 1; j >= 0; j-- {
			switch line[j] {
			case ')':
				depth++
			case '(':
				depth--
				if depth < 0 {
					start = i + 1
					enclosingsFound++
					if enclosingsFound >= 2 {
						return start
					}
				}
			}
		}
	}

	if enclosingsFound > 0 {
		return start
	}

	s := errorLine - 2
	if s < 1 {
		s = 1
	}
	return s
}

func readLinesFromFile(filename string) []string {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	return strings.Split(string(data), "\n")
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
func (mel *MacroExpansionLocation) SourceContext() string {
	return mel.CallSite.SourceContext()
}
