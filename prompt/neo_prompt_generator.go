// NeoPromptGenerator is a struct that represents a prompt generator with a customizable template.
//
// Fields:
//   - Template: A string representing the template used for generating prompts.
//
// Methods:
//   - NewNeoPromptGenerator(template string) *NeoPromptGenerator:
//       Creates a new instance of NeoPromptGenerator with the specified template.
//   - String() string:
//       Returns the current template as a string.
//   - WithContent(content string) *NeoPromptGenerator:
//       Updates the template with the provided content and returns the updated instance.
//   - GeneratePrompt(code string, fileName string) string:
//       Generates a prompt by replacing placeholders in the template with the provided code and file name.
//
// GeneratePrompt Method:
//   - Parameters:
//       - code: A string representing the code to be inserted into the template.
//       - fileName: A string representing the file name to be inserted into the template.
//   - Returns:
//       - A string containing the generated prompt with placeholders replaced by the provided values.
//
// WeaviateHandler:
//   - A type alias for a function that takes a pointer to a dao.Weaviate instance and a string,
//     and returns a string. This can be used to process and modify the template dynamically.
package prompt

import (
	"strings"

	"github.com/Marksagittarius/pinguis/dao"
)

type NeoPromptGenerator struct {
	Template string
	Code string
	FileName string
}

func NewNeoPromptGenerator(template string, code string, fileName string) *NeoPromptGenerator {
	return &NeoPromptGenerator{
		Template: template,
		Code: code,
		FileName: fileName,
	}
}

func (npg *NeoPromptGenerator) String() string {
	return npg.Template
}

func (npg *NeoPromptGenerator) WithContent(content string) *NeoPromptGenerator {
	npg.Template = content
	return npg
}

type WeaviateHandler func(*dao.Weaviate, string, string) string

func (npg *NeoPromptGenerator) WithWeaviate(weaviate *dao.Weaviate, handler WeaviateHandler) *NeoPromptGenerator {
	npg.Template += handler(weaviate, npg.Code, npg.FileName)
	return npg
}

func (npg *NeoPromptGenerator) GeneratePrompt(code string, fileName string) string {
	prompt := npg.Template
	prompt = strings.ReplaceAll(prompt, "{code}", code)
	prompt = strings.ReplaceAll(prompt, "{fileName}", fileName)
	return prompt
}

func (npg *NeoPromptGenerator) WithCode(code, fileName string) *NeoPromptGenerator {
	npg.Template = strings.ReplaceAll(npg.Template, "{code}", code)
	npg.Template = strings.ReplaceAll(npg.Template, "{fileName}", fileName)
	return npg
}

func (npg *NeoPromptGenerator) WithString(str string) *NeoPromptGenerator {
	npg.Template += str
	return npg
}
