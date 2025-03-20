package preprocessor

import "strings"

type CodePreprocessor interface {
	Preprocess(code string) string
}

type DefaultPreprocessor struct{}

// Preprocess trims leading and trailing white spaces from the given code string.
// It takes a single parameter:
// - code: the input string containing the code to be preprocessed.
// It returns a string with the leading and trailing white spaces removed.
func (dp *DefaultPreprocessor) Preprocess(code string) string {
	return strings.TrimSpace(code)
}
