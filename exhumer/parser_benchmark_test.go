package exhumer

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func BenchmarkParsePrelude(b *testing.B) {
	data, err := os.ReadFile("../prelude/prelude.ghl")
	if err != nil {
		b.Fatalf("failed to read prelude: %v", err)
	}
	input := string(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, _ := Parse(strings.NewReader(input))
		if res != 0 {
			b.Fatal("parse failed")
		}
	}
}

func BenchmarkParsePreludeWithFilename(b *testing.B) {
	data, err := os.ReadFile("../prelude/prelude.ghl")
	if err != nil {
		b.Fatalf("failed to read prelude: %v", err)
	}
	input := string(data)
	filename := "prelude.ghl"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, _ := ParseWithFilename(strings.NewReader(input), &filename)
		if res != 0 {
			b.Fatal("parse failed")
		}
	}
}

func BenchmarkParseLargeInput(b *testing.B) {
	// Generate a large input of simple expressions (not define-syntax
	// to avoid duplicate definition errors in the evaluator).
	var buf strings.Builder
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&buf, "(+ %d %d)\n", i, i+1)
	}
	input := buf.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, _ := Parse(strings.NewReader(input))
		if res != 0 {
			b.Fatal("parse failed")
		}
	}
}

func BenchmarkParseDeepNesting(b *testing.B) {
	var buf strings.Builder
	depth := 200
	for i := 0; i < depth; i++ {
		buf.WriteByte('(')
	}
	buf.WriteString("42")
	for i := 0; i < depth; i++ {
		buf.WriteByte(')')
	}
	input := buf.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, _ := Parse(strings.NewReader(input))
		if res != 0 {
			b.Fatal("parse failed")
		}
	}
}

func BenchmarkParseManyAtoms(b *testing.B) {
	var buf strings.Builder
	buf.WriteString("(begin")
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&buf, " x%d", i)
	}
	buf.WriteString(")")
	input := buf.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, _ := Parse(strings.NewReader(input))
		if res != 0 {
			b.Fatal("parse failed")
		}
	}
}

func BenchmarkParseStringHeavy(b *testing.B) {
	var buf strings.Builder
	buf.WriteString("(begin")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&buf, ` "string number %d"`, i)
	}
	buf.WriteString(")")
	input := buf.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, _ := Parse(strings.NewReader(input))
		if res != 0 {
			b.Fatal("parse failed")
		}
	}
}

func BenchmarkParseCommentHeavy(b *testing.B) {
	var buf strings.Builder
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&buf, ";; this is comment number %d with some content\n", i)
	}
	buf.WriteString("42")
	input := buf.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, _ := Parse(strings.NewReader(input))
		if res != 0 {
			b.Fatal("parse failed")
		}
	}
}
