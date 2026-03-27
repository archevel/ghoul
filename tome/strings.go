package tome

import (
	"fmt"
	"strconv"
	"strings"

	e "github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
)

func registerStrings(env *ev.Environment) {
	env.Register("string-append", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		var b strings.Builder
		for _, arg := range args {
			if arg.Kind != e.StringNode {
				return nil, fmt.Errorf("string-append: expected string, got %s", e.NodeTypeName(arg))
			}
			b.WriteString(arg.StrVal)
		}
		return e.StrNode(b.String()), nil
	})

	env.Register("string-length", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.StringNode {
			return nil, fmt.Errorf("string-length: expected string, got %s", e.NodeTypeName(args[0]))
		}
		return e.IntNode(int64(len([]rune(args[0].StrVal)))), nil
	})

	env.Register("substring", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.StringNode {
			return nil, fmt.Errorf("substring: expected string as first argument, got %s", e.NodeTypeName(args[0]))
		}
		runes := []rune(args[0].StrVal)
		if args[1].Kind != e.IntegerNode {
			return nil, fmt.Errorf("substring: expected integer as second argument, got %s", e.NodeTypeName(args[1]))
		}
		start := args[1].IntVal
		if args[2].Kind != e.IntegerNode {
			return nil, fmt.Errorf("substring: expected integer as third argument, got %s", e.NodeTypeName(args[2]))
		}
		end := args[2].IntVal
		if int(start) < 0 || int(end) > len(runes) || int(start) > int(end) {
			return nil, fmt.Errorf("substring: index out of bounds (start=%d, end=%d, length=%d)", start, end, len(runes))
		}
		return e.StrNode(string(runes[start:end])), nil
	})

	env.Register("string-ref", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.StringNode {
			return nil, fmt.Errorf("string-ref: expected string, got %s", e.NodeTypeName(args[0]))
		}
		runes := []rune(args[0].StrVal)
		if args[1].Kind != e.IntegerNode {
			return nil, fmt.Errorf("string-ref: expected integer index, got %s", e.NodeTypeName(args[1]))
		}
		idx := args[1].IntVal
		if int(idx) < 0 || int(idx) >= len(runes) {
			return nil, fmt.Errorf("string-ref: index %d out of bounds (length %d)", idx, len(runes))
		}
		return e.StrNode(string(runes[idx])), nil
	})

	env.Register("string-contains?", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.StringNode {
			return nil, fmt.Errorf("string-contains?: expected string as first argument, got %s", e.NodeTypeName(args[0]))
		}
		if args[1].Kind != e.StringNode {
			return nil, fmt.Errorf("string-contains?: expected string as second argument, got %s", e.NodeTypeName(args[1]))
		}
		return e.BoolNode(strings.Contains(args[0].StrVal, args[1].StrVal)), nil
	})

	env.Register("string-split", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.StringNode {
			return nil, fmt.Errorf("string-split: expected string as first argument, got %s", e.NodeTypeName(args[0]))
		}
		if args[1].Kind != e.StringNode {
			return nil, fmt.Errorf("string-split: expected string as second argument, got %s", e.NodeTypeName(args[1]))
		}
		parts := strings.Split(args[0].StrVal, args[1].StrVal)
		children := make([]*e.Node, len(parts))
		for i, p := range parts {
			children[i] = e.StrNode(p)
		}
		return e.NewListNode(children), nil
	})

	env.Register("string-upcase", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.StringNode {
			return nil, fmt.Errorf("string-upcase: expected string, got %s", e.NodeTypeName(args[0]))
		}
		return e.StrNode(strings.ToUpper(args[0].StrVal)), nil
	})

	env.Register("string-downcase", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.StringNode {
			return nil, fmt.Errorf("string-downcase: expected string, got %s", e.NodeTypeName(args[0]))
		}
		return e.StrNode(strings.ToLower(args[0].StrVal)), nil
	})

	env.Register("string->number", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		if args[0].Kind != e.StringNode {
			return nil, fmt.Errorf("string->number: expected string, got %s", e.NodeTypeName(args[0]))
		}
		str := args[0].StrVal
		if i, err := strconv.ParseInt(str, 10, 64); err == nil {
			return e.IntNode(i), nil
		}
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return e.FloatNode(f), nil
		}
		return nil, fmt.Errorf("string->number: cannot parse '%s' as a number", str)
	})

	env.Register("number->string", func(args []*e.Node, evaluator *ev.Evaluator) (*e.Node, error) {
		switch args[0].Kind {
		case e.IntegerNode:
			return e.StrNode(strconv.FormatInt(args[0].IntVal, 10)), nil
		case e.FloatNodeKind:
			return e.StrNode(strconv.FormatFloat(args[0].FloatVal, 'g', -1, 64)), nil
		default:
			return nil, fmt.Errorf("number->string: expected number, got %s", e.NodeTypeName(args[0]))
		}
	})
}
