package stdlib

import (
	"fmt"
	"strconv"
	"strings"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
)

func registerStrings(env *ev.Environment) {
	env.Register("string-append", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		var b strings.Builder
		for args != e.NIL {
			s, ok := args.First().(e.String)
			if !ok {
				return nil, fmt.Errorf("string-append: expected string, got %s", e.TypeName(args.First()))
			}
			b.WriteString(string(s))
			args, _ = args.Tail()
		}
		return e.String(b.String()), nil
	})

	env.Register("string-length", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		s, ok := args.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("string-length: expected string, got %s", e.TypeName(args.First()))
		}
		return e.Integer(len([]rune(string(s)))), nil
	})

	env.Register("substring", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		s, ok := args.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("substring: expected string as first argument, got %s", e.TypeName(args.First()))
		}
		runes := []rune(string(s))
		t, _ := args.Tail()
		start, ok := t.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("substring: expected integer as second argument, got %s", e.TypeName(t.First()))
		}
		t2, _ := t.Tail()
		end, ok := t2.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("substring: expected integer as third argument, got %s", e.TypeName(t2.First()))
		}
		if int(start) < 0 || int(end) > len(runes) || int(start) > int(end) {
			return nil, fmt.Errorf("substring: index out of bounds (start=%d, end=%d, length=%d)", start, end, len(runes))
		}
		return e.String(string(runes[start:end])), nil
	})

	env.Register("string-ref", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		s, ok := args.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("string-ref: expected string, got %s", e.TypeName(args.First()))
		}
		runes := []rune(string(s))
		t, _ := args.Tail()
		idx, ok := t.First().(e.Integer)
		if !ok {
			return nil, fmt.Errorf("string-ref: expected integer index, got %s", e.TypeName(t.First()))
		}
		if int(idx) < 0 || int(idx) >= len(runes) {
			return nil, fmt.Errorf("string-ref: index %d out of bounds (length %d)", idx, len(runes))
		}
		return e.String(string(runes[idx])), nil
	})

	env.Register("string-contains?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		s, ok := args.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("string-contains?: expected string as first argument, got %s", e.TypeName(args.First()))
		}
		t, _ := args.Tail()
		substr, ok := t.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("string-contains?: expected string as second argument, got %s", e.TypeName(t.First()))
		}
		return e.Boolean(strings.Contains(string(s), string(substr))), nil
	})

	env.Register("string-split", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		s, ok := args.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("string-split: expected string as first argument, got %s", e.TypeName(args.First()))
		}
		t, _ := args.Tail()
		sep, ok := t.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("string-split: expected string as second argument, got %s", e.TypeName(t.First()))
		}
		parts := strings.Split(string(s), string(sep))
		var result e.Expr = e.NIL
		for i := len(parts) - 1; i >= 0; i-- {
			result = e.Cons(e.String(parts[i]), result)
		}
		return result, nil
	})

	env.Register("string-upcase", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		s, ok := args.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("string-upcase: expected string, got %s", e.TypeName(args.First()))
		}
		return e.String(strings.ToUpper(string(s))), nil
	})

	env.Register("string-downcase", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		s, ok := args.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("string-downcase: expected string, got %s", e.TypeName(args.First()))
		}
		return e.String(strings.ToLower(string(s))), nil
	})

	env.Register("string->number", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		s, ok := args.First().(e.String)
		if !ok {
			return nil, fmt.Errorf("string->number: expected string, got %s", e.TypeName(args.First()))
		}
		str := string(s)
		if i, err := strconv.ParseInt(str, 10, 64); err == nil {
			return e.Integer(i), nil
		}
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return e.Float(f), nil
		}
		return nil, fmt.Errorf("string->number: cannot parse '%s' as a number", str)
	})

	env.Register("number->string", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		switch v := args.First().(type) {
		case e.Integer:
			return e.String(strconv.FormatInt(int64(v), 10)), nil
		case e.Float:
			return e.String(strconv.FormatFloat(float64(v), 'g', -1, 64)), nil
		default:
			return nil, fmt.Errorf("number->string: expected number, got %s", e.TypeName(args.First()))
		}
	})
}
