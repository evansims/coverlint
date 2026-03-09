package coverage

import (
	"fmt"
	"io"
)

// Annotator controls annotation emission based on an AnnotationConfig.
// It tracks how many annotations have been emitted and respects mode/cap settings.
type Annotator struct {
	config AnnotationConfig
	writer io.Writer
	count  int
}

// NewAnnotator creates an Annotator that writes to the given writer
// using the provided configuration.
func NewAnnotator(config AnnotationConfig, w io.Writer) *Annotator {
	return &Annotator{
		config: config,
		writer: w,
	}
}

// Emit writes a GitHub Actions annotation if permitted by the current config.
// It does nothing when annotations are disabled or the cap has been reached.
func (a *Annotator) Emit(level, message string) {
	if a.config.Mode == "none" {
		return
	}
	if a.config.Mode == "limited" && a.count >= a.config.MaxCount {
		return
	}
	_, _ = fmt.Fprintf(a.writer, "::%s::%s\n", level, sanitizeWorkflowCommand(message))
	a.count++
}

// Count returns the number of annotations emitted so far.
func (a *Annotator) Count() int {
	return a.count
}
