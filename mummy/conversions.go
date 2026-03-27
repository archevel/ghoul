package mummy

import (
	"fmt"

	e "github.com/archevel/ghoul/bones"
)

// NodeConversionFunc is the signature for Node-based conversion functions.
type NodeConversionFunc func(args []*e.Node, evaluator interface{}) (*e.Node, error)

func BytesConvNode(args []*e.Node, evaluator interface{}) (*e.Node, error) {
	if len(args) == 0 || args[0].Kind != e.StringNode {
		typeName := "empty"
		if len(args) > 0 {
			typeName = e.NodeTypeName(args[0])
		}
		return nil, fmt.Errorf("bytes: expected string, got %s", typeName)
	}
	return e.MummyNodeVal([]byte(args[0].StrVal), "[]byte"), nil
}

func StringFromBytesNode(args []*e.Node, evaluator interface{}) (*e.Node, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("string-from-bytes: expected mummy wrapping []byte")
	}
	if args[0].Kind != e.MummyNode {
		return nil, fmt.Errorf("string-from-bytes: expected mummy wrapping []byte, got %s", e.NodeTypeName(args[0]))
	}
	bs, ok := args[0].ForeignVal.([]byte)
	if !ok {
		return nil, fmt.Errorf("string-from-bytes: mummy contains %T, expected []byte", args[0].ForeignVal)
	}
	return e.StrNode(string(bs)), nil
}

func IntSliceNode(args []*e.Node, evaluator interface{}) (*e.Node, error) {
	var result []int
	for _, arg := range args {
		if arg.Kind != e.IntegerNode {
			return nil, fmt.Errorf("int-slice: expected integer, got %s", e.NodeTypeName(arg))
		}
		result = append(result, int(arg.IntVal))
	}
	if result == nil {
		result = []int{}
	}
	return e.MummyNodeVal(result, "[]int"), nil
}

func FloatSliceNode(args []*e.Node, evaluator interface{}) (*e.Node, error) {
	var result []float64
	for _, arg := range args {
		if arg.Kind != e.FloatNodeKind {
			return nil, fmt.Errorf("float-slice: expected float, got %s", e.NodeTypeName(arg))
		}
		result = append(result, arg.FloatVal)
	}
	if result == nil {
		result = []float64{}
	}
	return e.MummyNodeVal(result, "[]float64"), nil
}

func GoNilNode(args []*e.Node, evaluator interface{}) (*e.Node, error) {
	return e.MummyNodeVal(nil, "nil"), nil
}
