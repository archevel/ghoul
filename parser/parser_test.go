package parser

import (
	"fmt"
	"math"
	"strings"
	"testing"

	e "github.com/archevel/ghoul/expressions"
)

func TestParseInteger(t *testing.T) {
	cases := []struct {
		in  string
		out e.Integer
	}{
		{"1", e.Integer(1)},
		{"0", e.Integer(0)},
		{"-1", e.Integer(-1)},
		{"9223372036854775806", e.Integer(math.MaxInt64 - 1)},
		{"9223372036854775807", e.Integer(math.MaxInt64)},
		{"9223372036854775808", e.Integer(math.MaxInt64)},
		{"-9223372036854775807", e.Integer(math.MinInt64 + 1)},
		{"-9223372036854775808", e.Integer(math.MinInt64)},
		{"-9223372036854775809", e.Integer(math.MinInt64)},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		if lp.First() != c.out {
			t.Errorf("Parsed \"%s\", expected %v but got %T%v", c.in, c.out, lp.First(), lp.First())
		}
	}

}

func TestParseFloat(t *testing.T) {
	d := 4.940656458412465441765687928682213723651e-324
	max := 1.797693134862315708145274237317043567981e+308
	cases := []struct {
		in  string
		out e.Float
	}{
		{"1.0", e.Float(1.0)},
		{"0.0", e.Float(0.0)},
		{"-1.0", e.Float(-1.0)},
		{fmt.Sprintf("%g", max), e.Float(math.MaxFloat64)},
		{fmt.Sprintf("%g", max-d), e.Float(math.MaxFloat64 - d)},
		{fmt.Sprintf("%g", max+d), e.Float(math.MaxFloat64)},

		{fmt.Sprintf("%g", -max), e.Float(-math.MaxFloat64)},
		{fmt.Sprintf("%g", (-max)-d), e.Float(-math.MaxFloat64)},
		{fmt.Sprintf("%g", (-max)+d), e.Float(-math.MaxFloat64 + d)},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		if lp.First() != c.out {
			t.Errorf("Parsed \"%s\", expected %v but got %v", c.in, c.out, lp)
		}
	}

}

func TestParseBoolean(t *testing.T) {
	cases := []struct {
		in  string
		out e.Boolean
	}{
		{"#t", e.Boolean(true)},
		{"#f", e.Boolean(false)},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		if lp.First() != c.out {
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
		actual := string(lp.First().(e.String))

		if actual != c.out {
			t.Errorf("Test #%v:Parsed \"%s\", expected %v but got %v", i, c.in, c.out, lp)
		}
	}

}

func TestParseIdententifiers(t *testing.T) {

	cases := []struct {
		in  string
		out e.Identifier
	}{
		{"anIdentifier", e.Identifier("anIdentifier")},
		{"--", e.Identifier("--")},
		{"-----", e.Identifier("-----")},
		{"--1", e.Identifier("--1")},
		{"a.", e.Identifier("a.")},
		{SPECIAL_IDENTIFIERS, e.Identifier(SPECIAL_IDENTIFIERS)},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		if lp.First() != c.out {
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

		if lp.First() != e.Identifier(c) {
			t.Errorf("Parsed \"%s\", expected %v but got %v", c, c, lp)
		}
	}

}

func TestParseElipsis(t *testing.T) {
	cases := []struct {
		in  string
		out e.Expr
	}{
		{"(a ...)", e.Cons(e.Identifier("a"), e.Cons(e.Identifier("..."), e.NIL))},
		{"(a ... b)", e.Cons(e.Identifier("a"), e.Cons(e.Identifier("..."), e.Cons(e.Identifier("b"), e.NIL)))},
		{"(a c ... b)", e.Cons(e.Identifier("a"), e.Cons(e.Identifier("c"), e.Cons(e.Identifier("..."), e.Cons(e.Identifier("b"), e.NIL))))},
	}

	for _, c := range cases {
		res, parsed := Parse(strings.NewReader(c.in))
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c)
		}

		if !c.out.Equiv(lp.First()) {
			t.Errorf("Could not parse elipsis. Expected %s, but got %s", c.out.Repr(), lp.First().Repr())
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
		q, ok := lp.First().(*e.Quote)
		if !ok {
			t.Errorf("Parsed \"'%s\", expected a quoted expression, but got %T", c, lp.First())
		}

		if q.Quoted != e.Identifier(c) {
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
	q, ok := lp.First().(*e.Quote)

	q2, ok2 := q.Quoted.(*e.Quote)

	if !ok || !ok2 {
		t.Errorf("Parsed \"'%s\", expected a quoted expression, but got %T", c, lp.First())
	}

	if q2.Quoted != e.Identifier("c") {
		t.Errorf("Parsed \"%s\", expected quoted %v but got %v", c, c, lp)
	}
}

func TestParseLists(t *testing.T) {

	cases := []struct {
		in  string
		out e.Expr
	}{
		{"()", e.NIL},
		{"(foo)", e.Cons(e.Identifier("foo"), e.NIL)},
		{"(bar foo)", e.Cons(e.Identifier("bar"), e.Cons(e.Identifier("foo"), e.NIL))},
		{"(1 #f)", e.Cons(e.Integer(1), e.Cons(e.Boolean(false), e.NIL))},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		var actualList = lp.First().(e.List)
		var expectedList = c.out.(e.List)

		for actualList != e.NIL || expectedList != e.NIL {
			if actualList.First() != expectedList.First() {
				t.Errorf("List element missmatch, expected: %s was: %s in %s", expectedList.First().Repr(), actualList.First().Repr(), lp.First().Repr())
			}

			actualList, _ = actualList.Tail()
			expectedList, _ = expectedList.Tail()
		}

		if actualList != expectedList && expectedList != e.NIL {
			t.Errorf("Parsed \"%s\", expected %v but got %v", c.in, c.out, lp)
		}
	}

}

func TestParsePairs(t *testing.T) {

	cases := []struct {
		in  string
		out *e.Pair
	}{
		{"(a . b)", e.Cons(e.Identifier("a"), e.Identifier("b"))},

		{"(`bar` . `foo`)", e.Cons(e.String("bar"), e.String("foo"))},
		{"(1 . #f)", e.Cons(e.Integer(1), e.Boolean(false))},
		{"(1 2 . #f)", e.Cons(e.Integer(1), e.Cons(e.Integer(2), e.Boolean(false)))},
		{"(a b c . d)", e.Cons(e.Identifier("a"), e.Cons(e.Identifier("b"), e.Cons(e.Identifier("c"), e.Identifier("d"))))},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Fatalf("Parser failed to parse \"%s\"", c.in)
		}

		actual := lp.First().(*e.Pair)

		if !actual.First().Equiv(c.out.First()) {
			t.Errorf("Got %v as head, expected %v", actual.First().Repr(), c.out.First().Repr())
		}

		if !actual.Second().Equiv(c.out.Second()) {
			t.Errorf("Got %v as tail, expected %v", actual.Second().Repr(), c.out.Second().Repr())
		}

	}
}

func TestParseTwoSymbols(t *testing.T) {

	cases := []struct {
		in  string
		out e.Expr
	}{
		{"'foo 'mmm", e.Cons(e.Quote{e.Identifier("foo")}, e.Cons(e.Quote{e.Identifier("mmm")}, e.NIL))},
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
		out e.Expr
	}{
		{"#!/usr/bin/ghoul", e.NIL},
		{"#!/usr/bin/ghoul\n", e.NIL},
		{"\n#!/usr/bin/ghoul\n", e.NIL},
		{";; foo bar \n#!/usr/bin/ghoul\n", e.NIL},
		{"#!/usr/bin/ghoul\n77", e.Cons(e.Integer(77), e.NIL)},
		{"\n#!/usr/bin/ghoul\n`foo`", e.Cons(e.String("foo"), e.NIL)},
		{";; foo bar \n#!/usr/bin/ghoul\n(bar)", e.Cons(e.Cons(e.Identifier("bar"), e.NIL), e.NIL)},
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

func TestParserRegistersPositionOfPairs(t *testing.T) {

	cases := []struct {
		in                    string
		expectedPosOfLastExpr Position
	}{
		{"\n mmm", Position{2, 2}},
		{"123 (mmm foo)", Position{1, 5}},
	}

	for _, c := range cases {

		r := strings.NewReader(c.in)
		lex := NewLexer(r)
		res := yyParse(lex)

		if res != 0 {
			t.Fatalf("Parser failed to parse \"%s\"", c.in)
		}

		expr := lex.lpair
		var last e.List
		for expr != e.NIL {
			last = expr
			expr, _ = last.Tail()
		}

		lastPos, ok := lex.PairSrcPositions[*last.(*e.Pair)]
		if !ok || lastPos != c.expectedPosOfLastExpr {
			t.Errorf("Expected position of last id in %q tobe %q was %q", c.in, c.expectedPosOfLastExpr, lastPos)
		}
	}

}

func TestInnerPairsHavePositionsRegistered(t *testing.T) {
	cases := []struct {
		in                    string
		expectedPosOfLastExpr Position
	}{
		{"\n\n(mmm foo)", Position{3, 6}},
	}

	for _, c := range cases {

		r := strings.NewReader(c.in)
		lex := NewLexer(r)
		res := yyParse(lex)

		if res != 0 {
			t.Fatalf("Parser failed to parse \"%s\"", c.in)
		}

		expr := lex.lpair.First().(e.List)
		var last e.List
		for expr != e.NIL {
			last = expr
			expr, _ = last.Tail()
		}

		lastPos, ok := lex.PairSrcPositions[*last.(*e.Pair)]
		if !ok || lastPos != c.expectedPosOfLastExpr {
			t.Errorf("Expected position of last id in %q tobe %q was %q", c.in, c.expectedPosOfLastExpr, lastPos)
		}
	}
}
