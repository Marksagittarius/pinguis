package model

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// ChatModel represents an interface for generating chat messages based on a given prompt.
// The Generate method takes a context and a prompt string, and returns a generated message
// along with any error encountered during the generation process.
type ChatModel interface {
	Generate(ctx context.Context, prompt string) (*schema.Message, error)
}
