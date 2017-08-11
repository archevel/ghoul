package macromancy

import (
	"strings"
	"testing"

	"github.com/archevel/ghoul/parser"
)

func TestMacromancerDoesNotModifyNonMacroCode(t *testing.T) {

	cases := []struct {
		in  string
		out string
	}{
		{`(a b c)`, `((a b c))`},
	}

	for _, c := range cases {
		ok, parseRes := parser.Parse(strings.NewReader(c.in))

		if ok != 0 {
			t.Errorf("Failed to parse: %s\n", c)
		}

		var transformer Transformer = Macromancer{}
		mancedCode, err := transformer.Transform(parseRes.Expressions)

		if err != nil {
			t.Errorf(`Got error: "%s" when mancing code: %s`, err, c)
		}

		if mancedCode.Repr() != c.out {
			t.Errorf(`Macromancing "%s" failed, should have the result "%s", but the result was "%s"`, c.in, c.out, mancedCode.Repr())
		}

	}

}
