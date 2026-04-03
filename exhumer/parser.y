%{
package exhumer

import (
	e "github.com/archevel/ghoul/bones"
	"io"
	"strconv"
	"strings"
)


%}

%union {
  node *e.Node
  tok string
  row int
  col int
}

%token UNEXPECTED_TOKEN
%token QUOTE
%token DOT
%token IDENTIFIER
%token DASH
%token INTEGER
%token FLOAT
%token TRUE
%token FALSE
%token HASHBANG
%token STRING
%token BEG_LIST
%token END_LIST

%%
progr: sexpr
     { l := yylex.(*schemeLexer)
       l.result = $1.node }
    | HASHBANG sexpr
     { l := yylex.(*schemeLexer)
       l.result = $2.node }
;
sexpr:
     { $$.node = e.Nil }
     | value sexpr
     {
       list := $2.node
       children := make([]*e.Node, 0, len(list.Children)+1)
       children = append(children, $1.node)
       children = append(children, list.Children...)
       result := e.NewListNode(children)
       pos := Position{$1.row, $1.col}
       l := yylex.(*schemeLexer)
       result.Loc = &e.SourcePosition{Ln: pos.Row, Col: pos.Col, Filename: l.Filename}
       $$.node = result
     }
;
value:
     INTEGER
     { i, _ := strconv.ParseInt($1.tok, 0, 64)
     $$.node = e.IntNode(i) }
     | FLOAT
     { f, _ := strconv.ParseFloat($1.tok, 64)
     $$.node = e.FloatNode(f) }
     | TRUE
     { $$.node = e.BoolNode(true) }
     | FALSE
     { $$.node = e.BoolNode(false) }
     | IDENTIFIER
     { $$.node = e.IdentNode($1.tok) }
     | QUOTE value
     { $$.node = e.QuoteNodeVal($2.node) }
     | STRING
     { $$.node = e.StrNode(strings.Trim($1.tok, "\"`")) }
     | BEG_LIST value sexpr DOT value END_LIST
     {
       children := make([]*e.Node, 0, len($3.node.Children)+1)
       children = append(children, $2.node)
       children = append(children, $3.node.Children...)
       result := &e.Node{Kind: e.ListNode, Children: children, DottedTail: $5.node}
       pos := Position{$2.row, $2.col}
       l := yylex.(*schemeLexer)
       result.Loc = &e.SourcePosition{Ln: pos.Row, Col: pos.Col, Filename: l.Filename}
       $$.node = result
     }
     | BEG_LIST sexpr END_LIST
     { $$.node = $2.node }
;
%%

type ParsedExpressions struct {
	Expressions *e.Node
}

func Parse(r io.Reader) (int, *ParsedExpressions) {
	return ParseWithFilename(r, nil)
}

func ParseWithFilename(r io.Reader, filename *string) (int, *ParsedExpressions) {
	lex := NewLexer(r)
	lex.Filename = filename
	res := yyParse(lex)
	return res, &ParsedExpressions{Expressions: lex.result}
}

func ParseWithDebug(r io.Reader, filename *string) (int, *ParsedExpressions) {
	lex := NewDebugLexer(r)
	lex.Filename = filename
	res := yyParse(lex)
	return res, &ParsedExpressions{Expressions: lex.result}
}
