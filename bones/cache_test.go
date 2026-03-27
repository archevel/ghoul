package bones

import "testing"

// --- Singleton tests: cached values should return the same pointer ---

func TestBoolNodeTrueSingleton(t *testing.T) {
	a := BoolNode(true)
	b := BoolNode(true)
	if a != b {
		t.Error("BoolNode(true) should return the same pointer")
	}
}

func TestBoolNodeFalseSingleton(t *testing.T) {
	a := BoolNode(false)
	b := BoolNode(false)
	if a != b {
		t.Error("BoolNode(false) should return the same pointer")
	}
}

func TestBoolNodeTrueNotEqualFalse(t *testing.T) {
	if BoolNode(true) == BoolNode(false) {
		t.Error("true and false singletons should be different")
	}
}

func TestIntNodeZeroSingleton(t *testing.T) {
	a := IntNode(0)
	b := IntNode(0)
	if a != b {
		t.Error("IntNode(0) should return the same pointer")
	}
}

func TestIntNodeOneSingleton(t *testing.T) {
	a := IntNode(1)
	b := IntNode(1)
	if a != b {
		t.Error("IntNode(1) should return the same pointer")
	}
}

func TestIntNodeSmallNegativeSingleton(t *testing.T) {
	a := IntNode(-1)
	b := IntNode(-1)
	if a != b {
		t.Error("IntNode(-1) should return the same pointer")
	}
}

func TestIntNodeBoundarySingleton(t *testing.T) {
	// Cache boundary: 127
	a := IntNode(127)
	b := IntNode(127)
	if a != b {
		t.Error("IntNode(127) should return the same pointer")
	}

	// -128
	a = IntNode(-128)
	b = IntNode(-128)
	if a != b {
		t.Error("IntNode(-128) should return the same pointer")
	}
}

func TestIntNodeOutsideCacheNotSingleton(t *testing.T) {
	// Values outside the cache range should NOT be the same pointer
	a := IntNode(128)
	b := IntNode(128)
	if a == b {
		t.Error("IntNode(128) should allocate fresh (outside cache)")
	}

	a = IntNode(-129)
	b = IntNode(-129)
	if a == b {
		t.Error("IntNode(-129) should allocate fresh (outside cache)")
	}
}

func TestIntNodeLargeValueCorrect(t *testing.T) {
	// Large values should still work correctly even if not cached
	n := IntNode(1000000)
	if n.IntVal != 1000000 {
		t.Errorf("expected 1000000, got %d", n.IntVal)
	}
}

func TestNilSingleton(t *testing.T) {
	// Nil should always be the same pointer
	a := Nil
	b := Nil
	if a != b {
		t.Error("Nil should be a singleton")
	}
}

// --- Cached values should still behave correctly ---

func TestCachedBoolRepr(t *testing.T) {
	if BoolNode(true).Repr() != "#t" {
		t.Error("cached true should repr as #t")
	}
	if BoolNode(false).Repr() != "#f" {
		t.Error("cached false should repr as #f")
	}
}

func TestCachedIntRepr(t *testing.T) {
	if IntNode(0).Repr() != "0" {
		t.Error("cached 0 should repr as 0")
	}
	if IntNode(42).Repr() != "42" {
		t.Error("cached 42 should repr as 42")
	}
	if IntNode(-1).Repr() != "-1" {
		t.Error("cached -1 should repr as -1")
	}
}

func TestCachedIntEquiv(t *testing.T) {
	if !IntNode(0).Equiv(IntNode(0)) {
		t.Error("cached 0 should equal cached 0")
	}
	if IntNode(0).Equiv(IntNode(1)) {
		t.Error("cached 0 should not equal cached 1")
	}
}

func TestCachedBoolEquiv(t *testing.T) {
	if !BoolNode(true).Equiv(BoolNode(true)) {
		t.Error("cached true should equal cached true")
	}
	if BoolNode(true).Equiv(BoolNode(false)) {
		t.Error("cached true should not equal cached false")
	}
}

// --- Allocation benchmarks ---

func BenchmarkBoolNodeAllocs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = BoolNode(true)
	}
}

func BenchmarkIntNodeZeroAllocs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = IntNode(0)
	}
}

func BenchmarkIntNodeSmallAllocs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = IntNode(42)
	}
}

func BenchmarkIntNodeLargeAllocs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = IntNode(1000)
	}
}
