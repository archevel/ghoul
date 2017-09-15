package expressions

import (
	"testing"
)

func TestBooleanRepresentation(t *testing.T) {
	cases := []struct {
		in  bool
		out string
	}{
		{true, "#t"},
		{false, "#f"},
	}

	for _, c := range cases {

		n := Boolean(c.in)

		actual := n.Repr()

		if actual != c.out {
			t.Errorf("Input was %v. Expected %s but got %s", c.in, c.out, actual)
		}
	}
}

func TestIntegerRepresentation(t *testing.T) {
	cases := []struct {
		in  int64
		out string
	}{
		{1, "1"},
		{2, "2"},
		{0, "0"},
		{-1, "-1"},
		{99999999, "99999999"},
		{-99999999, "-99999999"},
	}

	for _, c := range cases {

		n := Integer(c.in)

		actual := n.Repr()

		if actual != c.out {
			t.Errorf("Input was %d. Expected %s but got %s", c.in, c.out, actual)
		}
	}
}

func TestFloatRepresentation(t *testing.T) {
	cases := []struct {
		in  float64
		out string
	}{
		{1.0, "1"},
		{2.0, "2"},
		{0.0, "0"},
		{-1.0, "-1"},
		{99999999.3, "9.99999993e+07"},
		{-99999999.3, "-9.99999993e+07"},
		{-99999999.3e10, "-9.99999993e+17"},
		{999999.39999, "999999.39999"},
		{1000000.0, "1e+06"},
		{1000000.01, "1.00000001e+06"},
	}

	for _, c := range cases {

		n := Float(c.in)

		actual := n.Repr()

		if actual != c.out {
			t.Errorf("Input was %f. Expected %s but got %s", c.in, c.out, actual)
		}
	}
}

func TestStringRepresentation(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"foo", `"foo"`},
		{"'foo", `"'foo"`},
		{"(bas biz)", `"(bas biz)"`},
		{"foo\n\tzip", "\"foo\n\tzip\""},
	}

	for _, c := range cases {

		s := String(c.in)

		actual := s.Repr()

		if actual != c.out {
			t.Errorf("Input was %f. Expected %s but got %s", c.in, c.out, actual)
		}

	}
}

func TestQuoteRepresentation(t *testing.T) {
	cases := []struct {
		in  Expr
		out string
	}{
		{String("foo"), `'"foo"`},
		{String("'foo"), `'"'foo"`},
		{String("(bas biz)"), `'"(bas biz)"`},
		{String("foo\n\tzip"), "'\"foo\n\tzip\""},
		{Integer(0), "'0"},
		{Float(0.01), "'0.01"},
	}

	for _, c := range cases {

		q := Quote{c.in}

		actual := q.Repr()

		if actual != c.out {
			t.Errorf("Input was '%s. Expected %s but got %s", c.in.Repr(), c.out, actual)
		}
	}
}

func TestPairRepresentation(t *testing.T) {
	cases := []struct {
		in  Expr
		out string
	}{
		{Cons(String("foo"), String("foo")), `("foo" . "foo")`},
		{Cons(String("foo"), NIL), `("foo")`},
		{Cons(Quote{String("foo\n\tzip")}, NIL), "('\"foo\n\tzip\")"},
		{Cons(Integer(0), NIL), "(0)"},
		{Cons(Float(0.01), NIL), "(0.01)"},
		{Cons(Float(0.01), Cons(Integer(1), Cons(String("gargamell"), NIL))), `(0.01 1 "gargamell")`},
	}

	for _, c := range cases {

		p := c.in

		actual := p.Repr()

		if actual != c.out {
			t.Errorf("Input was %v. Expected %s but got %s", c.in, c.out, actual)
		}
	}
}

type testStructA struct {
	value string
}

var testA = testStructA{"a"}

type testStructB struct {
	num int64
}

var testB = testStructB{23}

type testStructC struct {
	something interface{}
}

var testC = testStructC{nil}

func TestForeignRepresentation(t *testing.T) {
	cases := []struct {
		in  Expr
		out string
	}{
		{Wrapp(testA), "#<foreign:expressions.testStructA{value:\"a\"}>"},
		{Wrapp(testStructA{"b"}), "#<foreign:expressions.testStructA{value:\"b\"}>"},
		{Wrapp(testB), "#<foreign:expressions.testStructB{num:23}>"},
		{Wrapp(testC), "#<foreign:expressions.testStructC{something:interface {}(nil)}>"},
		{Wrapp(&testA), "#<foreign:&expressions.testStructA{value:\"a\"}>"},
		{Wrapp(&testStructA{"b"}), "#<foreign:&expressions.testStructA{value:\"b\"}>"},
		{Wrapp(&testB), "#<foreign:&expressions.testStructB{num:23}>"},
		{Wrapp(&testC), "#<foreign:&expressions.testStructC{something:interface {}(nil)}>"},
		{Wrapp(nil), "#<foreign:<nil>>"},
		{Wrapp([]testStructA{testStructA{"a"}}), "#<foreign:[]expressions.testStructA{expressions.testStructA{value:\"a\"}}>"},
	}

	for _, c := range cases {

		p := c.in

		actual := p.Repr()

		if actual != c.out {
			t.Errorf("Input was %v. Expected:\n\t'%s'\n\nbut got:\n\t'%s'", c.in, c.out, actual)
		}
	}
}

func TestForeignValue(t *testing.T) {

	cases := []struct {
		in                Expr
		isPointer         bool
		structName        string
		structIntValue    int64
		structStringValue string
	}{

		{Wrapp(testA), false, "testStructA", 0, "a"},
		{Wrapp(testStructA{"b"}), false, "testStructA", 0, "b"},
		{Wrapp(testB), false, "testStructB", 23, ""},
		{Wrapp(testC), false, "testStructC", 0, ""},
		{Wrapp(&testA), true, "testStructA", 0, "a"},
		{Wrapp(&testStructA{"b"}), true, "testStructA", 0, "b"},
		{Wrapp(&testB), true, "testStructB", 23, ""},
		{Wrapp(&testC), true, "testStructC", 0, ""},
	}

	for _, c := range cases {

		f := c.in.(*Foreign)

		actual := f.Val()

		if c.structName == "testStructA" && c.isPointer {
			s, ok := actual.(*testStructA)
			if !ok {
				t.Errorf("Failed to convert wrapped value to *testStructA, was: %#v", actual)
			}
			if s.value != c.structStringValue {
				t.Errorf("Got value %#v. Expected value %#v", s.value, c.structStringValue)
			}
		}

		if c.structName == "testStructA" && !c.isPointer {
			s, ok := actual.(testStructA)
			if !ok {
				t.Errorf("Failed to convert wrapped value to *testStructA, was: %#v", actual)
			}
			if s.value != c.structStringValue {
				t.Errorf("Got value %#v. Expected value %#v", s.value, c.structStringValue)
			}
		}

		if c.structName == "testStructB" && c.isPointer {
			s, ok := actual.(*testStructB)
			if !ok {
				t.Errorf("Failed to convert wrapped value to *testStructA, was: %#v", actual)
			}
			if s.num != c.structIntValue {
				t.Errorf("Got value %#v. Expected value %#v", s.num, c.structIntValue)
			}
		}

		if c.structName == "testStructB" && !c.isPointer {
			s, ok := actual.(testStructB)
			if !ok {
				t.Errorf("Failed to convert wrapped value to *testStructA, was: %#v", actual)
			}
			if s.num != c.structIntValue {
				t.Errorf("Got value %#v. Expected value %#v", s.num, c.structIntValue)
			}
		}

		if c.structName == "testStructC" && c.isPointer {
			s, ok := actual.(*testStructC)
			if !ok {
				t.Errorf("Failed to convert wrapped value to *testStructA, was: %#v", actual)
			}
			if s.something != nil {
				t.Errorf("Got value %#v. Expected nil", s.something)
			}
		}

		if c.structName == "testStructC" && !c.isPointer {
			s, ok := actual.(testStructC)
			if !ok {
				t.Errorf("Failed to convert wrapped value to *testStructA, was: %#v", actual)
			}
			if s.something != nil {
				t.Errorf("Got value %#v. Expected nil", s.something)
			}
		}
	}

}

var equivCases = []struct {
	a  Expr
	b  Expr
	eq bool
}{
	{Integer(1), Integer(1), true},
	{Integer(2), Integer(1), false},
	{Integer(1), Float(1.0), true},
	{Float(1.0), Integer(1), true},
	{Float(1.0), Float(2.0), false},
	{Float(2.0), Float(2.0), true},
	{String(""), String(""), true},
	{String(""), String("a"), false},
	{String("a"), String("a"), true},
	{String("a"), String(""), false},
	{Integer(1), String(""), false},
	{String(""), Integer(1), false},
	{Identifier("--"), Identifier("a"), false},
	{Identifier("--"), String("a"), false},
	{Identifier("--"), Identifier("--"), true},
	{Identifier("--"), String("--"), false},
	{Boolean(false), Boolean(false), true},
	{Boolean(true), Boolean(true), true},
	{Boolean(true), Boolean(false), false},
	{Boolean(false), Boolean(true), false},
	{NIL, Boolean(false), false},
	{NIL, Boolean(true), false},
	{NIL, String(""), false},
	{NIL, Cons(NIL, NIL), false},
	{NIL, NIL, true},
	{Cons(Identifier("sum"), NIL), Cons(Identifier("sum"), NIL), true},
	{*Cons(Identifier("gr"), NIL), Cons(Identifier("gr"), NIL), true},
	{Cons(Identifier("sum"), NIL), *Cons(Identifier("sum"), NIL), true},
	{*Cons(Identifier("min"), NIL), *Cons(Identifier("min"), NIL), true},
	{&Quote{Identifier("ludlum")}, &Quote{Identifier("ludlum")}, true},
	{Quote{Identifier("dare")}, &Quote{Identifier("dare")}, true},
	{Quote{Identifier("hudlum")}, Quote{Identifier("hudlum")}, true},
	{Wrapp(testA), Wrapp(testA), true},
	{*Wrapp(testA), Wrapp(testA), true},
	{Wrapp(testA), *Wrapp(testA), true},
	{*Wrapp(testA), *Wrapp(testA), true},
	{Wrapp(testB), Wrapp(testA), false},
	{Wrapp(testB), Wrapp(testStructB{23}), true},
	{Wrapp(testB), Wrapp(testStructB{99}), false},
	{Wrapp(testStructC{testB}), Wrapp(testStructC{testStructB{23}}), true},
	{Wrapp(testStructC{&testB}), Wrapp(testStructC{&testB}), true},
	{Wrapp(testStructC{&testB}), Wrapp(testStructC{&testStructB{23}}), false},
	{Wrapp(&testStructC{testB}), Wrapp(&testStructC{testB}), false},
	{Wrapp(testStructC{testB}), Wrapp(testStructC{&testStructB{23}}), false},
	{*Wrapp(testStructC{testB}), Wrapp(testStructC{testStructB{23}}), true},
	{*Wrapp(testStructC{&testB}), Wrapp(testStructC{&testB}), true},
	{*Wrapp(testStructC{&testB}), Wrapp(testStructC{&testStructB{23}}), false},
	{*Wrapp(&testStructC{testB}), Wrapp(&testStructC{testB}), false},
	{*Wrapp(testStructC{testB}), Wrapp(testStructC{&testStructB{23}}), false},
	{Wrapp(testStructC{testB}), *Wrapp(testStructC{testStructB{23}}), true},
	{Wrapp(testStructC{&testB}), *Wrapp(testStructC{&testB}), true},
	{Wrapp(testStructC{&testB}), *Wrapp(testStructC{&testStructB{23}}), false},
	{Wrapp(&testStructC{testB}), *Wrapp(&testStructC{testB}), false},
	{Wrapp(testStructC{testB}), *Wrapp(testStructC{&testStructB{23}}), false},
	{*Wrapp(testStructC{testB}), *Wrapp(testStructC{testStructB{23}}), true},
	{*Wrapp(testStructC{&testB}), *Wrapp(testStructC{&testB}), true},
	{*Wrapp(testStructC{&testB}), *Wrapp(testStructC{&testStructB{23}}), false},
	{*Wrapp(&testStructC{testB}), *Wrapp(&testStructC{testB}), false},
	{*Wrapp(testStructC{testB}), *Wrapp(testStructC{&testStructB{23}}), false},
}

func TestSimpleExpressionEquiv(t *testing.T) {

	for _, c := range equivCases {
		actual := c.a.Equiv(c.b)
		if actual != c.eq {
			t.Errorf("('%v') Equiv ('%v') was %v, expected %v", c.a, c.b, actual, c.eq)
		}
	}
}

func TestQuotedExpressionEquiv(t *testing.T) {
	for _, c := range equivCases {
		quotedA := Quote{c.a}
		quotedB := Quote{c.b}
		actual := quotedA.Equiv(quotedB)
		if actual != c.eq {
			t.Errorf("%v Equiv %v was %v, expected %v", quotedA.Repr(), quotedB.Repr(), actual, c.eq)
		}
	}
}

func TestPairExpressionEquiv(t *testing.T) {
	for _, c := range equivCases {
		var pairA List
		var pairB List
		var actual bool
		pairA = Cons(c.a, c.b)
		pairB = Cons(c.a, c.b)
		actual = pairA.Equiv(pairB)
		if !actual {
			t.Errorf("%v Equiv %v, expected to be equal but was not", pairA.Repr(), pairB.Repr())
		}

		pairA = Cons(c.b, c.a)
		pairB = Cons(c.a, c.b)
		actual = pairA.Equiv(pairB)
		if actual != c.eq {
			t.Errorf("%v Equiv %v was %v, expected %v", pairA.Repr(), pairB.Repr(), actual, c.eq)
		}

		pairA = Cons(c.b, Cons(c.a, Cons(c.b, NIL)))
		pairB = Cons(c.a, Cons(c.b, Cons(c.a, NIL)))
		actual = pairA.Equiv(pairB)
		if actual != c.eq {
			t.Errorf("%v Equiv %v was %v, expected %v", pairA.Repr(), pairB.Repr(), actual, c.eq)
		}

		pairA = Cons(c.a, Cons(c.b, Cons(Integer(1), NIL)))
		pairB = Cons(c.a, Cons(c.b, NIL))
		actual = pairA.Equiv(pairB)
		if actual {
			t.Errorf("%v Equiv %v was equal, expected they would not be", pairA.Repr(), pairB.Repr())
		}

		pairA = Cons(c.a, Cons(c.b, NIL))
		pairB = Cons(c.a, Cons(c.b, Cons(Integer(1), NIL)))
		actual = pairA.Equiv(pairB)
		if actual {
			t.Errorf("%v Equiv %v was equal, expected they would not be", pairA.Repr(), pairB.Repr())
		}
	}
}
