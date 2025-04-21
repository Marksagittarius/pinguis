package postprocessor

import (
	"regexp"
	"strings"
)

type CodePostprocessor interface {
	Postprocess(raw string) string
}

type PythonCodeExtractor struct{}

// Postprocess extracts Python code blocks from the given raw string.
// It searches for text enclosed in triple backticks (```), optionally
// prefixed with "python". If a match is found, it returns the content
// inside the backticks, trimmed of any leading or trailing whitespace.
// If no match is found, it returns the original raw string, also trimmed
// of any leading or trailing whitespace.
//
// Parameters:
//   raw - the input string potentially containing Python code blocks.
//
// Returns:
//   A string containing the extracted Python code block or the trimmed
//   original input string if no code block is found.
func (pce *PythonCodeExtractor) Postprocess(raw string) string {
	re := regexp.MustCompile("(?s)```(?:python)?\\n(.*?)```")
	match := re.FindStringSubmatch(raw)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return strings.TrimSpace(raw)
}

type CodeExtractor struct{
	codeType string
}

func NewCodeExtractor(codeType string) *CodeExtractor {
	return &CodeExtractor{
		codeType: codeType,
	}
}

func (ce *CodeExtractor) Postprocess(raw string) string {
	re := regexp.MustCompile("(?s)```" + ce.codeType + "\\n(.*?)```")
	match := re.FindStringSubmatch(raw)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return strings.TrimSpace(raw)
}
