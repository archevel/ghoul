package macromancy

import (
	"fmt"

	e "github.com/archevel/ghoul/expressions"
)

type MacroGroup struct {
	matchId e.Identifier
	macros  []Macro
}

func (mg MacroGroup) Matches(code e.Expr) []Macro {
	id, ok := code.(e.Identifier)
	if codeList, codeOk := code.(e.List); !ok && codeOk {
		id, _ = codeList.First().(e.Identifier)
	}
	if mg.matchId.Equiv(id) {
		return mg.macros
	}
	return nil
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

	rules, rulesErr := extractRulesList(syntaxDefList)
	if rulesErr != nil {
		return nil, rulesErr
	}

	macros, err := extractMacros(rules.First().(e.List))
	if err != nil {
		return nil, err
	}

	return &MacroGroup{matchId, macros}, nil
}

func extractRulesList(syntaxDefList e.List) (e.List, error) {
	syntaxRulesList, syntaxRulesListOk := syntaxDefList.Tail()
	if !syntaxRulesListOk {
		return nil, fmt.Errorf("invalid syntax-rules: expected syntax-rules form, got %s", syntaxDefList.Repr())
	}

	syntaxRules, syntaxRulesOk := syntaxRulesList.First().(e.List)
	if !syntaxRulesOk || !e.Identifier("syntax-rules").Equiv(syntaxRules.First()) || e.NIL.Equiv(syntaxRules.Second()) {
		if syntaxRulesOk {
			return nil, fmt.Errorf("invalid syntax-rules: malformed syntax-rules structure in %s", syntaxRules.Repr())
		} else {
			return nil, fmt.Errorf("invalid syntax-rules: expected syntax-rules form, got %s", syntaxRulesList.First().Repr())
		}
	}

	litsAndRules, litsAndRulesOk := syntaxRules.Tail()
	if !litsAndRulesOk || e.NIL.Equiv(litsAndRules.Second()) {
		if litsAndRulesOk {
			return nil, fmt.Errorf("invalid rules in syntax definition: missing rules list in %s", litsAndRules.Repr())
		} else {
			return nil, fmt.Errorf("invalid rules in syntax definition: expected literals and rules, got %s", syntaxRules.Repr())
		}
	}

	rules, rulesOk := litsAndRules.Tail()
	if !rulesOk {
		return nil, fmt.Errorf("invalid rules in syntax definition: expected rules after literals, got %s", litsAndRules.Repr())
	}

	return rules, nil
}

func extractMacros(rules e.List) ([]Macro, error) {

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
		macros = append(macros, Macro{pat, bdyList.First()})

	}
	return macros, nil
}
