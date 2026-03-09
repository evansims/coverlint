package coverage

import (
	"bytes"
	"testing"
)

func TestAnnotatorAllMode(t *testing.T) {
	var buf bytes.Buffer
	a := NewAnnotator(AnnotationConfig{Mode: "all"}, &buf)

	a.Emit("notice", "hello world")
	a.Emit("warning", "something happened")
	a.Emit("error", "bad thing")

	if a.Count() != 3 {
		t.Errorf("Count() = %d, want 3", a.Count())
	}

	want := "::notice::hello world\n::warning::something happened\n::error::bad thing\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}

func TestAnnotatorNoneMode(t *testing.T) {
	var buf bytes.Buffer
	a := NewAnnotator(AnnotationConfig{Mode: "none"}, &buf)

	a.Emit("notice", "hello world")
	a.Emit("warning", "something happened")

	if a.Count() != 0 {
		t.Errorf("Count() = %d, want 0", a.Count())
	}
	if buf.String() != "" {
		t.Errorf("output = %q, want empty", buf.String())
	}
}

func TestAnnotatorLimitedMode(t *testing.T) {
	var buf bytes.Buffer
	a := NewAnnotator(AnnotationConfig{Mode: "limited", MaxCount: 2}, &buf)

	a.Emit("notice", "first")
	a.Emit("warning", "second")
	a.Emit("error", "third — should be dropped")

	if a.Count() != 2 {
		t.Errorf("Count() = %d, want 2", a.Count())
	}

	want := "::notice::first\n::warning::second\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}

func TestAnnotatorLimitedZero(t *testing.T) {
	var buf bytes.Buffer
	a := NewAnnotator(AnnotationConfig{Mode: "limited", MaxCount: 0}, &buf)

	a.Emit("notice", "should be dropped")

	if a.Count() != 0 {
		t.Errorf("Count() = %d, want 0", a.Count())
	}
	if buf.String() != "" {
		t.Errorf("output = %q, want empty", buf.String())
	}
}

func TestAnnotatorSanitization(t *testing.T) {
	var buf bytes.Buffer
	a := NewAnnotator(AnnotationConfig{Mode: "all"}, &buf)

	a.Emit("warning", "line1\nline2\r::inject::payload")

	want := "::warning::line1 line2 : :inject: :payload\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}
