package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Marksagittarius/pinguis/model"
	"github.com/Marksagittarius/pinguis/postprocessor"
)

type TestTask struct {
	SourceCode    string  // The source code to test
	SourcePath    string  // Path to the source file
	Iterations    int     // Number of iterations done so far
	BestCoverage  float64 // Best coverage rate achieved so far
	GeneratedTest string  // The latest generated test code (empty initially)
	TestReport    string  // The latest test execution report (empty initially)
	CodeType      string  // The programming language of the source code (e.g., "go", "python")
}

type TestCallback func(sourceCode, testCode, sourcePath string) (coverage float64, report string, err error)

type DeepWorker struct {
	pool              WorkerPool
	model             model.ChatModel
	tasks             chan *TestTask
	callback          TestCallback
	coverageThreshold float64
	maxIterations     int
	wg                sync.WaitGroup
	mu                sync.Mutex
	activeTasks       map[string]*TestTask
	ctx               context.Context
	cancel            context.CancelFunc
	SourcePath        string
	TestPath          string
	PromptGenerator   TaskPromptGenerator
}

type DeepWorkerConfig struct {
	WorkerCount       int
	Model             model.ChatModel
	Callback          TestCallback
	CoverageThreshold float64
	MaxIterations     int
	SourcePath        string
	TestPath          string
	PromptGenerator   TaskPromptGenerator
}

func NewDeepWorker(config *DeepWorkerConfig) *DeepWorker {
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewGoWorkerPool(config.WorkerCount)

	return &DeepWorker{
		pool:              pool,
		model:             config.Model,
		tasks:             make(chan *TestTask, config.WorkerCount),
		callback:          config.Callback,
		coverageThreshold: config.CoverageThreshold,
		maxIterations:     config.MaxIterations,
		activeTasks:       make(map[string]*TestTask),
		ctx:               ctx,
		cancel:            cancel,
		SourcePath:        config.SourcePath,
		TestPath:          config.TestPath,
		PromptGenerator:   config.PromptGenerator,
	}
}

func getCodeType(sourcePath string) string {
	if strings.HasSuffix(sourcePath, ".go") {
		return "go"
	}
	if strings.HasSuffix(sourcePath, ".py") {
		return "python"
	}
	if strings.HasSuffix(sourcePath, ".js") {
		return "javascript"
	}
	if strings.HasSuffix(sourcePath, ".java") {
		return "java"
	}
	if strings.HasSuffix(sourcePath, ".cpp") {
		return "cpp"
	}
	return ""
}

func (dw *DeepWorker) SubmitTask(sourceCode, sourcePath string) error {
	dw.mu.Lock()
	defer dw.mu.Unlock()

	if _, exists := dw.activeTasks[sourcePath]; exists {
		return fmt.Errorf("already processing tests for %s", sourcePath)
	}

	task := &TestTask{
		SourceCode:   sourceCode,
		SourcePath:   sourcePath,
		Iterations:   0,
		BestCoverage: 0.0,
		CodeType:     getCodeType(sourcePath),
		TestReport:   "",
	}

	dw.activeTasks[sourcePath] = task

	select {
	case dw.tasks <- task:
		return nil
	default:
		delete(dw.activeTasks, sourcePath)
		return fmt.Errorf("task queue is full")
	}
}

func (dw *DeepWorker) Run() {
	dw.pool.Run()

	dw.wg.Add(1)
	go func() {
		defer dw.wg.Done()

		for {
			select {
			case task := <-dw.tasks:
				err := dw.pool.Submit(func() {
					dw.processTask(task)
				})

				if err != nil {
					log.Printf("Failed to submit task: %v", err)
				}

			case <-dw.ctx.Done():
				return
			}
		}
	}()
}

func processTestFilePath(sourcePath, codeType string) string {
	if codeType == "go" {
		return strings.Replace(sourcePath, ".go", "_test.go", 1)
	}
	if codeType == "python" {
		return strings.Replace(sourcePath, ".py", "_test.py", 1)
	}
	if codeType == "javascript" {
		return strings.Replace(sourcePath, ".js", "_test.js", 1)
	}
	if codeType == "java" {
		return strings.Replace(sourcePath, ".java", "Test"+sourcePath, 1)
	}
	return sourcePath
}

func (dw *DeepWorker) processTask(task *TestTask) {
	prompt := dw.buildPrompt(task, dw.PromptGenerator)

	msg, err := dw.model.Generate(dw.ctx, prompt)
	if err != nil {
		log.Printf("Failed to generate test for %s (iteration %d): %v", task.SourcePath, task.Iterations, err)
		dw.completeTask(task.SourcePath)
		return
	}

	testCode := extractCodeFromMessage(msg.Content, task.CodeType)
	task.GeneratedTest = testCode

	coverage, report, err := dw.callback(task.SourceCode, testCode, processTestFilePath(task.SourcePath, task.CodeType))
	if err != nil {
		log.Printf("Failed to run tests for %s (iteration %d): %v", task.SourcePath, task.Iterations, err)
		dw.completeTask(task.SourcePath)
		return
	}

	task.TestReport = report

	if coverage > task.BestCoverage {
		task.BestCoverage = coverage
	}

	if coverage < dw.coverageThreshold && task.Iterations < dw.maxIterations {
		task.Iterations++

		select {
		case dw.tasks <- task:
		default:
			log.Printf("Failed to re-queue task for %s: queue full", task.SourcePath)
			dw.completeTask(task.SourcePath)
		}
	} else {
		log.Printf("Completed test generation for %s after %d iterations with %.2f%% coverage",
			task.SourcePath, task.Iterations, task.BestCoverage*100)
		dw.completeTask(task.SourcePath)
	}
}

type TaskPromptGenerator func(*TestTask) string

func (dw *DeepWorker) buildPrompt(task *TestTask, generator TaskPromptGenerator) string {
	currentPrompt := generator(task)
	return currentPrompt
}

func (dw *DeepWorker) completeTask(sourcePath string) {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	delete(dw.activeTasks, sourcePath)
}

func (dw *DeepWorker) Shutdown() {
	dw.cancel()
	dw.wg.Wait()
	dw.pool.Shutdown()
}

func (dw *DeepWorker) ActiveTaskCount() int {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	return len(dw.activeTasks)
}

func (dw *DeepWorker) GetTaskStatus(sourcePath string) (*TestTask, bool) {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	task, exists := dw.activeTasks[sourcePath]
	return task, exists
}

func extractCodeFromMessage(content, codeType string) string {
	ce := postprocessor.NewCodeExtractor(codeType)
	return ce.Postprocess(content)
}

func PyTestCallBack(sourceCode, testCode, sourcePath string) (float64, string, error) {
    testDir := filepath.Dir(sourcePath)
    
    if err := os.WriteFile(sourcePath, []byte(testCode), 0644); err != nil {
        return 0, "", fmt.Errorf("failed to write test file to %s: %v", sourcePath, err)
    }
    
    cmd := exec.Command("coverage", "run", "--source=.", filepath.Base(sourcePath))
    cmd.Dir = testDir
    
    testOutput, err := cmd.CombinedOutput()
    testReport := string(testOutput)
    
    if err != nil {
        return 0, testReport, fmt.Errorf("coverage run failed: %v", err)
    }
    
    reportCmd := exec.Command("coverage", "report")
    reportCmd.Dir = testDir
    
    reportOutput, err := reportCmd.CombinedOutput()
	if err != nil {
		return 0, "", fmt.Errorf("coverage report failed: %v", err)
	}
    coverageReport := string(reportOutput)
    
    fullReport := testReport + "\n" + coverageReport
    
    return 0, fullReport, nil
}
