package consume

import (
	"context"
	"testing"

	"github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/engraving"
)

func newTestVM() *VM {
	env := NewEnvironment()
	ev := New(engraving.StandardLogger, env)
	return newVM(ev)
}

func compileAndRun(t *testing.T, nodes []*bones.Node, env *environment) *bones.Node {
	t.Helper()
	code, err := compileTopLevel(nodes)
	if err != nil {
		t.Fatal(err)
	}
	ev := New(engraving.StandardLogger, env)
	vm := newVM(ev)
	result, err := vm.run(context.Background(), code)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// --- Literal evaluation ---

func TestVMInteger(t *testing.T) {
	result := compileAndRun(t, []*bones.Node{bones.IntNode(42)}, NewEnvironment())
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestVMFloat(t *testing.T) {
	result := compileAndRun(t, []*bones.Node{bones.FloatNode(3.14)}, NewEnvironment())
	if result.FloatVal != 3.14 {
		t.Errorf("expected 3.14, got %s", result.Repr())
	}
}

func TestVMString(t *testing.T) {
	result := compileAndRun(t, []*bones.Node{bones.StrNode("hello")}, NewEnvironment())
	if result.StrVal != "hello" {
		t.Errorf("expected hello, got %s", result.Repr())
	}
}

func TestVMBoolTrue(t *testing.T) {
	result := compileAndRun(t, []*bones.Node{bones.BoolNode(true)}, NewEnvironment())
	if !result.BoolVal {
		t.Error("expected true")
	}
}

func TestVMBoolFalse(t *testing.T) {
	result := compileAndRun(t, []*bones.Node{bones.BoolNode(false)}, NewEnvironment())
	if result.BoolVal {
		t.Error("expected false")
	}
}

func TestVMNil(t *testing.T) {
	result := compileAndRun(t, []*bones.Node{bones.Nil}, NewEnvironment())
	if !result.IsNil() {
		t.Errorf("expected Nil, got %s", result.Repr())
	}
}

func TestVMEmpty(t *testing.T) {
	result := compileAndRun(t, nil, NewEnvironment())
	if !result.IsNil() {
		t.Errorf("expected Nil for empty, got %s", result.Repr())
	}
}

func TestVMMultipleExprs(t *testing.T) {
	// Returns last value
	result := compileAndRun(t, []*bones.Node{bones.IntNode(1), bones.IntNode(2)}, NewEnvironment())
	if result.IntVal != 2 {
		t.Errorf("expected 2, got %s", result.Repr())
	}
}

func TestVMIdentifierLookup(t *testing.T) {
	env := NewEnvironment()
	bindNode(bones.IdentNode("x"), bones.IntNode(42), env)
	result := compileAndRun(t, []*bones.Node{bones.IdentNode("x")}, env)
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestVMIdentifierUndefined(t *testing.T) {
	env := NewEnvironment()
	code, _ := compileTopLevel([]*bones.Node{bones.IdentNode("x")})
	ev := New(engraving.StandardLogger, env)
	vm := newVM(ev)
	_, err := vm.run(context.Background(), code)
	if err == nil {
		t.Error("expected error for undefined identifier")
	}
}

func TestVMQuote(t *testing.T) {
	q := &bones.Node{Kind: bones.QuoteNode, Quoted: bones.IntNode(42)}
	result := compileAndRun(t, []*bones.Node{q}, NewEnvironment())
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

// --- Step 4: Define + Set! + Begin ---

func TestVMDefine(t *testing.T) {
	env := NewEnvironment()
	// (define x 42) x
	nodes := []*bones.Node{
		{Kind: bones.DefineNode, Children: []*bones.Node{bones.IdentNode("x"), bones.IntNode(42)}},
		bones.IdentNode("x"),
	}
	result := compileAndRun(t, nodes, env)
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestVMSetBang(t *testing.T) {
	env := NewEnvironment()
	// (define x 1) (set! x 2) x
	nodes := []*bones.Node{
		{Kind: bones.DefineNode, Children: []*bones.Node{bones.IdentNode("x"), bones.IntNode(1)}},
		{Kind: bones.SetNode, Children: []*bones.Node{bones.IdentNode("x"), bones.IntNode(2)}},
		bones.IdentNode("x"),
	}
	result := compileAndRun(t, nodes, env)
	if result.IntVal != 2 {
		t.Errorf("expected 2, got %s", result.Repr())
	}
}

func TestVMBegin(t *testing.T) {
	// (begin 1 2 3) → 3
	nodes := []*bones.Node{
		{Kind: bones.BeginNode, Children: []*bones.Node{bones.IntNode(1), bones.IntNode(2), bones.IntNode(3)}},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if result.IntVal != 3 {
		t.Errorf("expected 3, got %s", result.Repr())
	}
}

func TestVMBeginEmpty(t *testing.T) {
	nodes := []*bones.Node{
		{Kind: bones.BeginNode, Children: nil},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if !result.IsNil() {
		t.Errorf("expected Nil, got %s", result.Repr())
	}
}

// --- Step 5: Cond ---

func TestVMCondTrue(t *testing.T) {
	// (cond (#t 42))
	nodes := []*bones.Node{
		{Kind: bones.CondNode, Clauses: []*bones.CondClause{
			{Test: bones.BoolNode(true), Consequent: []*bones.Node{bones.IntNode(42)}},
		}},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestVMCondFalse(t *testing.T) {
	// (cond (#f 42)) → Nil
	nodes := []*bones.Node{
		{Kind: bones.CondNode, Clauses: []*bones.CondClause{
			{Test: bones.BoolNode(false), Consequent: []*bones.Node{bones.IntNode(42)}},
		}},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if !result.IsNil() {
		t.Errorf("expected Nil, got %s", result.Repr())
	}
}

func TestVMCondElse(t *testing.T) {
	// (cond (#f 1) (else 2))
	nodes := []*bones.Node{
		{Kind: bones.CondNode, Clauses: []*bones.CondClause{
			{Test: bones.BoolNode(false), Consequent: []*bones.Node{bones.IntNode(1)}},
			{IsElse: true, Consequent: []*bones.Node{bones.IntNode(2)}},
		}},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if result.IntVal != 2 {
		t.Errorf("expected 2, got %s", result.Repr())
	}
}

func TestVMCondMultipleClauses(t *testing.T) {
	// (cond (#f 1) (#f 2) (#t 3))
	nodes := []*bones.Node{
		{Kind: bones.CondNode, Clauses: []*bones.CondClause{
			{Test: bones.BoolNode(false), Consequent: []*bones.Node{bones.IntNode(1)}},
			{Test: bones.BoolNode(false), Consequent: []*bones.Node{bones.IntNode(2)}},
			{Test: bones.BoolNode(true), Consequent: []*bones.Node{bones.IntNode(3)}},
		}},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if result.IntVal != 3 {
		t.Errorf("expected 3, got %s", result.Repr())
	}
}

func TestVMCondEmpty(t *testing.T) {
	nodes := []*bones.Node{
		{Kind: bones.CondNode, Clauses: nil},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if !result.IsNil() {
		t.Errorf("expected Nil, got %s", result.Repr())
	}
}

// --- Step 6: Function calls (Go natives) ---

func TestVMCallGoFunction(t *testing.T) {
	env := NewEnvironment()
	env.Register("+", func(args []*bones.Node, ev *Evaluator) (*bones.Node, error) {
		return bones.IntNode(args[0].IntVal + args[1].IntVal), nil
	})
	// (+ 1 2)
	nodes := []*bones.Node{
		{Kind: bones.CallNode, Children: []*bones.Node{
			bones.IdentNode("+"), bones.IntNode(1), bones.IntNode(2),
		}},
	}
	result := compileAndRun(t, nodes, env)
	if result.IntVal != 3 {
		t.Errorf("expected 3, got %s", result.Repr())
	}
}

func TestVMCallNestedGoFunction(t *testing.T) {
	env := NewEnvironment()
	env.Register("+", func(args []*bones.Node, ev *Evaluator) (*bones.Node, error) {
		return bones.IntNode(args[0].IntVal + args[1].IntVal), nil
	})
	// (+ (+ 1 2) 3)
	inner := &bones.Node{Kind: bones.CallNode, Children: []*bones.Node{
		bones.IdentNode("+"), bones.IntNode(1), bones.IntNode(2),
	}}
	nodes := []*bones.Node{
		{Kind: bones.CallNode, Children: []*bones.Node{
			bones.IdentNode("+"), inner, bones.IntNode(3),
		}},
	}
	result := compileAndRun(t, nodes, env)
	if result.IntVal != 6 {
		t.Errorf("expected 6, got %s", result.Repr())
	}
}

func TestVMCallNotProcedure(t *testing.T) {
	env := NewEnvironment()
	bindNode(bones.IdentNode("x"), bones.IntNode(42), env)
	code, _ := compileTopLevel([]*bones.Node{
		{Kind: bones.CallNode, Children: []*bones.Node{bones.IdentNode("x"), bones.IntNode(1)}},
	})
	ev := New(engraving.StandardLogger, env)
	vm := newVM(ev)
	_, err := vm.run(context.Background(), code)
	if err == nil {
		t.Error("expected error for calling non-procedure")
	}
}

// --- Step 7: Lambda + closures ---

func TestVMLambdaIdentity(t *testing.T) {
	// ((lambda (x) x) 42)
	lambda := &bones.Node{
		Kind:   bones.LambdaNode,
		Params: &bones.ParamSpec{Fixed: []*bones.Node{bones.IdentNode("x")}},
		Children: []*bones.Node{bones.IdentNode("x")},
	}
	nodes := []*bones.Node{
		{Kind: bones.CallNode, Children: []*bones.Node{lambda, bones.IntNode(42)}},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestVMLambdaWithBody(t *testing.T) {
	env := NewEnvironment()
	env.Register("+", func(args []*bones.Node, ev *Evaluator) (*bones.Node, error) {
		return bones.IntNode(args[0].IntVal + args[1].IntVal), nil
	})
	// (define add (lambda (x y) (+ x y)))
	// (add 3 4)
	lambda := &bones.Node{
		Kind:   bones.LambdaNode,
		Params: &bones.ParamSpec{Fixed: []*bones.Node{bones.IdentNode("x"), bones.IdentNode("y")}},
		Children: []*bones.Node{
			{Kind: bones.CallNode, Children: []*bones.Node{bones.IdentNode("+"), bones.IdentNode("x"), bones.IdentNode("y")}},
		},
	}
	nodes := []*bones.Node{
		{Kind: bones.DefineNode, Children: []*bones.Node{bones.IdentNode("add"), lambda}},
		{Kind: bones.CallNode, Children: []*bones.Node{bones.IdentNode("add"), bones.IntNode(3), bones.IntNode(4)}},
	}
	result := compileAndRun(t, nodes, env)
	if result.IntVal != 7 {
		t.Errorf("expected 7, got %s", result.Repr())
	}
}

func TestVMClosure(t *testing.T) {
	env := NewEnvironment()
	env.Register("+", func(args []*bones.Node, ev *Evaluator) (*bones.Node, error) {
		return bones.IntNode(args[0].IntVal + args[1].IntVal), nil
	})
	// (define make-adder (lambda (n) (lambda (x) (+ x n))))
	// (define add5 (make-adder 5))
	// (add5 10)
	innerLambda := &bones.Node{
		Kind:   bones.LambdaNode,
		Params: &bones.ParamSpec{Fixed: []*bones.Node{bones.IdentNode("x")}},
		Children: []*bones.Node{
			{Kind: bones.CallNode, Children: []*bones.Node{bones.IdentNode("+"), bones.IdentNode("x"), bones.IdentNode("n")}},
		},
	}
	makeAdder := &bones.Node{
		Kind:   bones.LambdaNode,
		Params: &bones.ParamSpec{Fixed: []*bones.Node{bones.IdentNode("n")}},
		Children: []*bones.Node{innerLambda},
	}
	nodes := []*bones.Node{
		{Kind: bones.DefineNode, Children: []*bones.Node{bones.IdentNode("make-adder"), makeAdder}},
		{Kind: bones.DefineNode, Children: []*bones.Node{
			bones.IdentNode("add5"),
			&bones.Node{Kind: bones.CallNode, Children: []*bones.Node{bones.IdentNode("make-adder"), bones.IntNode(5)}},
		}},
		{Kind: bones.CallNode, Children: []*bones.Node{bones.IdentNode("add5"), bones.IntNode(10)}},
	}
	result := compileAndRun(t, nodes, env)
	if result.IntVal != 15 {
		t.Errorf("expected 15, got %s", result.Repr())
	}
}

func TestVMLambdaVariadic(t *testing.T) {
	// (define f (lambda (x . rest) x))
	// (f 42 1 2 3)
	lambda := &bones.Node{
		Kind: bones.LambdaNode,
		Params: &bones.ParamSpec{
			Fixed:    []*bones.Node{bones.IdentNode("x")},
			Variadic: bones.IdentNode("rest"),
		},
		Children: []*bones.Node{bones.IdentNode("x")},
	}
	nodes := []*bones.Node{
		{Kind: bones.DefineNode, Children: []*bones.Node{bones.IdentNode("f"), lambda}},
		{Kind: bones.CallNode, Children: []*bones.Node{
			bones.IdentNode("f"), bones.IntNode(42), bones.IntNode(1), bones.IntNode(2),
		}},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if result.IntVal != 42 {
		t.Errorf("expected 42, got %s", result.Repr())
	}
}

func TestVMLambdaMultiBody(t *testing.T) {
	// ((lambda () 1 2 3)) → 3
	lambda := &bones.Node{
		Kind:     bones.LambdaNode,
		Params:   &bones.ParamSpec{},
		Children: []*bones.Node{bones.IntNode(1), bones.IntNode(2), bones.IntNode(3)},
	}
	nodes := []*bones.Node{
		{Kind: bones.CallNode, Children: []*bones.Node{lambda}},
	}
	result := compileAndRun(t, nodes, NewEnvironment())
	if result.IntVal != 3 {
		t.Errorf("expected 3, got %s", result.Repr())
	}
}

// --- Step 8: Tail call optimization ---

func TestVMTailCallOptimization(t *testing.T) {
	env := NewEnvironment()
	env.Register("-", func(args []*bones.Node, ev *Evaluator) (*bones.Node, error) {
		return bones.IntNode(args[0].IntVal - args[1].IntVal), nil
	})
	env.Register("eq?", func(args []*bones.Node, ev *Evaluator) (*bones.Node, error) {
		return bones.BoolNode(args[0].Equiv(args[1])), nil
	})
	// (define countdown (lambda (n)
	//   (cond ((eq? n 0) 0) (else (countdown (- n 1))))))
	// (countdown 100000)
	lambda := &bones.Node{
		Kind:   bones.LambdaNode,
		Params: &bones.ParamSpec{Fixed: []*bones.Node{bones.IdentNode("n")}},
		Children: []*bones.Node{
			{Kind: bones.CondNode, Clauses: []*bones.CondClause{
				{
					Test: &bones.Node{Kind: bones.CallNode, Children: []*bones.Node{
						bones.IdentNode("eq?"), bones.IdentNode("n"), bones.IntNode(0),
					}},
					Consequent: []*bones.Node{bones.IntNode(0)},
				},
				{
					IsElse: true,
					Consequent: []*bones.Node{
						{Kind: bones.CallNode, Children: []*bones.Node{
							bones.IdentNode("countdown"),
							{Kind: bones.CallNode, Children: []*bones.Node{
								bones.IdentNode("-"), bones.IdentNode("n"), bones.IntNode(1),
							}},
						}},
					},
				},
			}},
		},
	}
	nodes := []*bones.Node{
		{Kind: bones.DefineNode, Children: []*bones.Node{bones.IdentNode("countdown"), lambda}},
		{Kind: bones.CallNode, Children: []*bones.Node{bones.IdentNode("countdown"), bones.IntNode(100000)}},
	}
	result := compileAndRun(t, nodes, env)
	if result.IntVal != 0 {
		t.Errorf("expected 0, got %s", result.Repr())
	}
}

func TestVMScopeIsolation(t *testing.T) {
	env := NewEnvironment()
	// (define x 1)
	// ((lambda () (define x 99) x))
	// x → should still be 1
	lambda := &bones.Node{
		Kind:   bones.LambdaNode,
		Params: &bones.ParamSpec{},
		Children: []*bones.Node{
			{Kind: bones.DefineNode, Children: []*bones.Node{bones.IdentNode("x"), bones.IntNode(99)}},
			bones.IdentNode("x"),
		},
	}
	nodes := []*bones.Node{
		{Kind: bones.DefineNode, Children: []*bones.Node{bones.IdentNode("x"), bones.IntNode(1)}},
		{Kind: bones.CallNode, Children: []*bones.Node{lambda}},
		bones.IdentNode("x"),
	}
	result := compileAndRun(t, nodes, env)
	if result.IntVal != 1 {
		t.Errorf("expected 1 (outer x), got %s", result.Repr())
	}
}
