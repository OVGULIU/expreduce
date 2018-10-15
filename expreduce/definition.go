package expreduce

import "strings"

type DownValue struct {
	rule        *Expression
	specificity int
}

type Def struct {
	downvalues  []DownValue
	attributes  Attributes
	defaultExpr Ex

	// A function defined here will override downvalues.
	legacyEvalFn (func(*Expression, *EvalState) Ex)
}

func (def *Def) StringForm(defSym *Symbol, params ToStringParams) string {
	var buffer []string

	attrs := def.attributes.toStrings()
	if len(attrs) > 0 {
		e := E(
			S("Set"),
			E(
				S("Attributes"),
				defSym,
			),
			def.attributes.toSymList(),
		)
		buffer = append(buffer, e.StringForm(params))
	}

	for _, dv := range def.downvalues {
		e := E(
			S("SetDelayed"),
			dv.rule.Parts[1].(*Expression).Parts[1],
			dv.rule.Parts[2],
		)
		buffer = append(buffer, e.StringForm(params))
	}

	if def.defaultExpr != nil {
		e := E(
			S("Set"),
			E(
				S("Default"),
				defSym,
			),
			def.defaultExpr,
		)
		buffer = append(buffer, e.StringForm(params))
	}
	return strings.Join(buffer[:], "\n\n")
}
