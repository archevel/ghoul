package bones

import (
	"fmt"
	"os"
	"strings"
)

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
	if lines == nil || sp.Ln < 1 || sp.Ln > len(lines) {
		return ""
	}
	if strings.TrimSpace(lines[sp.Ln-1]) == "" {
		return ""
	}
	return sourceContextImpl(lines, sp.Ln, sp.Col)
}

func SourceContextFromLines(lines []string, errorLine int, errorCol int) string {
	return sourceContextImpl(lines, errorLine, errorCol)
}

func sourceContextImpl(lines []string, errorLine int, errorCol int) string {
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
