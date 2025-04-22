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
	"time"

	"github.com/Marksagittarius/pinguis/model"
	"github.com/Marksagittarius/pinguis/postprocessor"
)

// TestTask represents a task for testing source code.
// It contains information about the source code, its path, testing iterations,
// coverage metrics, generated test code, test execution reports, and the programming language.
//
// Fields:
// - SourceCode: The source code to be tested.
// - SourcePath: The file path to the source code.
// - Iterations: The number of testing iterations completed so far.
// - BestCoverage: The highest code coverage rate achieved during testing.
// - GeneratedTest: The most recently generated test code (initially empty).
// - TestReport: The most recent test execution report (initially empty).
// - CodeType: The programming language of the source code (e.g., "go", "python").
type TestTask struct {
	SourceCode    string  // The source code to test
	SourcePath    string  // Path to the source file
	Iterations    int     // Number of iterations done so far
	BestCoverage  float64 // Best coverage rate achieved so far
	GeneratedTest string  // The latest generated test code (empty initially)
	TestReport    string  // The latest test execution report (empty initially)
	CodeType      string  // The programming language of the source code (e.g., "go", "python")
}

// TestCallback defines a function type that is used to execute a test and return its results.
// Parameters:
//   - sourceCode: The source code to be tested.
//   - testCode: The test code to be executed against the source code.
//   - sourcePath: The file path of the source code.
// Returns:
//   - coverage: A float64 value representing the code coverage achieved by the test.
//   - report: A string containing the test report or output.
//   - err: An error object if any issues occur during the test execution.
type TestCallback func(sourceCode, testCode, sourcePath string) (coverage float64, report string, err error)

// DeepWorker represents a worker that processes test tasks in a concurrent manner.
// It manages a pool of workers, handles task execution, and provides mechanisms
// for controlling task flow and lifecycle.
//
// Fields:
// - pool: The WorkerPool used to manage worker goroutines.
// - model: The ChatModel used for processing tasks.
// - tasks: A channel for queuing test tasks to be processed.
// - callback: A callback function invoked upon task completion.
// - coverageThreshold: The minimum coverage threshold required for task success.
// - maxIterations: The maximum number of iterations allowed for task processing.
// - wg: A WaitGroup to synchronize the completion of all tasks.
// - mu: A mutex to ensure thread-safe access to shared resources.
// - activeTasks: A map of currently active tasks, keyed by task ID.
// - ctx: A context for managing task cancellation and timeouts.
// - cancel: A function to cancel the context and stop task processing.
// - SourcePath: The file path to the source code being tested.
// - TestPath: The file path to the test cases.
// - PromptGenerator: A generator for creating task-specific prompts.
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
		tasks:             make(chan *TestTask, config.WorkerCount * 5),
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

// SubmitTask submits a new test task for processing. It ensures that no duplicate
// tasks are submitted for the same sourcePath and that the task queue has capacity
// to accept new tasks.
//
// Parameters:
//   - sourceCode: The source code to be tested.
//   - sourcePath: The file path of the source code.
//
// Returns:
//   - error: An error is returned if a task for the given sourcePath is already
//     being processed or if the task queue is full.
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

// Run starts the DeepWorker's main processing loop. It initializes the worker pool
// and launches a goroutine to process tasks from the task channel. Each task is
// submitted to the worker pool for execution. If a task submission fails, it attempts
// to requeue the task or marks it as complete after a timeout. The processing loop
// listens for tasks or a cancellation signal from the context to gracefully shut down.
// This method is non-blocking and logs the status of the worker and tasks.
func (dw *DeepWorker) Run() {
    dw.pool.Run()
    dw.wg.Add(1)

    go func() {
        defer dw.wg.Done()
        log.Println("Task processor started")
        
        for {
            select {
            case task, ok := <-dw.tasks:
                if !ok {
                    log.Println("Task channel closed, processor shutting down")
                    return
                }
                
                taskCopy := *task
                err := dw.pool.Submit(func() {
                    log.Printf("Processing task for: %s (iteration %d)", 
                        taskCopy.SourcePath, taskCopy.Iterations)
                    dw.processTask(&taskCopy)
                })
                
                if err != nil {
                    log.Printf("Failed to submit task for %s: %v", 
                        task.SourcePath, err)
                    
                    select {
                    case dw.tasks <- task:
                        log.Printf("Requeued failed task for: %s", task.SourcePath)
                    case <-time.After(3 * time.Second):
                        log.Printf("Failed to requeue task, marking as complete: %s", 
                            task.SourcePath)
                        dw.completeTask(task.SourcePath)
                    }
                }
                
            case <-dw.ctx.Done():
                log.Println("Context canceled, processor shutting down")
                return
            }
        }
    }()
    
    log.Println("DeepWorker is now running")
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

// processTask processes a given test generation task by generating test code,
// evaluating its coverage, and re-queuing the task if necessary based on the
// coverage threshold and iteration limits.
//
// Parameters:
//   - task (*TestTask): The test generation task to process.
//
// Behavior:
//   1. Builds a prompt for the task using the buildPrompt method.
//   2. Generates a response from the model using the prompt.
//   3. If generation fails, marks the task as complete and exits.
//   4. Extracts test code from the model's response and assigns it to the task.
//   5. Evaluates the test code's coverage and generates a test report.
//   6. Updates the task's best coverage if the new coverage is higher.
//   7. If the coverage is below the threshold and the iteration limit is not
//      reached, increments the iteration count and re-queues the task.
//   8. If the task is completed (either due to sufficient coverage or reaching
//      the iteration limit), logs the result and marks the task as complete.
//
// Notes:
//   - The method ensures tasks are not re-queued if the task queue is full.
//   - Logs relevant information about task completion and re-queuing failures.
func (dw *DeepWorker) processTask(task *TestTask) {
	prompt := dw.buildPrompt(task)

	msg, err := dw.model.Generate(dw.ctx, prompt)
	if err != nil {
		dw.completeTask(task.SourcePath)
		return
	}

	testCode := extractCodeFromMessage(msg.Content, task.CodeType)
	task.GeneratedTest = testCode

	coverage, report, err := dw.callback(task.SourceCode, testCode, processTestFilePath(task.SourcePath, task.CodeType))
	if err != nil {
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

func (dw *DeepWorker) buildPrompt(task *TestTask) string {
	return dw.PromptGenerator(task)
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
