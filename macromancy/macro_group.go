package macromancy

import (
	"fmt"

	e "github.com/archevel/ghoul/expressions"
)

type MacroGroup struct {
	matchId e.Identifier
	macros  []Macro
}

func (mg MacroGroup) Macros() []Macro {
	return mg.macros
}

func NewMacroGroup(code e.Expr) (*MacroGroup, error) {
	codeList, codeOk := code.(e.List)
	if !codeOk {
		return nil, fmt.Errorf("invalid syntax definition: expected (define-syntax <identifier> <transformer>), got %s", code.Repr())
	}
	syntaxDefList, syntaxDefOk := codeList.Tail()
	if !syntaxDefOk {
		return nil, fmt.Errorf("invalid syntax definition: expected list with syntax transformer, got %s", codeList.Repr())
	}
	matchId, matchIdOk := syntaxDefList.First().(e.Identifier)
	if !matchIdOk {
		return nil, fmt.Errorf("invalid identifier %s for macro group: must be an identifier", syntaxDefList.First().Repr())
	}

	literals, rules, rulesErr := extractLiteralsAndRules(syntaxDefList)
	if rulesErr != nil {
		return nil, rulesErr
	}

	macros, err := extractMacros(rules.First().(e.List), literals)
	if err != nil {
		return nil, err
	}

	return &MacroGroup{matchId, macros}, nil
}

func extractLiteralsAndRules(syntaxDefList e.List) (map[e.Identifier]bool, e.List, error) {
	syntaxRulesList, syntaxRulesListOk := syntaxDefList.Tail()
	if !syntaxRulesListOk {
		return nil, nil, fmt.Errorf("invalid syntax-rules: expected syntax-rules form, got %s", syntaxDefList.Repr())
	}

	syntaxRules, syntaxRulesOk := syntaxRulesList.First().(e.List)
	if !syntaxRulesOk || !e.Identifier("syntax-rules").Equiv(syntaxRules.First()) || e.NIL.Equiv(syntaxRules.Second()) {
		if syntaxRulesOk {
			return nil, nil, fmt.Errorf("invalid syntax-rules: malformed syntax-rules structure in %s", syntaxRules.Repr())
		} else {
			return nil, nil, fmt.Errorf("invalid syntax-rules: expected syntax-rules form, got %s", syntaxRulesList.First().Repr())
		}
	}

	litsAndRules, litsAndRulesOk := syntaxRules.Tail()
	if !litsAndRulesOk || e.NIL.Equiv(litsAndRules.Second()) {
		if litsAndRulesOk {
			return nil, nil, fmt.Errorf("invalid rules in syntax definition: missing rules list in %s", litsAndRules.Repr())
		} else {
			return nil, nil, fmt.Errorf("invalid rules in syntax definition: expected literals and rules, got %s", syntaxRules.Repr())
		}
	}

	literals := extractLiterals(litsAndRules.First())

	rules, rulesOk := litsAndRules.Tail()
	if !rulesOk {
		return nil, nil, fmt.Errorf("invalid rules in syntax definition: expected rules after literals, got %s", litsAndRules.Repr())
	}

	return literals, rules, nil
}

func extractLiterals(litExpr e.Expr) map[e.Identifier]bool {
	literals := map[e.Identifier]bool{}
	list, ok := litExpr.(e.List)
	if !ok {
		return literals
	}
	for list != e.NIL {
		if id, ok := list.First().(e.Identifier); ok {
			literals[id] = true
		}
		tail, ok := list.Tail()
		if !ok {
			break
		}
		list = tail
	}
	return literals
}

func extractMacros(rules e.List, literals map[e.Identifier]bool) ([]Macro, error) {

	macros := []Macro{}
	rulesOk := false
	for rules != e.NIL {
		first := rules.First()
		r, rOk := first.(e.List)
		rules, rulesOk = rules.Tail()
		if !rOk {
			return nil, fmt.Errorf("invalid rule definition: expected list for rule, got %T at position %d", first, len(macros))
		}
		if !rulesOk {
			return nil, fmt.Errorf("invalid rule definition: malformed rules list at position %d", len(macros))
		}
		pat := r.First()
		bdyList, bdyOk := r.Tail()

		if !bdyOk || e.NIL.Equiv(bdyList) {
			return nil, fmt.Errorf("invalid rule definition: rule must have pattern and body, got %s", r.Repr())
		}
		macros = append(macros, Macro{
			Pattern:     pat,
			Body:        bdyList.First(),
			PatternVars: ExtractPatternVarsWithLiterals(pat, literals),
			Literals:    literals,
		})

	}
	return macros, nil
}
