package tome

import (
	"fmt"

	ev "github.com/archevel/ghoul/consume"
	e "github.com/archevel/ghoul/bones"
)

// extractNums extracts two numeric arguments, promoting to float if mixed.
// Returns (intA, intB, true) for integer pair, or (floatA, floatB, false) for float/mixed.
func extractNums(name string, args e.List) (e.Integer, e.Integer, e.Float, e.Float, bool, error) {
	fstExpr := args.First()
	t, _ := args.Tail()
	sndExpr := t.First()

	fstInt, fstIsInt := fstExpr.(e.Integer)
	fstFloat, fstIsFloat := fstExpr.(e.Float)
	sndInt, sndIsInt := sndExpr.(e.Integer)
	sndFloat, sndIsFloat := sndExpr.(e.Float)

	if fstIsInt && sndIsInt {
		return fstInt, sndInt, 0, 0, true, nil
	}
	if fstIsFloat && sndIsFloat {
		return 0, 0, fstFloat, sndFloat, false, nil
	}
	if fstIsInt && sndIsFloat {
		return 0, 0, e.Float(fstInt), sndFloat, false, nil
	}
	if fstIsFloat && sndIsInt {
		return 0, 0, fstFloat, e.Float(sndInt), false, nil
	}

	if !fstIsInt && !fstIsFloat {
		return 0, 0, 0, 0, false, fmt.Errorf("%s: expected number as first argument, got %s", name, e.TypeName(fstExpr))
	}
	return 0, 0, 0, 0, false, fmt.Errorf("%s: expected number as second argument, got %s", name, e.TypeName(sndExpr))
}

func numBinOp(name string, intOp func(a, b e.Integer) (e.Expr, error), floatOp func(a, b e.Float) (e.Expr, error)) func(e.List, *ev.Evaluator) (e.Expr, error) {
	return func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		ia, ib, fa, fb, isInt, err := extractNums(name, args)
		if err != nil {
			return nil, err
		}
		if isInt {
			return intOp(ia, ib)
		}
		return floatOp(fa, fb)
	}
}

func registerArithmetic(env *ev.Environment) {
	env.Register("+", numBinOp("+",
		func(a, b e.Integer) (e.Expr, error) { return e.Integer(a + b), nil },
		func(a, b e.Float) (e.Expr, error) { return e.Float(a + b), nil },
	))

	env.Register("-", numBinOp("-",
		func(a, b e.Integer) (e.Expr, error) { return e.Integer(a - b), nil },
		func(a, b e.Float) (e.Expr, error) { return e.Float(a - b), nil },
	))

	env.Register("*", numBinOp("*",
		func(a, b e.Integer) (e.Expr, error) { return e.Integer(a * b), nil },
		func(a, b e.Float) (e.Expr, error) { return e.Float(a * b), nil },
	))

	env.Register("/", numBinOp("/",
		func(a, b e.Integer) (e.Expr, error) {
			if b == 0 {
				return nil, fmt.Errorf("/: division by zero")
			}
			return e.Integer(a / b), nil
		},
		func(a, b e.Float) (e.Expr, error) {
			if b == 0 {
				return nil, fmt.Errorf("/: division by zero")
			}
			return e.Float(a / b), nil
		},
	))

	env.Register("mod", numBinOp("mod",
		func(a, b e.Integer) (e.Expr, error) {
			if b == 0 {
				return nil, fmt.Errorf("mod: division by zero")
			}
			return e.Integer(a % b), nil
		},
		func(a, b e.Float) (e.Expr, error) {
			return nil, fmt.Errorf("mod: not supported for floats")
		},
	))
}
