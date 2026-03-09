package calc

import "testing"

func TestMultiply(t *testing.T) {
	if Multiply(3, 4) != 12 {
		t.Error("expected 12")
	}
}

func TestAbs(t *testing.T) {
	if Abs(-5) != 5 {
		t.Error("expected 5")
	}
	if Abs(3) != 3 {
		t.Error("expected 3")
	}
	if Abs(0) != 0 {
		t.Error("expected 0")
	}
}
