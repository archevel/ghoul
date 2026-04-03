package exhumer

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/bones"
)

func TestParsePreludeFile(t *testing.T) {
	// Regression test: the full prelude must parse without errors.

	// Test via os.Open + file reader (same path as ProcessFile)
	f, err := os.Open("../prelude/prelude.ghl")
	if err != nil {
		t.Fatalf("failed to open prelude: %v", err)
	}
	defer f.Close()
	res, _ := Parse(f)
	if res != 0 {
		t.Errorf("Failed to parse prelude via file reader, parse result: %d", res)
	}

	// Also test via ReadFile + StringReader (same path as test helper)
	content, err := os.ReadFile("../prelude/prelude.ghl")
	if err != nil {
		t.Fatalf("failed to read prelude: %v", err)
	}
	res2, _ := Parse(strings.NewReader(string(content)))
	if res2 != 0 {
		t.Errorf("Failed to parse prelude via string reader, parse result: %d", res2)
	}
}

func TestParseInteger(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		{"1", e.IntNode(1)},
		{"0", e.IntNode(0)},
		{"-1", e.IntNode(-1)},
		{"9223372036854775806", e.IntNode(math.MaxInt64 - 1)},
		{"9223372036854775807", e.IntNode(math.MaxInt64)},
		{"9223372036854775808", e.IntNode(math.MaxInt64)},
		{"-9223372036854775807", e.IntNode(math.MinInt64 + 1)},
		{"-9223372036854775808", e.IntNode(math.MinInt64)},
		{"-9223372036854775809", e.IntNode(math.MinInt64)},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		if !lp.First().Equiv(c.out) {
			t.Errorf("Parsed \"%s\", expected %v but got %T%v", c.in, c.out, lp.First(), lp.First())
		}
	}

}

func TestParseFloat(t *testing.T) {
	d := 4.940656458412465441765687928682213723651e-324
	max := 1.797693134862315708145274237317043567981e+308
	cases := []struct {
		in  string
		out *e.Node
	}{
		{"1.0", e.FloatNode(1.0)},
		{"0.0", e.FloatNode(0.0)},
		{"-1.0", e.FloatNode(-1.0)},
		{fmt.Sprintf("%g", max), e.FloatNode(math.MaxFloat64)},
		{fmt.Sprintf("%g", max-d), e.FloatNode(math.MaxFloat64 - d)},
		{fmt.Sprintf("%g", max+d), e.FloatNode(math.MaxFloat64)},

		{fmt.Sprintf("%g", -max), e.FloatNode(-math.MaxFloat64)},
		{fmt.Sprintf("%g", (-max)-d), e.FloatNode(-math.MaxFloat64)},
		{fmt.Sprintf("%g", (-max)+d), e.FloatNode(-math.MaxFloat64 + d)},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		if !lp.First().Equiv(c.out) {
			t.Errorf("Parsed \"%s\", expected %v but got %v", c.in, c.out, lp)
		}
	}

}

func TestParseBoolean(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		{"#t", e.BoolNode(true)},
		{"#f", e.BoolNode(false)},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		if !lp.First().Equiv(c.out) {
			t.Errorf("Parsed \"%s\", expected %v but got %v", c.in, c.out, lp)
		}
	}

}

func TestParseString(t *testing.T) {

	cases := []struct {
		in  string
		out string
	}{
		{`"some string"`, "some string"},
		{"`some\n\tstring`", "some\n\tstring"},
		{"`some\\n\\tstring`", `some\n\tstring`},
		{`"some\n\tstring"`, "some\n\tstring"},
		{`"some\\n\\tstring"`, `some\n\tstring`},
	}

	for i, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}
		actual := lp.First().StrVal

		if actual != c.out {
			t.Errorf("Test #%v:Parsed \"%s\", expected %v but got %v", i, c.in, c.out, lp)
		}
	}

}

func TestParseIdententifiers(t *testing.T) {

	cases := []struct {
		in  string
		out *e.Node
	}{
		{"anIdentifier", e.IdentNode("anIdentifier")},
		{"--", e.IdentNode("--")},
		{"-----", e.IdentNode("-----")},
		{"--1", e.IdentNode("--1")},
		{"a.", e.IdentNode("a.")},
		{SPECIAL_IDENTIFIERS, e.IdentNode(SPECIAL_IDENTIFIERS)},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		if !lp.First().Equiv(c.out) {
			t.Errorf("Parsed \"%s\", expected %v but got %v", c.in, c.out, lp)
		}
	}

}

func TestParseSpecialIdentifiers(t *testing.T) {

	cases := strings.Split(SPECIAL_IDENTIFIERS, "")

	for _, c := range cases {
		r := strings.NewReader(c)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c)
		}

		if !lp.First().Equiv(e.IdentNode(c)) {
			t.Errorf("Parsed \"%s\", expected %v but got %v", c, c, lp)
		}
	}

}

func TestParseEllipsis(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		{"(a ...)", e.NewListNode([]*e.Node{e.IdentNode("a"), e.IdentNode("...")})},
		{"(a ... b)", e.NewListNode([]*e.Node{e.IdentNode("a"), e.IdentNode("..."), e.IdentNode("b")})},
		{"(a c ... b)", e.NewListNode([]*e.Node{e.IdentNode("a"), e.IdentNode("c"), e.IdentNode("..."), e.IdentNode("b")})},
	}

	for _, c := range cases {
		res, parsed := Parse(strings.NewReader(c.in))
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		if !c.out.Equiv(lp.First()) {
			t.Errorf("Could not parse ellipsis. Expected %s, but got %s", c.out.Repr(), lp.First().Repr())
		}

	}
}

func TestParseQuotedSpecialIdentifiers(t *testing.T) {

	cases := strings.Split(SPECIAL_IDENTIFIERS, "")

	for _, c := range cases {
		r := strings.NewReader("'" + c)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c)
		}
		q := lp.First()
		if q.Kind != e.QuoteNode {
			t.Errorf("Parsed \"'%s\", expected a quoted expression, but got %s", c, q.Repr())
		}

		if q.Quoted == nil || q.Quoted.IdentName() != c {
			t.Errorf("Parsed \"%s\", expected quoted %v but got %v", c, c, lp)
		}

	}

}

func TestParseNestedQuotes(t *testing.T) {
	c := "''c"
	r := strings.NewReader(c)
	res, parsed := Parse(r)
	lp := parsed.Expressions
	if res != 0 {
		t.Errorf("Parser failed to parse \"%s\"", c)
	}
	q := lp.First()
	if q.Kind != e.QuoteNode || q.Quoted == nil {
		t.Fatalf("expected outer QuoteNode, got %s", q.Repr())
	}
	q2 := q.Quoted
	if q2.Kind != e.QuoteNode || q2.Quoted == nil {
		t.Fatalf("expected inner QuoteNode, got %s", q2.Repr())
	}
	if q2.Quoted.IdentName() != "c" {
		t.Errorf("expected quoted c, got %s", q2.Quoted.Repr())
	}
}

func TestParseLists(t *testing.T) {

	cases := []struct {
		in  string
		out *e.Node
	}{
		{"()", e.Nil},
		{"(foo)", e.NewListNode([]*e.Node{e.IdentNode("foo")})},
		{"(bar foo)", e.NewListNode([]*e.Node{e.IdentNode("bar"), e.IdentNode("foo")})},
		{"(1 #f)", e.NewListNode([]*e.Node{e.IntNode(1), e.BoolNode(false)})},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		actual := lp.First()
		if !actual.Equiv(c.out) {
			t.Errorf("List mismatch for %q: expected %s, got %s", c.in, c.out.Repr(), actual.Repr())
		}
	}

}

func TestParsePairs(t *testing.T) {

	cases := []struct {
		in  string
		out *e.Node
	}{
		{"(a . b)", &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IdentNode("a")}, DottedTail: e.IdentNode("b")}},

		{"(`bar` . `foo`)", &e.Node{Kind: e.ListNode, Children: []*e.Node{e.StrNode("bar")}, DottedTail: e.StrNode("foo")}},
		{"(1 . #f)", &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IntNode(1)}, DottedTail: e.BoolNode(false)}},
		{"(1 2 . #f)", &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IntNode(1), e.IntNode(2)}, DottedTail: e.BoolNode(false)}},
		{"(a b c . d)", &e.Node{Kind: e.ListNode, Children: []*e.Node{e.IdentNode("a"), e.IdentNode("b"), e.IdentNode("c")}, DottedTail: e.IdentNode("d")}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Fatalf("Parser failed to parse \"%s\"", c.in)
		}

		actual := lp.First()
		if !actual.Equiv(c.out) {
			t.Errorf("Pair mismatch for %q: expected %s, got %s", c.in, c.out.Repr(), actual.Repr())
		}

	}
}

func TestParseTwoSymbols(t *testing.T) {

	cases := []struct {
		in  string
		out *e.Node
	}{
		{"'foo 'mmm", e.NewListNode([]*e.Node{e.QuoteNodeVal(e.IdentNode("foo")), e.QuoteNodeVal(e.IdentNode("mmm"))})},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Fatalf("Parser failed to parse \"%s\"", c.in)
		}

		actual := lp

		if !actual.Equiv(c.out) {
			t.Errorf("Expected output |%v| to be equivalent to |%v|, but it was not", actual.Repr(), c.out.Repr())
		}

	}
}

func TestParseIgnoresInitialHashbangLine(t *testing.T) {
	cases := []struct {
		in  string
		out *e.Node
	}{
		{"#!/usr/bin/ghoul", e.Nil},
		{"#!/usr/bin/ghoul\n", e.Nil},
		{"\n#!/usr/bin/ghoul\n", e.Nil},
		{";; foo bar \n#!/usr/bin/ghoul\n", e.Nil},
		{"#!/usr/bin/ghoul\n77", e.NewListNode([]*e.Node{e.IntNode(77)})},
		{"\n#!/usr/bin/ghoul\n`foo`", e.NewListNode([]*e.Node{e.StrNode("foo")})},
		{";; foo bar \n#!/usr/bin/ghoul\n(bar)", e.NewListNode([]*e.Node{e.NewListNode([]*e.Node{e.IdentNode("bar")})})},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Fatalf("Parser failed to parse \"%s\"", c.in)
		}

		actual := lp

		if !actual.Equiv(c.out) {
			t.Errorf("Expected output |%v| to be equivalent to |%v|, but it was not", actual.Repr(), c.out.Repr())
		}

	}

}

func TestParserRegistersPositionOfNodes(t *testing.T) {
	cases := []struct {
		in          string
		expectedPos Position
	}{
		{"\n mmm", Position{2, 2}},
		{"123 (mmm foo)", Position{1, 1}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		if res != 0 {
			t.Fatalf("Parser failed to parse %q", c.in)
		}

		node := parsed.Expressions
		if node == nil || node.Loc == nil {
			t.Errorf("Expected node with location for %q, got nil", c.in)
			continue
		}
		actualPos := Position{node.Loc.Line(), node.Loc.Column()}
		if actualPos != c.expectedPos {
			t.Errorf("Expected position of top-level node in %q to be %q, was %q", c.in, c.expectedPos, actualPos)
		}
	}
}

func TestInnerNodesHavePositions(t *testing.T) {
	r := strings.NewReader("\n\n(mmm foo)")
	res, parsed := Parse(r)
	if res != 0 {
		t.Fatal("Parser failed to parse")
	}

	node := parsed.Expressions
	if node == nil || len(node.Children) == 0 {
		t.Fatal("Expected at least one child")
	}
	innerList := node.Children[0]
	if innerList.Kind != e.ListNode {
		t.Fatalf("Expected inner ListNode, got kind %d", innerList.Kind)
	}
	if innerList.Loc == nil {
		t.Fatal("Expected location on inner list")
	}
	// Inner list (mmm foo) starts at mmm which is at line 3, col 2
	actualPos := Position{innerList.Loc.Line(), innerList.Loc.Column()}
	expectedPos := Position{3, 2}
	if actualPos != expectedPos {
		t.Errorf("Expected position %q, was %q", expectedPos, actualPos)
	}
}

func TestParseTestFileLevelGhl(t *testing.T) {
	// Test parsing the test_file_level.ghl file which was failing
	f, err := os.Open("test_file_level.ghl")
	if err != nil {
		t.Fatalf("failed to open test_file_level.ghl: %v", err)
	}
	defer f.Close()

	filename := "test_file_level.ghl"
	res, parsed := ParseWithDebug(f, &filename)
	if res != 0 {
		t.Errorf("Failed to parse test_file_level.ghl, parse result: %d", res)
	}

	// Also test via ReadFile + StringReader
	content, err := os.ReadFile("test_file_level.ghl")
	if err != nil {
		t.Fatalf("failed to read test_file_level.ghl: %v", err)
	}
	res2, _ := Parse(strings.NewReader(string(content)))
	if res2 != 0 {
		t.Errorf("Failed to parse test_file_level.ghl via string reader, parse result: %d", res2)
	}

	// Sanity check: should have parsed some expressions
	if parsed.Expressions == nil || parsed.Expressions == e.Nil {
		t.Error("Expected non-empty parse result")
	}
}
