package stdlib

import (
	ev "github.com/archevel/ghoul/evaluator"
	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/macromancy"
	"github.com/archevel/ghoul/mummy"
)

// stripMarks recursively removes hygiene marks from expressions,
// converting ScopedIdentifiers to plain Identifiers and unwrapping
// SyntaxObjects. Used by syntax-match? to compare by name.
func stripMarks(expr e.Expr) e.Expr {
	if si, ok := expr.(e.ScopedIdentifier); ok {
		return si.Name
	}
	if so, ok := expr.(macromancy.SyntaxObject); ok {
		return stripMarks(so.Datum)
	}
	if expr == e.NIL {
		return e.NIL
	}
	if pair, ok := expr.(*e.Pair); ok {
		return e.Cons(stripMarks(pair.H), stripMarks(pair.T))
	}
	return expr
}

func registerSyntax(env *ev.Environment) {
	env.Register("syntax->datum", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if so, ok := arg.(macromancy.SyntaxObject); ok {
			return so.Datum, nil
		}
		if si, ok := arg.(e.ScopedIdentifier); ok {
			return si.Name, nil
		}
		return arg, nil
	})

	env.Register("datum->syntax", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		ctxArg := args.First()
		t, _ := args.Tail()
		datum := t.First()
		marks := macromancy.NewMarkSet()
		if so, ok := ctxArg.(macromancy.SyntaxObject); ok {
			marks = so.Marks
		}
		return macromancy.WrapExpr(datum, marks), nil
	})

	env.Register("identifier?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		arg := args.First()
		if so, ok := arg.(macromancy.SyntaxObject); ok {
			_, isId := so.Datum.(e.Identifier)
			return e.Boolean(isId), nil
		}
		if _, ok := arg.(e.ScopedIdentifier); ok {
			return e.Boolean(true), nil
		}
		_, isId := arg.(e.Identifier)
		return e.Boolean(isId), nil
	})

	env.Register("syntax-match?", func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
		// (syntax-match? expr pattern literals)
		// Returns an association list of bindings or #f.
		// Both expr and pattern are stripped of hygiene marks before matching
		// so that identifier comparison works by name.
		expr := stripMarks(args.First())
		t1, _ := args.Tail()
		pattern := stripMarks(t1.First())
		t2, _ := t1.Tail()
		litList := t2.First()

		// Build literals map from the literals list
		literals := map[e.Identifier]bool{}
		if ll, ok := litList.(e.List); ok {
			for ll != e.NIL {
				if id, ok := ll.First().(e.Identifier); ok {
					literals[id] = true
				}
				next, ok := ll.Tail()
				if !ok {
					break
				}
				ll = next
			}
		}

		patternVars := macromancy.ExtractPatternVars(pattern, literals)
		ellipsisVars := macromancy.ExtractEllipsisVars(pattern, literals)
		macro := macromancy.Macro{
			Pattern:      pattern,
			PatternVars:  patternVars,
			EllipsisVars: ellipsisVars,
			Literals:     literals,
		}

		ok, alist := macro.MatchAndBind(expr)
		if !ok {
			return e.Boolean(false), nil
		}
		return alist, nil
	})

	// Mummy conversion functions
	wrapConv := func(fn func(e.List, interface{}) (e.Expr, error)) func(e.List, *ev.Evaluator) (e.Expr, error) {
		return func(args e.List, evaluator *ev.Evaluator) (e.Expr, error) {
			return fn(args, evaluator)
		}
	}
	env.Register("bytes", wrapConv(mummy.BytesConv))
	env.Register("string-from-bytes", wrapConv(mummy.StringFromBytes))
	env.Register("int-slice", wrapConv(mummy.IntSlice))
	env.Register("float-slice", wrapConv(mummy.FloatSlice))
	env.Register("go-nil", wrapConv(mummy.GoNil))
}
