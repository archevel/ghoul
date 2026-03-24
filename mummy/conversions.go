package mummy

import (
	"fmt"

	e "github.com/archevel/ghoul/expressions"
)

// ConversionFunc is the signature for conversion functions.
// Uses interface{} for the evaluator to avoid importing the evaluator package.
type ConversionFunc func(args e.List, evaluator interface{}) (e.Expr, error)

func BytesConv(args e.List, evaluator interface{}) (e.Expr, error) {
	s, ok := args.First().(e.String)
	if !ok {
		return nil, fmt.Errorf("bytes: expected string, got %s", e.TypeName(args.First()))
	}
	return Entomb([]byte(string(s)), "[]byte"), nil
}

func StringFromBytes(args e.List, evaluator interface{}) (e.Expr, error) {
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

func IntSlice(args e.List, evaluator interface{}) (e.Expr, error) {
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

func FloatSlice(args e.List, evaluator interface{}) (e.Expr, error) {
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

func GoNil(args e.List, evaluator interface{}) (e.Expr, error) {
	return Entomb(nil, "nil"), nil
}
