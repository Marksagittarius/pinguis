package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Marksagittarius/pinguis/dao"
	"github.com/Marksagittarius/pinguis/fileio"
	"github.com/Marksagittarius/pinguis/prompt"
	"github.com/Marksagittarius/pinguis/worker"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/schema"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
)

type ChatModelTest struct {
	model *ollama.ChatModel
}

func NewChatModelTest(ctx context.Context) *ChatModelTest {
	model, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: "http://localhost:11434",
		Model:   "qwen2.5-coder:7b",
	})

	if err != nil {
		panic(err)
	}
	return &ChatModelTest{
		model: model,
	}
}

func (c *ChatModelTest) Generate(ctx context.Context, prompt string) (*schema.Message, error) {
	return c.model.Generate(ctx, []*schema.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	})
}

func main() {
	rootPath := "./test"
	var pyFiles []string
	weaviate, err := dao.New(weaviate.Config{
		Host:   "localhost:8080",
		Scheme: "http",
	}, context.Background())

	if err != nil {
		panic(err)
	}

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".py") {
			if strings.HasSuffix(info.Name(), "_test.py") {
				return nil
			}
			if strings.Contains(info.Name(), "test_case") {
				return nil
			}
			pyFiles = append(pyFiles, path)
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	simpleFileIO := &fileio.SimpleFileIO{}
	promptTemplate, err := simpleFileIO.Read("./prompt.txt")

	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	model := NewChatModelTest(ctx)
	symWorker := worker.NewSymPromptWorker(&worker.DeepWorkerConfig{
		WorkerCount:       2,
		Model:             model,
		Callback:          worker.PyTestCallBack,
		CoverageThreshold: 0.8,
		MaxIterations:     3,
		SourcePath:        rootPath,
		TestPath:          rootPath,
		PromptGenerator: func(task *worker.TestTask) string {
			npg := prompt.NewNeoPromptGenerator(string(promptTemplate), task.SourceCode, task.SourcePath)
			basePrompt := npg.WithCode(task.SourceCode, task.SourcePath).WithWeaviate(weaviate, dao.FileInfoHandler).String()
			if task.Iterations == 0 {
				return basePrompt
			}

			basePrompt += "\n"
			basePrompt += "Your code need to be improved, the report is following:\n"
			basePrompt += task.TestReport
			basePrompt += "\n"

			return basePrompt
		},
	}, simpleFileIO)
	
	for _, pyFile := range pyFiles {
		if err := symWorker.SubmitSymTask(pyFile); err != nil {
			fmt.Printf("Unable to Submit %s: %v\n", pyFile, err)
			continue
		}
	}

	symWorker.Run()
	for symWorker.ActiveTaskCount() > 0 {
		time.Sleep(1 * time.Second)
		fmt.Printf("Tasks Remain: %d\n", symWorker.ActiveTaskCount())
	}

	fmt.Println("All Tasks Completed.")
	symWorker.Shutdown()
}
