package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestGeneratesPackageName(t *testing.T) {

	cases := []struct {
		path                string
		expectedPackageLine string
	}{
		{"com/lo", "package com/lo_ghoul"},
		{"github.com/bar", "package github.com/bar_ghoul"},
		{"github.com/foo/bar", "package github.com/foo/bar_ghoul"},
	}

	for _, c := range cases {

		r := strings.NewReader("")
		buffer := new(bytes.Buffer)
		GenerateFor(c.path, r, buffer)
		res := buffer.String()

		if res[:len(c.expectedPackageLine)] != c.expectedPackageLine {
			t.Errorf("Expected first line to be %s. Got %s", c.expectedPackageLine, res)
		}
	}

}

func TestPrelude(t *testing.T) {
	expected :=
		"package pkg_ghoul\n\n" +
			"import (\n\t\"fmt\"\n\t\"os\"\n" +
			"\n\t\"path/to/pkg\"" +
			"\n\tev \"github.com/archevel/ghoul/evaluator\"" +
			"\n\te \"github.com/archevel/ghoul/expressions\"\n)\n"

	buffer := new(bytes.Buffer)
	w := bufio.NewWriter(buffer)
	err := CreatePrelude("path/to/pkg", []string{"fmt", "os"}, w)

	w.Flush()
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	actual := buffer.String()
	if actual != expected {
		t.Errorf("Failed to create prelude. Expected :\n%s\n, got: \n%s", expected, actual)
	}

}

func TestBooleanTemplate(t *testing.T) {

	var expected = `
	var arg9 bool

	switch v := args.Head().(type) {
		case e.Boolean:
			arg9 = bool(v)
		case *e.Boolean:
			arg9 = bool(*v)
		case e.Foreign:
			switch vv := v.Val().(type) {
			case *bool:
				arg9 = *vv
			case bool:
				arg9 = vv
			default:
				return nil, errors.New("Could not convert arg9 to bool")
			}
		case *e.Foreign:
			switch vv := v.Val().(type) {
			case *bool:
				arg9 = *vv
			case bool:
				arg9 = vv
			default:
				return nil, errors.New("Could not convert arg9 to bool")
			}
		default:
			return nil, errors.New("Could not convert arg9 to bool")
	}
`

	buffer := new(bytes.Buffer)
	w := bufio.NewWriter(buffer)
	err := ConvertArg(9, "bool", "Boolean", w)
	w.Flush()
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	actual := buffer.String()
	if actual != expected {
		t.Errorf("Converting argumenet failed. Wanted:\n%s\n, got: \n%s", expected, actual)
	}
}

func TestStringTemplate(t *testing.T) {

	var expected = `
	var arg5 string

	switch v := args.Head().(type) {
		case e.String:
			arg5 = string(v)
		case *e.String:
			arg5 = string(*v)
		case e.Foreign:
			switch vv := v.Val().(type) {
			case *string:
				arg5 = *vv
			case string:
				arg5 = vv
			default:
				return nil, errors.New("Could not convert arg5 to string")
			}
		case *e.Foreign:
			switch vv := v.Val().(type) {
			case *string:
				arg5 = *vv
			case string:
				arg5 = vv
			default:
				return nil, errors.New("Could not convert arg5 to string")
			}
		default:
			return nil, errors.New("Could not convert arg5 to string")
	}
`

	buffer := new(bytes.Buffer)
	w := bufio.NewWriter(buffer)
	err := ConvertArg(5, "string", "String", w)
	w.Flush()
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	actual := buffer.String()
	if actual != expected {
		t.Errorf("Converting argumenet failed. Wanted:\n%s\n, got: \n%s", expected, actual)
	}
}

func TestFloatTemplate(t *testing.T) {

	var expected = `
	var arg1 float

	switch v := args.Head().(type) {
		case e.Float:
			arg1 = float(v)
		case *e.Float:
			arg1 = float(*v)
		case e.Foreign:
			switch vv := v.Val().(type) {
			case *float:
				arg1 = *vv
			case float:
				arg1 = vv
			default:
				return nil, errors.New("Could not convert arg1 to float")
			}
		case *e.Foreign:
			switch vv := v.Val().(type) {
			case *float:
				arg1 = *vv
			case float:
				arg1 = vv
			default:
				return nil, errors.New("Could not convert arg1 to float")
			}
		default:
			return nil, errors.New("Could not convert arg1 to float")
	}
`

	buffer := new(bytes.Buffer)
	w := bufio.NewWriter(buffer)
	err := ConvertArg(1, "float", "Float", w)
	w.Flush()
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	actual := buffer.String()
	if actual != expected {
		t.Errorf("Converting argumenet failed. Wanted:\n%s\n, got: \n%s", expected, actual)
	}
}

func TestIntegerTemplate(t *testing.T) {

	var expected = `
	var arg1 int

	switch v := args.Head().(type) {
		case e.Integer:
			arg1 = int(v)
		case *e.Integer:
			arg1 = int(*v)
		case e.Foreign:
			switch vv := v.Val().(type) {
			case *int:
				arg1 = *vv
			case int:
				arg1 = vv
			default:
				return nil, errors.New("Could not convert arg1 to int")
			}
		case *e.Foreign:
			switch vv := v.Val().(type) {
			case *int:
				arg1 = *vv
			case int:
				arg1 = vv
			default:
				return nil, errors.New("Could not convert arg1 to int")
			}
		default:
			return nil, errors.New("Could not convert arg1 to int")
	}
`

	buffer := new(bytes.Buffer)
	w := bufio.NewWriter(buffer)
	err := ConvertArg(1, "int", "Integer", w)
	w.Flush()
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	actual := buffer.String()
	if actual != expected {
		t.Errorf("Converting argumenet failed. Wanted:\n%s\n, got: \n%s", expected, actual)
	}
}

func TestStructAndInterfaceTemplate(t *testing.T) {
	var expected = `
	var arg99 Ball

	switch f := args.Head().(type) {
	case e.Foreign:
		switch v := f.Val().(type) {
		case *Ball:
			arg99 = *v
		case Ball:
			arg99 = v
		default:
			return nil, errors.New("Could not converrt arg99 to Ball")
		}
	case *e.Foreign:
		switch v := f.Val().(type) {
		case *Ball:
			arg99 = *v
		case Ball:
			arg99 = v
		default:
			return nil, errors.New("Could not converrt arg99 to Ball")
		}
	default:
		return nil, errors.New("Could not converrt arg99 to Ball")
	}
`
	buffer := new(bytes.Buffer)
	w := bufio.NewWriter(buffer)
	err := ConvertArg(99, "Ball", "", w)
	w.Flush()
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	actual := buffer.String()
	if actual != expected {
		t.Errorf("Converting argumenet failed. Wanted:\n%s\n, got: \n%s", expected, actual)
	}
}

func TestWrapResult(t *testing.T) {
	var expected = "\n\tres0, res1, res2 := AFunc(arg0, arg1)\n" +
		"\tresultList := e.Cons(e.ToExpr(res0), e.Cons(e.ToExpr(res1), e.Cons(e.ToExpr(res2), e.NIL)))\n" +
		"\treturn resultList, nil\n"

	buffer := new(bytes.Buffer)
	w := bufio.NewWriter(buffer)
	err := WrapResult("AFunc", 2, 3, w)
	w.Flush()
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	actual := buffer.String()
	if actual != expected {
		t.Errorf("Converting result failed. Wanted:\n%s\n, got: \n%s", expected, actual)
	}
}

func TestFunctionArgTemplate(t *testing.T) {
	/*	var expected = `
			var arg9 func(int) int

			switch f := args.Head().(type) {
			case e.Foreign:
				switch v := f.Val().(type) {
				case *func(int) int:
					arg9 = *v
				case func(int) int:
					arg9 = v
				default:
					return nil, errors.New("Could not converrt arg9 to func(int) int")
				}
			case *e.Foreign:
				switch v := f.Val().(type) {
				case *func(int) int:
					arg9 = *v
				case func(int) int:
					arg9 = v
				default:
					return nil, errors.New("Could not converrt arg9 to func(int) int")
				}
			case e.Function:
			case *e.Function:
				arg9 = func(p1 int) int {
					args := e.NIL
					TODO!!!!!
				}
			default:
				return nil, errors.New("Could not converrt arg9 to func(int) int")
			}
		`
			buffer := new(bytes.Buffer)
			w := bufio.NewWriter(buffer)
			err := ConvertArg(9, "func(int) int", "Function", w)
			w.Flush()
			if err != nil {
				t.Errorf("Got unexpected error: %s", err)
			}

			actual := buffer.String()
			if actual != expected {
				t.Errorf("Converting argumenet failed. Wanted:\n%s\n, got: \n%s", expected, actual)
			}*/
	t.Log("Not implemented...")
}
