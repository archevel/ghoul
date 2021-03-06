package macromancy

import (
	"errors"

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
		return nil, errors.New("Invalid syntax definition.")
	}
	syntaxDefList, syntaxDefOk := codeList.Tail()
	if !syntaxDefOk {
		return nil, errors.New("Invalid syntax definition.")
	}
	matchId, matchIdOk := syntaxDefList.First().(e.Identifier)
	if !matchIdOk {
		return nil, errors.New("Identifier for macro group '" + syntaxDefList.First().Repr() + "' is invalid.")
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
		return nil, errors.New("Invalid syntax-rules.")
	}

	syntaxRules, syntaxRulesOk := syntaxRulesList.First().(e.List)
	if !syntaxRulesOk || !e.Identifier("syntax-rules").Equiv(syntaxRules.First()) || e.NIL.Equiv(syntaxRules.Second()) {
		return nil, errors.New("Invalid syntax-rules.")
	}

	litsAndRules, litsAndRulesOk := syntaxRules.Tail()
	if !litsAndRulesOk || e.NIL.Equiv(litsAndRules.Second()) {
		return nil, errors.New("Invalid rules in syntax definition.")
	}

	rules, rulesOk := litsAndRules.Tail()
	if !rulesOk {
		return nil, errors.New("Invalid rules in syntax definition.")
	}

	return rules, nil
}

func extractMacros(rules e.List) ([]Macro, error) {

	macros := []Macro{}
	rulesOk := false
	for rules != e.NIL {
		r, rOk := rules.First().(e.List)
		rules, rulesOk = rules.Tail()
		if !rOk || !rulesOk {
			return nil, errors.New("Invalid rule definition.")
		}
		pat := r.First()
		bdyList, bdyOk := r.Tail()

		if !bdyOk || e.NIL.Equiv(bdyList) {
			return nil, errors.New("Invalid rule definition.")
		}
		macros = append(macros, Macro{pat, bdyList.First()})

	}
	return macros, nil
}
