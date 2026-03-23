%{
package parser

import (
	e "github.com/archevel/ghoul/expressions"
	"io"
	"strconv"
	"strings"
)


%}

%union {
  expr e.Expr
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
       l.lpair = $1.expr.(e.List) }
    | HASHBANG sexpr
     { l := yylex.(*schemeLexer)
       l.lpair = $2.expr.(e.List) }
;
sexpr:
     { $$.expr = e.NIL}
     | value sexpr
     {
       pair := e.Cons($1.expr, $2.expr)
       pos := Position{$1.row, $1.col}
       pair.Loc = &e.SourcePosition{Ln: pos.Row, Col: pos.Col}
       $$.expr = pair
	   l := yylex.(*schemeLexer)
	   l.SetPairSrcPosition(pair, pos)
     }
;
value:
     INTEGER
     { i, _ := strconv.ParseInt($1.tok, 0, 64)
     $$.expr = e.Integer(i) }
     | FLOAT
     { f, _ := strconv.ParseFloat($1.tok, 64)
     $$.expr = e.Float(f) }
     | TRUE
     { $$.expr = e.Boolean(true) }
     | FALSE
     { $$.expr = e.Boolean(false) }
     | IDENTIFIER
     { $$.expr = e.Identifier($1.tok) }
     | QUOTE value
     { $$.expr = &e.Quote{$2.expr} }
     | STRING
     { $$.expr = e.String(strings.Trim($1.tok, "\"`")) }
     | BEG_LIST value sexpr DOT value END_LIST
     { 
       p := e.Cons($2.expr, $3.expr)
       pos := Position{$2.row, $2.col}
       p.Loc = &e.SourcePosition{Ln: pos.Row, Col: pos.Col}
       $$.expr = setLastTail(p, $5.expr)
	   l := yylex.(*schemeLexer)
	   l.SetPairSrcPosition(p, pos)
     }
     | BEG_LIST sexpr END_LIST
     { $$.expr = $2.expr }
;
%%

type ParsedExpressions struct {
	Expressions      e.List
	pairSrcPositions map[e.Pair]Position
}

func (pe ParsedExpressions) PositionOf(p e.Pair) (pos Position, found bool) {
	pos, found = pe.pairSrcPositions[p]
	return
}

func Parse(r io.Reader) (int, *ParsedExpressions) {
	lex := NewLexer(r)
	res := yyParse(lex)
	return res, &ParsedExpressions{lex.lpair, lex.PairSrcPositions}
}

func setLastTail(p *e.Pair, newEnd e.Expr) *e.Pair {
	lastPair := p
	tail, _ := lastPair.Tail()
	for tail != e.NIL {
		lastPair = tail.(*e.Pair)
		tail, _ = lastPair.Tail()
	}

	lastPair.T = newEnd
	return p
}
