package expressions

import "testing"

func TestSourcePositionLineAndColumn(t *testing.T) {
	pos := &SourcePosition{Ln: 3, Col: 7}
	if pos.Line() != 3 {
		t.Errorf("expected line 3, got %d", pos.Line())
	}
	if pos.Column() != 7 {
		t.Errorf("expected column 7, got %d", pos.Column())
	}
}

func TestSourcePositionString(t *testing.T) {
	pos := &SourcePosition{Ln: 1, Col: 10}
	if pos.String() != "1:10" {
		t.Errorf("expected '1:10', got '%s'", pos.String())
	}
}

func TestPairCarriesLocation(t *testing.T) {
	loc := &SourcePosition{Ln: 3, Col: 5}
	pair := Cons(Integer(1), NIL)
	pair.Loc = loc

	if pair.Loc == nil {
		t.Fatal("expected Loc to be set")
	}
	if pair.Loc.Line() != 3 || pair.Loc.Column() != 5 {
		t.Errorf("expected 3:5, got %s", pair.Loc.String())
	}
}

func TestConsCreatesNilLocation(t *testing.T) {
	pair := Cons(Integer(1), NIL)
	if pair.Loc != nil {
		t.Error("Cons should create pair with nil Loc by default")
	}
}

func TestSourcePositionImplementsCodeLocation(t *testing.T) {
	var loc CodeLocation = &SourcePosition{Ln: 1, Col: 1}
	if loc.Line() != 1 {
		t.Error("should implement CodeLocation")
	}
}

func TestMacroExpansionLocationString(t *testing.T) {
	callSite := &SourcePosition{Ln: 5, Col: 1}
	loc := &MacroExpansionLocation{MacroName: "my-swap", CallSite: callSite}

	if loc.Line() != 5 {
		t.Errorf("expected line from call site, got %d", loc.Line())
	}
	if loc.String() != "5:1 in expansion of 'my-swap'" {
		t.Errorf("expected '5:1 in expansion of 'my-swap'', got '%s'", loc.String())
	}
}

func TestMacroExpansionLocationImplementsCodeLocation(t *testing.T) {
	var loc CodeLocation = &MacroExpansionLocation{
		MacroName: "test",
		CallSite:  &SourcePosition{Ln: 1, Col: 1},
	}
	if loc.Line() != 1 {
		t.Error("should implement CodeLocation")
	}
}
