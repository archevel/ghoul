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

		if lp.Head() != c.out {
			t.Errorf("Parsed \"%s\", expected %v but got %T%v", c.in, c.out, lp.Head(), lp.Head())
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

		if lp.Head() != c.out {
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

		if lp.Head() != c.out {
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
		actual := string(lp.Head().(e.String))

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

		if lp.Head() != c.out {
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

		if lp.Head() != e.Identifier(c) {
			t.Errorf("Parsed \"%s\", expected %v but got %v", c, c, lp)
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
		q, ok := lp.Head().(*e.Quote)
		if !ok {
			t.Errorf("Parsed \"'%s\", expected a quoted expression, but got %T", c, lp.Head())
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
	q, ok := lp.Head().(*e.Quote)

	q2, ok2 := q.Quoted.(*e.Quote)

	if !ok || !ok2 {
		t.Errorf("Parsed \"'%s\", expected a quoted expression, but got %T", c, lp.Head())
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
		{"(foo)", &e.Pair{e.Identifier("foo"), e.NIL}},
		{"(bar foo)", &e.Pair{e.Identifier("bar"), &e.Pair{e.Identifier("foo"), e.NIL}}},
		{"(1 #f)", &e.Pair{e.Integer(1), &e.Pair{e.Boolean(false), e.NIL}}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Errorf("Parser failed to parse \"%s\"", c.in)
		}

		var actualList = lp.Head().(e.List)
		var expectedList = c.out.(e.List)

		for actualList != e.NIL || expectedList != e.NIL {
			if actualList.Head() != expectedList.Head() {
				t.Errorf("List element missmatch, expected: %s was: %s in %s", expectedList.Head().Repr(), actualList.Head().Repr(), lp.Head().Repr())
			}

			actualList = actualList.Tail().(e.List)
			expectedList = expectedList.Tail().(e.List)
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
		{"(a . b)", &e.Pair{e.Identifier("a"), e.Identifier("b")}},

		{"(`bar` . `foo`)", &e.Pair{e.String("bar"), e.String("foo")}},
		{"(1 . #f)", &e.Pair{e.Integer(1), e.Boolean(false)}},
		{"(1 2 . #f)", &e.Pair{e.Integer(1), &e.Pair{e.Integer(2), e.Boolean(false)}}},
	}

	for _, c := range cases {
		r := strings.NewReader(c.in)
		res, parsed := Parse(r)
		lp := parsed.Expressions
		if res != 0 {
			t.Fatalf("Parser failed to parse \"%s\"", c.in)
		}

		actual := lp.Head().(*e.Pair)

		if !actual.Head().Equiv(c.out.Head()) {
			t.Errorf("Got %v as head, expected %v", actual.Head().Repr(), c.out.Head().Repr())
		}

		if !actual.Tail().Equiv(c.out.Tail()) {
			t.Errorf("Got %v as tail, expected %v", actual.Tail().Repr(), c.out.Tail().Repr())
		}

	}
}

func TestParseTwoSymbols(t *testing.T) {

	cases := []struct {
		in  string
		out e.Expr
	}{
		{"'foo 'mmm", e.Pair{e.Quote{e.Identifier("foo")}, e.Pair{e.Quote{e.Identifier("mmm")}, e.NIL}}},
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
		{"#!/usr/bin/ghoul\n77", e.Pair{e.Integer(77), e.NIL}},
		{"\n#!/usr/bin/ghoul\n`foo`", e.Pair{e.String("foo"), e.NIL}},
		{";; foo bar \n#!/usr/bin/ghoul\n(bar)", e.Pair{e.Pair{e.Identifier("bar"), e.NIL}, e.NIL}},
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
			t.Fatal("Parser failed to parse \"%s\"", c.in)
		}

		expr := lex.lpair
		var last e.List
		for expr != e.NIL {
			last = expr
			expr = last.Tail().(e.List)
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
			t.Fatal("Parser failed to parse \"%s\"", c.in)
		}

		expr := lex.lpair.Head().(e.List)
		var last e.List
		for expr != e.NIL {
			last = expr
			expr = last.Tail().(e.List)
		}

		lastPos, ok := lex.PairSrcPositions[*last.(*e.Pair)]
		if !ok || lastPos != c.expectedPosOfLastExpr {
			t.Errorf("Expected position of last id in %q tobe %q was %q", c.in, c.expectedPosOfLastExpr, lastPos)
		}
	}
}
