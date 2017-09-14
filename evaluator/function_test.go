package evaluator

import (
	"testing"

	e "github.com/archevel/ghoul/expressions"
)

func TestFunctionRepr(t *testing.T) {
	argFun := func(expr e.List) (e.Expr, error) { return expr, nil }

	cases := []struct {
		in  Function
		out string
	}{
		{Function{&argFun}, "#<procedure>"},
	}

	for _, c := range cases {
		actual := c.in.Repr()
		if actual != c.out {
			t.Errorf("Input was %f. Expected %s but got %s", c.in, c.out, actual)
		}
	}
}

func TestFunctionEquiv(t *testing.T) {
	funA := func(args e.List) (e.Expr, error) { return args, nil }
	funB := func(args e.List) (e.Expr, error) { return args, nil }
	funcA := Function{&funA}
	funcB := Function{&funB}

	cases := []struct {
		first  e.Expr
		second e.Expr
		eq     bool
	}{
		{funcA, funcB, false},
		{funcA, funcA, true},
		{funcA, &funcA, true},
		{&funcA, funcA, true},
		{&funcA, &funcA, true},
		{&funcA, e.Integer(1), false},
	}
	for i, c := range cases {
		actual := c.first.Equiv(c.second)
		if actual != c.eq {
			t.Errorf("Case %d: %v Equiv %v was %v, expected %v", i, c.first.Repr(), c.second.Repr(), actual, c.eq)
		}
	}
}
