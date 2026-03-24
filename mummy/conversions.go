package mummy

import (
	"fmt"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
)

func bytesConv(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
	s, ok := args.First().(e.String)
	if !ok {
		return nil, fmt.Errorf("bytes: expected string, got %s", e.TypeName(args.First()))
	}
	return Entomb([]byte(string(s)), "[]byte"), nil
}

func stringFromBytes(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
	m, ok := args.First().(*Mummy)
	if !ok {
		return nil, fmt.Errorf("string-from-bytes: expected mummy wrapping []byte, got %s", e.TypeName(args.First()))
	}
	bs, ok := m.Unwrap().([]byte)
	if !ok {
		return nil, fmt.Errorf("string-from-bytes: mummy contains %T, expected []byte", m.Unwrap())
	}
	return e.String(bs), nil
}

func intSlice(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
	var result []int
	for args != e.NIL {
		val, ok := args.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("int-slice: expected integer, got %s", e.TypeName(args.First()))
		}
		result = append(result, int(val))
		args, _ = args.Tail()
	}
	if result == nil {
		result = []int{}
	}
	return Entomb(result, "[]int"), nil
}

func floatSlice(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
	var result []float64
	for args != e.NIL {
		val, ok := args.First().(e.Float)
		if !ok {
			return nil, fmt.Errorf("float-slice: expected float, got %s", e.TypeName(args.First()))
		}
		result = append(result, float64(val))
		args, _ = args.Tail()
	}
	if result == nil {
		result = []float64{}
	}
	return Entomb(result, "[]float64"), nil
}

func goNil(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
	return Entomb(nil, "nil"), nil
}

func RegisterConversions(env *ev.Environment) {
	env.Register("bytes", bytesConv)
	env.Register("string-from-bytes", stringFromBytes)
	env.Register("int-slice", intSlice)
	env.Register("float-slice", floatSlice)
	env.Register("go-nil", goNil)
}
