package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

func GenerateFor(path string, input io.Reader, output io.Writer) {
	io.WriteString(output, "package "+path+"_ghoul\n")
}

const testSuffix = "_test.go"

func filter(fs os.FileInfo) bool {
	name := fs.Name()
	index := len(name) - len(testSuffix)
	if index < 0 {
		index = 0
	}
	return name[index:] != testSuffix

}

type PreludeValues struct {
	PkgName string
	Pkg     string
	Imports []string
}

var PreludeTemplate, preludeErr = template.New("PreludeTemplate").Parse(
	"package {{.PkgName}}_ghoul\n\n" +
		"import (\n{{range .Imports}}\t\"{{.}}\"\n{{end}}" +
		"\n\t\"{{.Pkg}}\"" +
		"\n\tev \"github.com/archevel/ghoul/evaluator\"" +
		"\n\te \"github.com/archevel/ghoul/expressions\"\n)\n")

func CreatePrelude(pkg string, imports []string, w io.Writer) error {
	if preludeErr != nil {
		return preludeErr
	}
	pkgName := pkg
	splitOn := strings.LastIndex(pkg, "/") + 1
	if splitOn >= 0 {
		pkgName = pkg[splitOn:]
	}

	return PreludeTemplate.Execute(w, PreludeValues{pkgName, pkg, imports})
}

const FuncTemplate = `
func {{.FuncName}}(args e.List, ev *ev.Evaluator) (e.Expr, error) {
	{{.GeneratedBody}}
}
`

var BuiltInTypeTemplate, bittErr = template.New("BuiltInTypeTemplate").Parse(`
	var arg{{.N}} {{.Type}}

	switch v := args.Head().(type) {
		case e.{{.BuiltInType}}:
			arg{{.N}} = {{.Type}}(v)
		case *e.{{.BuiltInType}}:
			arg{{.N}} = {{.Type}}(*v)
		case e.Foreign:
			switch vv := v.Val().(type) {
			case *{{.Type}}:
				arg{{.N}} = *vv
			case {{.Type}}:
				arg{{.N}} = vv
			default:
				return nil, errors.New("Could not convert arg{{.N}} to {{.Type}}")
			}
		case *e.Foreign:
			switch vv := v.Val().(type) {
			case *{{.Type}}:
				arg{{.N}} = *vv
			case {{.Type}}:
				arg{{.N}} = vv
			default:
				return nil, errors.New("Could not convert arg{{.N}} to {{.Type}}")
			}
		default:
			return nil, errors.New("Could not convert arg{{.N}} to {{.Type}}")
	}
`)

var StructAndInterfaceTemplate, saitErr = template.New("StructAndInterfaceTemplate").Parse(`
	var arg{{.N}} {{.Type}}

	switch f := args.Head().(type) {
	case e.Foreign:
		switch v := f.Val().(type) {
		case *{{.Type}}:
			arg{{.N}} = *v
		case {{.Type}}:
			arg{{.N}} = v
		default:
			return nil, errors.New("Could not converrt arg{{.N}} to {{.Type}}")
		}
	case *e.Foreign:
		switch v := f.Val().(type) {
		case *{{.Type}}:
			arg{{.N}} = *v
		case {{.Type}}:
			arg{{.N}} = v
		default:
			return nil, errors.New("Could not converrt arg{{.N}} to {{.Type}}")
		}
	default:
		return nil, errors.New("Could not converrt arg{{.N}} to {{.Type}}")
	}
`)

type ArgTemplateValues struct {
	N           int
	Type        string
	BuiltInType string
}

func ConvertArg(n int, wantedType string, builtInType string, w io.Writer) error {
	if bittErr != nil {
		return bittErr
	}
	if saitErr != nil {
		return saitErr
	}
	vals := ArgTemplateValues{n, wantedType, builtInType}
	if builtInType != "" {
		return BuiltInTypeTemplate.Execute(w, vals)
	} else {
		return StructAndInterfaceTemplate.Execute(w, vals)
	}
}

var funcMap = template.FuncMap{
	"inc": func(i int) int { return i + 1 },
}

var WrapResultTemplate, wrErr = template.New("WrapResultTemplate").Funcs(funcMap).Parse(
	"\n\t{{$ResultListLength := len .ResultList}}" +
		"{{$ParamListLength := len .ParamList}}" +
		"{{range $i,$_ := .ResultList}}res{{$i}}" +
		"{{if not (eq ($ResultListLength) (inc $i))}},{{end}} " +
		"{{end}}:= {{.FuncName}}" +
		"({{range $i,$_ := .ParamList}}arg{{$i}}" +
		"{{if not (eq ($ParamListLength) (inc $i))}}, {{end}}" +
		"{{end}})" +
		"\n\tresultList := " +
		"{{range $i,$_ := .ResultList}}e.Cons(e.ToExpr(res{{$i}}), {{end}}" +
		"e.NIL{{range $i,$_ := .ResultList}}){{end}}" +
		"\n\treturn resultList, nil\n")

type ResultTemplateValues struct {
	FuncName   string
	ParamList  []int
	ResultList []int
}

func WrapResult(funcName string, paramCount int, resultCount int, w io.Writer) error {
	if wrErr != nil {
		return wrErr
	}
	vals := ResultTemplateValues{
		funcName,
		make([]int, paramCount),
		make([]int, resultCount),
	}
	return WrapResultTemplate.Execute(w, vals)
}

func main() {
	fmt.Println("")
}
