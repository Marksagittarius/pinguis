package dependency

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Marksagittarius/pinguis/dao"
	"github.com/Marksagittarius/pinguis/filetree"
	"github.com/Marksagittarius/pinguis/scripts/java"
	"github.com/Marksagittarius/pinguis/scripts/python"
	"github.com/Marksagittarius/pinguis/types"

	"github.com/weaviate/weaviate-go-client/v5/weaviate"
)

// Constants for dependency types
const (
	ImportDependency     = "import"
	ExtendsDependency    = "extends"
	ImplementsDependency = "implements"
	UsesDependency       = "uses"
	ReferencesDependency = "references"
)

// DependencyType represents the type of dependency between files or code elements
type DependencyType string

// Dependency represents a dependency relationship between code elements
type Dependency struct {
	SourceFile    string         `json:"source_file"`
	TargetFile    string         `json:"target_file"`
	Type          DependencyType `json:"type"`
	SourceElement string         `json:"source_element,omitempty"`
	TargetElement string         `json:"target_element,omitempty"`
	Weight        float64        `json:"weight"`
}

// DependencyGraph represents a graph of dependencies between files
type DependencyGraph struct {
	Dependencies []Dependency                  `json:"dependencies"`
	FileNodes    map[string]*filetree.FileNode `json:"file_nodes"`
}

// DependencyAnalyzer is the interface that defines dependency analysis operations
type DependencyAnalyzer interface {
	AnalyzeFile(filePath string) ([]Dependency, error)
	AnalyzeDirectory(dirPath string) (*DependencyGraph, error)
	GetDependencies(filePath string) ([]Dependency, error)
	GetDependents(filePath string) ([]Dependency, error)
}

// AnalyzerFactory creates appropriate analyzers based on file type
type AnalyzerFactory interface {
	CreateAnalyzer(filePath string) (DependencyAnalyzer, error)
}

// LanguageSpecificAnalyzer provides common functionality for language-specific analyzers
type LanguageSpecificAnalyzer struct {
	Cache    *DependencyCache
	FileTree *filetree.FileTree
}

// DependencyCache caches the results of dependency analysis
type DependencyCache struct {
	weaviateClient *dao.Weaviate
	cachedDeps     map[string][]Dependency
	mutex          sync.RWMutex
}

// NewDependencyCache creates a new dependency cache
func NewDependencyCache(weaviateConfig weaviate.Config) (*DependencyCache, error) {
	weaviateClient, err := dao.New(weaviateConfig, context.Background())
	if err != nil {
		return nil, err
	}

	return &DependencyCache{
		weaviateClient: weaviateClient,
		cachedDeps:     make(map[string][]Dependency),
	}, nil
}

// Get returns cached dependencies for a file
func (dc *DependencyCache) Get(filePath string) ([]Dependency, bool) {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()

	deps, ok := dc.cachedDeps[filePath]
	return deps, ok
}

// Store caches dependencies for a file
func (dc *DependencyCache) Store(filePath string, deps []Dependency) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	dc.cachedDeps[filePath] = deps
}

// DefaultAnalyzerFactory creates language-specific analyzers based on file extension
type DefaultAnalyzerFactory struct {
	Cache    *DependencyCache
	FileTree *filetree.FileTree
}

// NewDefaultAnalyzerFactory creates a new analyzer factory
func NewDefaultAnalyzerFactory(cache *DependencyCache, fileTree *filetree.FileTree) *DefaultAnalyzerFactory {
	return &DefaultAnalyzerFactory{
		Cache:    cache,
		FileTree: fileTree,
	}
}

// CreateAnalyzer creates an appropriate analyzer for the given file
func (f *DefaultAnalyzerFactory) CreateAnalyzer(filePath string) (DependencyAnalyzer, error) {
	ext := filepath.Ext(filePath)

	switch strings.ToLower(ext) {
	case ".java":
		return &JavaDependencyAnalyzer{
			LanguageSpecificAnalyzer: LanguageSpecificAnalyzer{
				Cache:    f.Cache,
				FileTree: f.FileTree,
			},
			Parser: java.NewTreeSitterJavaParser(),
		}, nil
	case ".py":
		return &PythonDependencyAnalyzer{
			LanguageSpecificAnalyzer: LanguageSpecificAnalyzer{
				Cache:    f.Cache,
				FileTree: f.FileTree,
			},
		}, nil
	case ".go":
		return &GoDependencyAnalyzer{
			LanguageSpecificAnalyzer: LanguageSpecificAnalyzer{
				Cache:    f.Cache,
				FileTree: f.FileTree,
			},
		}, nil
	default:
		return &GenericDependencyAnalyzer{
			LanguageSpecificAnalyzer: LanguageSpecificAnalyzer{
				Cache:    f.Cache,
				FileTree: f.FileTree,
			},
		}, nil
	}
}

// JavaDependencyAnalyzer analyzes dependencies in Java files
type JavaDependencyAnalyzer struct {
	LanguageSpecificAnalyzer
	Parser java.JavaParser
}

// AnalyzeFile analyzes dependencies in a Java file
func (a *JavaDependencyAnalyzer) AnalyzeFile(filePath string) ([]Dependency, error) {
	// Check cache first
	if deps, found := a.Cache.Get(filePath); found {
		return deps, nil
	}

	file, err := a.Parser.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Java file %s: %v", filePath, err)
	}

	dependencies := a.extractJavaDependencies(file)

	// Cache the results
	a.Cache.Store(filePath, dependencies)

	return dependencies, nil
}

// extractJavaDependencies extracts dependencies from a Java file model
func (a *JavaDependencyAnalyzer) extractJavaDependencies(file *types.File) []Dependency {
	var dependencies []Dependency

	// Extract import dependencies
	// Implementation would analyze imports, extends, implements relationships
	// And track which classes/methods are used within the file

	return dependencies
}

// AnalyzeDirectory analyzes dependencies in a directory
func (a *JavaDependencyAnalyzer) AnalyzeDirectory(dirPath string) (*DependencyGraph, error) {
	return analyzeDirectory(dirPath, a)
}

// GetDependencies returns dependencies for a file
func (a *JavaDependencyAnalyzer) GetDependencies(filePath string) ([]Dependency, error) {
	return a.AnalyzeFile(filePath)
}

// GetDependents returns files that depend on the given file
func (a *JavaDependencyAnalyzer) GetDependents(filePath string) ([]Dependency, error) {
	// This would require having analyzed the entire project first
	// Then filtering dependencies where TargetFile matches filePath
	return nil, fmt.Errorf("not implemented")
}

// PythonDependencyAnalyzer analyzes dependencies in Python files
type PythonDependencyAnalyzer struct {
	LanguageSpecificAnalyzer
}

// AnalyzeFile analyzes dependencies in a Python file
func (a *PythonDependencyAnalyzer) AnalyzeFile(filePath string) ([]Dependency, error) {
	// Check cache first
	if deps, found := a.Cache.Get(filePath); found {
		return deps, nil
	}

	file, err := python.GetFileMetaData(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Python file %s: %v", filePath, err)
	}

	dependencies := a.extractPythonDependencies(file)

	// Cache the results
	a.Cache.Store(filePath, dependencies)

	return dependencies, nil
}

// extractPythonDependencies extracts dependencies from a Python file model
func (a *PythonDependencyAnalyzer) extractPythonDependencies(file *types.File) []Dependency {
	var dependencies []Dependency

	// Analyze each function body for imports and function calls
	for _, function := range file.Functions {
		// Process import statements
		importDeps := a.extractImportsFromBody(file.Path, function.Body, function.Name)
		dependencies = append(dependencies, importDeps...)

		// Process function calls to identify dependencies on other modules
		callDeps := a.extractFunctionCallsFromBody(file.Path, function.Body, function.Name)
		dependencies = append(dependencies, callDeps...)
	}

	// Process class inheritance and method calls
	for _, class := range file.Classes {
		// Extract inheritance dependencies
		inheritDeps := a.extractClassInheritance(file.Path, class)
		dependencies = append(dependencies, inheritDeps...)

		// Extract method dependencies
		for _, method := range class.Methods {
			methodDeps := a.extractImportsFromBody(file.Path, method.Func.Body, method.Func.Name)
			dependencies = append(dependencies, methodDeps...)

			callDeps := a.extractFunctionCallsFromBody(file.Path, method.Func.Body, method.Func.Name)
			dependencies = append(dependencies, callDeps...)
		}
	}

	return dependencies
}

// extractImportsFromBody extracts import statements from a function or method body
func (a *PythonDependencyAnalyzer) extractImportsFromBody(sourceFilePath string, body string, sourceElement string) []Dependency {
	var dependencies []Dependency
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Handle "from X import Y" style imports
		if strings.HasPrefix(line, "from") {
			parts := strings.Fields(line)
			if len(parts) >= 4 && parts[2] == "import" {
				moduleName := parts[1]
				// Create dependency for the imported module
				targetFilePath := a.resolveModulePath(sourceFilePath, moduleName)

				// Extract imported elements
				importedElements := strings.Join(parts[3:], " ")
				importedElements = strings.ReplaceAll(importedElements, ",", " ")
				elements := strings.Fields(importedElements)

				for _, element := range elements {
					// Remove trailing comma if present
					element = strings.TrimSuffix(element, ",")

					dependencies = append(dependencies, Dependency{
						SourceFile:    sourceFilePath,
						TargetFile:    targetFilePath,
						Type:          DependencyType(ImportDependency),
						SourceElement: sourceElement,
						TargetElement: element,
						Weight:        1.0,
					})
				}
			}
		}

		// Handle direct "import X" style imports
		if strings.HasPrefix(line, "import") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				for i := 1; i < len(parts); i++ {
					moduleName := strings.TrimRight(parts[i], ",")
					targetFilePath := a.resolveModulePath(sourceFilePath, moduleName)

					dependencies = append(dependencies, Dependency{
						SourceFile:    sourceFilePath,
						TargetFile:    targetFilePath,
						Type:          DependencyType(ImportDependency),
						SourceElement: sourceElement,
						Weight:        1.0,
					})
				}
			}
		}
	}

	return dependencies
}

// extractFunctionCallsFromBody extracts function calls from a function or method body
func (a *PythonDependencyAnalyzer) extractFunctionCallsFromBody(sourceFilePath string, body string, sourceElement string) []Dependency {
	var dependencies []Dependency

	// Get directory containing the source file
	sourceDir := filepath.Dir(sourceFilePath)

	// Get all Python files in the same directory to check for function calls
	files, err := filepath.Glob(filepath.Join(sourceDir, "*.py"))
	if err != nil {
		return dependencies
	}

	for _, targetFilePath := range files {
		// Skip self-references
		if targetFilePath == sourceFilePath {
			continue
		}

		// Parse the target file to get its functions and classes
		targetFile, err := python.GetFileMetaData(targetFilePath)
		if err != nil {
			continue
		}

		// Check for calls to functions from the target file
		for _, function := range targetFile.Functions {
			// Simple detection: look for function_name( pattern
			pattern := function.Name + "("
			if strings.Contains(body, pattern) {
				dependencies = append(dependencies, Dependency{
					SourceFile:    sourceFilePath,
					TargetFile:    targetFilePath,
					Type:          DependencyType(UsesDependency),
					SourceElement: sourceElement,
					TargetElement: function.Name,
					Weight:        0.7, // Lower weight for usage vs import
				})
			}
		}

		// Check for calls to class methods from the target file
		for _, class := range targetFile.Classes {
			for _, method := range class.Methods {
				// Two patterns: either direct call to method or through an instance
				classMethodPattern := class.Name + "." + method.Func.Name + "("
				instanceMethodPattern := "." + method.Func.Name + "("

				if strings.Contains(body, classMethodPattern) || strings.Contains(body, instanceMethodPattern) {
					dependencies = append(dependencies, Dependency{
						SourceFile:    sourceFilePath,
						TargetFile:    targetFilePath,
						Type:          DependencyType(UsesDependency),
						SourceElement: sourceElement,
						TargetElement: class.Name + "." + method.Func.Name,
						Weight:        0.7, // Lower weight for usage vs import
					})
				}
			}
		}
	}

	return dependencies
}

// extractClassInheritance extracts class inheritance dependencies
func (a *PythonDependencyAnalyzer) extractClassInheritance(sourceFilePath string, class types.Class) []Dependency {
	var dependencies []Dependency

	// In a real implementation, we would need to parse the class definition to extract base classes
	// For simplicity, we'll just search for potential inheritance references in the source file

	sourceDir := filepath.Dir(sourceFilePath)

	// Get all Python files in the same directory
	files, err := filepath.Glob(filepath.Join(sourceDir, "*.py"))
	if err != nil {
		return dependencies
	}

	for _, targetFilePath := range files {
		// Skip self-references
		if targetFilePath == sourceFilePath {
			continue
		}

		// Parse the target file to get its classes
		targetFile, err := python.GetFileMetaData(targetFilePath)
		if err != nil {
			continue
		}

		// Check if any classes in the target file might be base classes
		for _, targetClass := range targetFile.Classes {
			// In real code, we'd check class definition for parent classes
			// Here we use a simplified approach to check for potential parent classes

			// Check for class inheritance patterns like "class MyClass(ParentClass):"
			if class.Name != targetClass.Name && strings.Contains(sourceFilePath, targetClass.Name) {
				dependencies = append(dependencies, Dependency{
					SourceFile:    sourceFilePath,
					TargetFile:    targetFilePath,
					Type:          DependencyType(ExtendsDependency),
					SourceElement: class.Name,
					TargetElement: targetClass.Name,
					Weight:        0.9, // High weight for inheritance
				})
			}
		}
	}

	return dependencies
}

// resolveModulePath resolves a Python module name to a file path
func (a *PythonDependencyAnalyzer) resolveModulePath(sourceFilePath string, moduleName string) string {
	// First, check if the module is in the same directory
	sourceDir := filepath.Dir(sourceFilePath)
	candidatePath := filepath.Join(sourceDir, moduleName+".py")

	if _, err := os.Stat(candidatePath); err == nil {
		return candidatePath
	}

	// If not found, look for modules in the root of the project
	projectRoot := filepath.Dir(sourceDir)
	candidatePath = filepath.Join(projectRoot, moduleName+".py")

	if _, err := os.Stat(candidatePath); err == nil {
		return candidatePath
	}

	// If we still can't find it, just use the module name as is for reference
	// In a real implementation, we might want to handle Python's import system more thoroughly
	return moduleName + ".py"
}

// AnalyzeDirectory analyzes dependencies in a directory
func (a *PythonDependencyAnalyzer) AnalyzeDirectory(dirPath string) (*DependencyGraph, error) {
	return analyzeDirectory(dirPath, a)
}

// GetDependencies returns dependencies for a file
func (a *PythonDependencyAnalyzer) GetDependencies(filePath string) ([]Dependency, error) {
	return a.AnalyzeFile(filePath)
}

// GetDependents returns files that depend on the given file
func (a *PythonDependencyAnalyzer) GetDependents(filePath string) ([]Dependency, error) {
	// This would require having analyzed the entire project first
	// Then filtering dependencies where TargetFile matches filePath
	return nil, fmt.Errorf("not implemented")
}

// GoDependencyAnalyzer analyzes dependencies in Go files
type GoDependencyAnalyzer struct {
	LanguageSpecificAnalyzer
}

// AnalyzeFile analyzes dependencies in a Go file
func (a *GoDependencyAnalyzer) AnalyzeFile(filePath string) ([]Dependency, error) {
	// Go-specific dependency analysis would go here
	// This would parse imports and track which packages/types are used
	return nil, fmt.Errorf("not implemented")
}

// AnalyzeDirectory analyzes dependencies in a directory
func (a *GoDependencyAnalyzer) AnalyzeDirectory(dirPath string) (*DependencyGraph, error) {
	return analyzeDirectory(dirPath, a)
}

// GetDependencies returns dependencies for a file
func (a *GoDependencyAnalyzer) GetDependencies(filePath string) ([]Dependency, error) {
	return a.AnalyzeFile(filePath)
}

// GetDependents returns files that depend on the given file
func (a *GoDependencyAnalyzer) GetDependents(filePath string) ([]Dependency, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenericDependencyAnalyzer provides basic dependency analysis for unsupported file types
type GenericDependencyAnalyzer struct {
	LanguageSpecificAnalyzer
}

// AnalyzeFile analyzes dependencies in a generic file
func (a *GenericDependencyAnalyzer) AnalyzeFile(filePath string) ([]Dependency, error) {
	// Generic dependency analysis based on string matching would go here
	return nil, nil
}

// AnalyzeDirectory analyzes dependencies in a directory
func (a *GenericDependencyAnalyzer) AnalyzeDirectory(dirPath string) (*DependencyGraph, error) {
	return analyzeDirectory(dirPath, a)
}

// GetDependencies returns dependencies for a file
func (a *GenericDependencyAnalyzer) GetDependencies(filePath string) ([]Dependency, error) {
	return a.AnalyzeFile(filePath)
}

// GetDependents returns files that depend on the given file
func (a *GenericDependencyAnalyzer) GetDependents(filePath string) ([]Dependency, error) {
	return nil, nil
}

// analyzeDirectory is a helper function to analyze all files in a directory
func analyzeDirectory(dirPath string, analyzer DependencyAnalyzer) (*DependencyGraph, error) {
	treeBuilder := &filetree.FileTreeBuilder{}
	tree, err := treeBuilder.BuildTree(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build file tree for %s: %v", dirPath, err)
	}

	graph := &DependencyGraph{
		Dependencies: []Dependency{},
		FileNodes:    make(map[string]*filetree.FileNode),
	}

	// Collect all files
	var filePaths []string
	collectFiles(tree.Root, dirPath, &filePaths, graph.FileNodes)

	// Analyze each file
	var allDeps []Dependency
	for _, filePath := range filePaths {
		deps, err := analyzer.AnalyzeFile(filePath)
		if err != nil {
			// Log the error but continue with other files
			fmt.Printf("Error analyzing %s: %v\n", filePath, err)
			continue
		}
		allDeps = append(allDeps, deps...)
	}

	// Update the file tree with dependencies
	for _, dep := range allDeps {
		sourceNode, sourceOk := graph.FileNodes[dep.SourceFile]
		targetNode, targetOk := graph.FileNodes[dep.TargetFile]

		if sourceOk && targetOk {
			sourceNode.AddDependency(targetNode)
		}
	}

	graph.Dependencies = allDeps
	return graph, nil
}

// collectFiles recursively collects file paths from a file tree
func collectFiles(node *filetree.FileNode, basePath string, filePaths *[]string, nodeMap map[string]*filetree.FileNode) {
	path := filepath.Join(basePath, node.FileName)

	if node.FileType != "dir" {
		*filePaths = append(*filePaths, path)
	}

	nodeMap[path] = node

	for _, child := range node.Children {
		childPath := filepath.Join(basePath, node.FileName)
		collectFiles(child, childPath, filePaths, nodeMap)
	}
}

// DependencyAnalysisManager manages the dependency analysis process
type DependencyAnalysisManager struct {
	AnalyzerFactory AnalyzerFactory
	Cache           *DependencyCache
	FileTree        *filetree.FileTree
}

// NewDependencyAnalysisManager creates a new dependency analysis manager
func NewDependencyAnalysisManager(weaviateConfig weaviate.Config, rootPath string) (*DependencyAnalysisManager, error) {
	cache, err := NewDependencyCache(weaviateConfig)
	if err != nil {
		return nil, err
	}

	treeBuilder := &filetree.FileTreeBuilder{}
	tree, err := treeBuilder.BuildTree(rootPath)
	if err != nil {
		return nil, err
	}

	factory := NewDefaultAnalyzerFactory(cache, tree)

	return &DependencyAnalysisManager{
		AnalyzerFactory: factory,
		Cache:           cache,
		FileTree:        tree,
	}, nil
}

// AnalyzeFile analyzes dependencies for a single file
func (m *DependencyAnalysisManager) AnalyzeFile(filePath string) ([]Dependency, error) {
	analyzer, err := m.AnalyzerFactory.CreateAnalyzer(filePath)
	if err != nil {
		return nil, err
	}

	return analyzer.AnalyzeFile(filePath)
}

// AnalyzeProject analyzes dependencies for an entire project
func (m *DependencyAnalysisManager) AnalyzeProject(projectPath string) (*DependencyGraph, error) {
	// Use a generic analyzer for the project directory
	analyzer, err := m.AnalyzerFactory.CreateAnalyzer("")
	if err != nil {
		return nil, err
	}

	return analyzer.AnalyzeDirectory(projectPath)
}

// GetFileDependencies gets all dependencies for a specific file
func (m *DependencyAnalysisManager) GetFileDependencies(filePath string) ([]Dependency, error) {
	analyzer, err := m.AnalyzerFactory.CreateAnalyzer(filePath)
	if err != nil {
		return nil, err
	}

	return analyzer.GetDependencies(filePath)
}

// GetFileDependents gets all files that depend on a specific file
func (m *DependencyAnalysisManager) GetFileDependents(filePath string) ([]Dependency, error) {
	analyzer, err := m.AnalyzerFactory.CreateAnalyzer(filePath)
	if err != nil {
		return nil, err
	}

	return analyzer.GetDependents(filePath)
}
