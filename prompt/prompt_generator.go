package prompt

import "strings"

type PromptGenerator interface {
	GeneratePrompt(code string, fileName string) string
}

type SimplePromptGenerator struct {
	Template string
}

// GeneratePrompt generates a prompt by replacing placeholders in the template
// with the provided code and file name.
//
// Parameters:
//   - code: The code snippet to be included in the prompt.
//   - fileName: The name of the file containing the code snippet.
//
// Returns:
//   A string with the placeholders in the template replaced by the provided code and file name.
func (spg *SimplePromptGenerator) GeneratePrompt(code string, fileName string) string {
	prompt := spg.Template
	prompt = strings.ReplaceAll(prompt, "{code}", code)
	prompt = strings.ReplaceAll(prompt, "{fileName}", fileName)
	return prompt
}

