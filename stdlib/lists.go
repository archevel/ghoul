package stdlib

import (
	"fmt"

	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
)

func registerLists(env *ev.Environment) {
	env.Register("car", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if list, ok := arg.(e.List); ok && list != e.NIL {
			return list.First(), nil
		}
		return nil, fmt.Errorf("car: expected a non-empty list, got %s", e.TypeName(arg))
	})

	env.Register("cdr", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if list, ok := arg.(e.List); ok && list != e.NIL {
			return list.Second(), nil
		}
		return nil, fmt.Errorf("cdr: expected a non-empty list, got %s", e.TypeName(arg))
	})

	env.Register("cons", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fst := args.First()
		t, _ := args.Tail()
		snd := t.First()
		return e.Cons(fst, snd), nil
	})

	env.Register("list", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		return args, nil
	})

	env.Register("length", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		lst, ok := args.First().(e.List)
		if !ok {
			return nil, fmt.Errorf("length: expected list, got %s", e.TypeName(args.First()))
		}
		count := 0
		for lst != e.NIL {
			count++
			next, ok := lst.Tail()
			if !ok {
				return nil, fmt.Errorf("length: improper list")
			}
			lst = next
		}
		return e.Integer(int64(count)), nil
	})

	env.Register("append", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		lst1, ok := args.First().(e.List)
		if !ok {
			return nil, fmt.Errorf("append: expected list as first argument, got %s", e.TypeName(args.First()))
		}
		t, _ := args.Tail()
		lst2 := t.First()

		if lst1 == e.NIL {
			return lst2, nil
		}

		// Collect elements of lst1, then build from the end
		var elems []e.Expr
		for lst1 != e.NIL {
			elems = append(elems, lst1.First())
			next, ok := lst1.Tail()
			if !ok {
				break
			}
			lst1 = next
		}
		result := lst2
		for i := len(elems) - 1; i >= 0; i-- {
			result = e.Cons(elems[i], result)
		}
		return result, nil
	})

	env.Register("reverse", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		lst, ok := args.First().(e.List)
		if !ok {
			return nil, fmt.Errorf("reverse: expected list, got %s", e.TypeName(args.First()))
		}
		var result e.Expr = e.NIL
		for lst != e.NIL {
			result = e.Cons(lst.First(), result)
			next, ok := lst.Tail()
			if !ok {
				break
			}
			lst = next
		}
		return result, nil
	})

	env.Register("map", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fnExpr := args.First()
		t, _ := args.Tail()
		lst, ok := t.First().(e.List)
		if !ok {
			return nil, fmt.Errorf("map: expected list as second argument, got %s", e.TypeName(t.First()))
		}

		var results []e.Expr
		for lst != e.NIL {
			callExpr := e.Cons(fnExpr, e.Cons(lst.First(), e.NIL))
			result, err := evaluator.EvalSubExpression(callExpr)
			if err != nil {
				return nil, fmt.Errorf("map: %w", err)
			}
			results = append(results, result)
			next, ok := lst.Tail()
			if !ok {
				break
			}
			lst = next
		}

		var resultList e.Expr = e.NIL
		for i := len(results) - 1; i >= 0; i-- {
			resultList = e.Cons(results[i], resultList)
		}
		return resultList, nil
	})

	env.Register("filter", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fnExpr := args.First()
		t, _ := args.Tail()
		lst, ok := t.First().(e.List)
		if !ok {
			return nil, fmt.Errorf("filter: expected list as second argument, got %s", e.TypeName(t.First()))
		}

		var results []e.Expr
		for lst != e.NIL {
			callExpr := e.Cons(fnExpr, e.Cons(lst.First(), e.NIL))
			result, err := evaluator.EvalSubExpression(callExpr)
			if err != nil {
				return nil, fmt.Errorf("filter: %w", err)
			}
			if b, ok := result.(e.Boolean); ok && bool(b) {
				results = append(results, lst.First())
			}
			next, ok := lst.Tail()
			if !ok {
				break
			}
			lst = next
		}

		var resultList e.Expr = e.NIL
		for i := len(results) - 1; i >= 0; i-- {
			resultList = e.Cons(results[i], resultList)
		}
		return resultList, nil
	})

	env.Register("foldl", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		fnExpr := args.First()
		t, _ := args.Tail()
		acc := t.First()
		t2, _ := t.Tail()
		lst, ok := t2.First().(e.List)
		if !ok {
			return nil, fmt.Errorf("foldl: expected list as third argument, got %s", e.TypeName(t2.First()))
		}

		for lst != e.NIL {
			callExpr := e.Cons(fnExpr, e.Cons(acc, e.Cons(lst.First(), e.NIL)))
			result, err := evaluator.EvalSubExpression(callExpr)
			if err != nil {
				return nil, fmt.Errorf("foldl: %w", err)
			}
			acc = result
			next, ok := lst.Tail()
			if !ok {
				break
			}
			lst = next
		}
		return acc, nil
	})

	env.Register("assoc", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		key := args.First()
		t, _ := args.Tail()
		lst, ok := t.First().(e.List)
		if !ok {
			return e.Boolean(false), nil
		}
		for lst != e.NIL {
			pair, ok := lst.First().(*e.Pair)
			if ok && pair.H.Equiv(key) {
				return pair, nil
			}
			next, ok := lst.Tail()
			if !ok {
				break
			}
			lst = next
		}
		return e.Boolean(false), nil
	})

	env.Register("null?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		return e.Boolean(args.First() == e.NIL), nil
	})

	env.Register("pair?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		_, ok := args.First().(*e.Pair)
		return e.Boolean(ok), nil
	})
}
