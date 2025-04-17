package java

import (
	"os"
	"path/filepath"

	"github.com/Marksagittarius/pinguis/types"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

// TreeSitterJavaParser is a struct that serves as a parser for Java code
// using the Tree-sitter parsing library. It provides functionality to
// analyze and manipulate Java syntax trees.
type TreeSitterJavaParser struct {

}

// NewTreeSitterJavaParser creates and returns a new instance of TreeSitterJavaParser.
// This function initializes the parser for processing Java code using the Tree-sitter library.
func NewTreeSitterJavaParser() *TreeSitterJavaParser {
	return &TreeSitterJavaParser{}
}

// ParseFile parses a Java source file located at the specified file path
// and returns a representation of the file as a *types.File object.
//
// Parameters:
//   - filePath: The path to the Java source file to be parsed.
//
// Returns:
//   - *types.File: A pointer to the parsed file representation.
//   - error: An error if the file cannot be read or parsed.
//
// This function uses the Tree-sitter library to parse the Java source code
// and analyze its syntax tree. If the file cannot be read or if there is
// an issue during parsing, an error is returned.
func (p *TreeSitterJavaParser) ParseFile(filePath string) (*types.File, error) {
	parser := tree_sitter.NewParser()
	parser.SetLanguage(tree_sitter.NewLanguage(tree_sitter_java.Language()))
	
	code, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	tree := parser.Parse(code, nil)
	rootNode := tree.RootNode()
	file := AnalyzeJavaFile(rootNode, code, filePath)
	return &file, nil
}

// ParseModule parses a Java module from the specified module path and returns
// a representation of the module as a *types.Module. If an error occurs during
// parsing, it will be returned.
//
// Parameters:
//   - modulePath: The file path to the Java module to be parsed.
//
// Returns:
//   - *types.Module: A pointer to the parsed module representation.
//   - error: An error object if parsing fails, otherwise nil.
func (p *TreeSitterJavaParser) ParseModule(modulePath string) (*types.Module, error) {
    return AnalyzeJavaModule(modulePath)
}

// getNodeText extracts and returns the text content of a given tree-sitter node
// from the provided source code.
//
// Parameters:
//   - node: A pointer to a tree-sitter Node whose text content is to be extracted.
//   - code: A byte slice representing the source code from which the node's text
//           will be extracted.
//
// Returns:
//   A string containing the text content of the specified node, derived from the
//   provided source code.
func getNodeText(node *tree_sitter.Node, code []byte) string {
    return string(code[node.StartByte():node.EndByte()])
}

// extractParameters extracts a list of parameters from a given tree-sitter node.
// It traverses the child nodes of the provided parameter node to identify formal
// parameters, extracting their names and types.
//
// Parameters:
//   - paramNode: A pointer to a tree-sitter Node representing the parameter list.
//   - code: A byte slice containing the source code being analyzed.
//
// Returns:
//   - A slice of types.Parameter, where each parameter contains its name and type.
//
// Notes:
//   - If paramNode is nil, an empty slice is returned.
//   - The function uses a tree-sitter cursor to iterate through the child nodes
//     of the parameter node.
func extractParameters(paramNode *tree_sitter.Node, code []byte) []types.Parameter {
    var params []types.Parameter
    if paramNode == nil {
        return params
    }

    cursor := paramNode.Walk()
    defer cursor.Close()

    if cursor.GotoFirstChild() {
        for {
            currentNode := cursor.Node()
            
            if currentNode.Kind() == "formal_parameter" {
                var paramName, paramType string
                
                typeNode := currentNode.ChildByFieldName("type")
                if typeNode != nil {
                    paramType = getNodeText(typeNode, code)
                }
                
                nameNode := currentNode.ChildByFieldName("name")
                if nameNode != nil {
                    paramName = getNodeText(nameNode, code)
                }
                
                params = append(params, types.Parameter{
                    Name: paramName,
                    Type: paramType,
                })
            }
            
            if !cursor.GotoNextSibling() {
                break
            }
        }
        cursor.GotoParent()
    }
    
    return params
}

// extractReturnType extracts the return type of a method from a tree-sitter Node.
// It first checks if the method node has a "type" field and retrieves its text.
// If the "type" field is not present, it traverses the child nodes of the method
// to check for a "void_type" node, returning "void" if found.
// If no return type is identified, it returns an empty string.
//
// Parameters:
//   - methodNode: A pointer to the tree-sitter Node representing the method.
//   - code: A byte slice containing the source code being analyzed.
//
// Returns:
//   A string representing the return type of the method, or an empty string
//   if no return type is found.
func extractReturnType(methodNode *tree_sitter.Node, code []byte) string {
    typeNode := methodNode.ChildByFieldName("type")
    if typeNode != nil {
        return getNodeText(typeNode, code)
    }
    
    cursor := methodNode.Walk()
    defer cursor.Close()
    
    if cursor.GotoFirstChild() {
        for {
            node := cursor.Node()
            if node.Kind() == "void_type" {
                return "void"
            }
            if !cursor.GotoNextSibling() {
                break
            }
        }
    }
    
    return ""
}

// extractMethods extracts method declarations from a given tree-sitter node
// representing a class or struct body and returns a slice of Method objects.
//
// Parameters:
//   - bodyNode: A pointer to a tree-sitter Node representing the body of a class or struct.
//   - code: A byte slice containing the source code being analyzed.
//
// Returns:
//   - A slice of types.Method, where each Method represents a method declaration
//     found within the provided bodyNode. Each Method includes details such as
//     the method name, parameters, return types, and body.
//
// Notes:
//   - If the bodyNode is nil, an empty slice is returned.
//   - The function uses a tree-sitter cursor to traverse the child nodes of the bodyNode
//     and identifies nodes of kind "method_declaration" to extract method details.
func extractMethods(bodyNode *tree_sitter.Node, code []byte) []types.Method {
    var methods []types.Method
    if bodyNode == nil {
        return methods
    }
    
    cursor := bodyNode.Walk()
    defer cursor.Close()
    
    if cursor.GotoFirstChild() {
        for {
            node := cursor.Node()
            
            if node.Kind() == "method_declaration" {
                var methodName, returnType string
                var parameters []types.Parameter
                
                nameNode := node.ChildByFieldName("name")
                if nameNode != nil {
                    methodName = getNodeText(nameNode, code)
                }
                
                returnType = extractReturnType(node, code)
                
                paramNode := node.ChildByFieldName("parameters")
                if paramNode != nil {
                    parameters = extractParameters(paramNode, code)
                }
                
                bodyNode := node.ChildByFieldName("body")
                body := ""
                if bodyNode != nil {
                    body = getNodeText(bodyNode, code)
                }
                
                method := types.Method{
                    Reciever: "", 
                    Func: types.Function{
                        Name:        methodName,
                        Parameters:  parameters,
                        ReturnTypes: []string{returnType},
                        Body:        body,
                    },
                }
                
                methods = append(methods, method)
            }
            
            if !cursor.GotoNextSibling() {
                break
            }
        }
    }
    
    return methods
}

// extractFields extracts a list of fields from the given bodyNode in a tree-sitter
// syntax tree. It traverses the node tree to identify "field_declaration" nodes,
// retrieves their types and names, and returns them as a slice of types.Field.
//
// Parameters:
// - bodyNode: A pointer to a tree-sitter Node representing the body of the code
//   to analyze. If nil, an empty slice is returned.
// - code: A byte slice containing the source code being analyzed.
//
// Returns:
// - A slice of types.Field, where each field contains the name and type of a
//   field declaration found in the bodyNode.
func extractFields(bodyNode *tree_sitter.Node, code []byte) []types.Field {
    var fields []types.Field
    if bodyNode == nil {
        return fields
    }
    
    cursor := bodyNode.Walk()
    defer cursor.Close()
    
    if cursor.GotoFirstChild() {
        for {
            node := cursor.Node()
            
            if node.Kind() == "field_declaration" {
                var fieldType string
                
                typeNode := node.ChildByFieldName("type")
                if typeNode != nil {
                    fieldType = getNodeText(typeNode, code)
                }
                
                declaratorCursor := node.Walk()
                defer declaratorCursor.Close()
                
                if declaratorCursor.GotoFirstChild() {
                    for {
                        declaratorNode := declaratorCursor.Node()
                        
                        if declaratorNode.Kind() == "variable_declarator" {
                            nameNode := declaratorNode.ChildByFieldName("name")
                            if nameNode != nil {
                                fieldName := getNodeText(nameNode, code)
                                fields = append(fields, types.Field{
                                    Name: fieldName,
                                    Type: fieldType,
                                })
                            }
                        }
                        
                        if !declaratorCursor.GotoNextSibling() {
                            break
                        }
                    }
                }
            }
            
            if !cursor.GotoNextSibling() {
                break
            }
        }
    }
    
    return fields
}

// extractInterfaceMethods extracts a list of methods from the body of an interface node.
// It traverses the child nodes of the provided bodyNode to identify method declarations,
// and for each method, it extracts the method name, return type, and parameters.
//
// Parameters:
//   - bodyNode: A pointer to a tree_sitter.Node representing the body of the interface.
//   - code: A byte slice containing the source code being analyzed.
//
// Returns:
//   - A slice of types.Function, where each Function represents a method with its name,
//     parameters, return types, and an empty body.
func extractInterfaceMethods(bodyNode *tree_sitter.Node, code []byte) []types.Function {
    var methods []types.Function
    if bodyNode == nil {
        return methods
    }
    
    cursor := bodyNode.Walk()
    defer cursor.Close()
    
    if cursor.GotoFirstChild() {
        for {
            node := cursor.Node()
            
            if node.Kind() == "method_declaration" {
                var methodName, returnType string
                var parameters []types.Parameter
                
                nameNode := node.ChildByFieldName("name")
                if nameNode != nil {
                    methodName = getNodeText(nameNode, code)
                }
                
                returnType = extractReturnType(node, code)
                
                paramNode := node.ChildByFieldName("parameters")
                if paramNode != nil {
                    parameters = extractParameters(paramNode, code)
                }
                
                methods = append(methods, types.Function{
                    Name:        methodName,
                    Parameters:  parameters,
                    ReturnTypes: []string{returnType},
                    Body:        "",
                })
            }
            
            if !cursor.GotoNextSibling() {
                break
            }
        }
    }
    
    return methods
}

// AnalyzeJavaFile analyzes a Java source file represented as a tree-sitter syntax tree
// and extracts its structural components such as classes, interfaces, and functions.
//
// Parameters:
//   - root: The root node of the tree-sitter syntax tree representing the Java file.
//   - code: The byte slice containing the source code of the Java file.
//   - filePath: The file path of the Java source file.
//
// Returns:
//   - A types.File object containing the extracted information, including:
//       - Path: The file path of the Java source file.
//       - Module: The package name of the Java file (if present).
//       - Classes: A slice of types.Class representing the classes in the file,
//         including their names, fields, and methods.
//       - Interfaces: A slice of types.Interface representing the interfaces in the file,
//         including their names and methods.
//       - Functions: A slice of types.Function representing standalone functions (if any).
func AnalyzeJavaFile(root *tree_sitter.Node, code []byte, filePath string) types.File {
    file := types.File{
        Path:    filePath,
        Classes: []types.Class{},
        Interfaces: []types.Interface{},
        Functions: []types.Function{},
    }
    
    cursor := root.Walk()
    defer cursor.Close()
    
    if cursor.GotoFirstChild() {
        for {
            node := cursor.Node()
            
            switch node.Kind() {
            case "package_declaration":
                nameNode := node.ChildByFieldName("name")
                if nameNode != nil {
                    file.Module = getNodeText(nameNode, code)
                }
                
            case "class_declaration":
                var className string
                var fields []types.Field
                var methods []types.Method
                
                nameNode := node.ChildByFieldName("name")
                if nameNode != nil {
                    className = getNodeText(nameNode, code)
                }
                
                bodyNode := node.ChildByFieldName("body")
                if bodyNode != nil {
                    fields = extractFields(bodyNode, code)
                    methods = extractMethods(bodyNode, code)
                }
                
                file.Classes = append(file.Classes, types.Class{
                    Name:    className,
                    Fields:  fields,
                    Methods: methods,
                })
                
            case "interface_declaration":
                var interfaceName string
                var methods []types.Function
                
                nameNode := node.ChildByFieldName("name")
                if nameNode != nil {
                    interfaceName = getNodeText(nameNode, code)
                }
                
                bodyNode := node.ChildByFieldName("body")
                if bodyNode != nil {
                    methods = extractInterfaceMethods(bodyNode, code)
                }
                
                file.Interfaces = append(file.Interfaces, types.Interface{
                    Name:    interfaceName,
                    Methods: methods,
                })
            }
            
            if !cursor.GotoNextSibling() {
                break
            }
        }
    }
    
    return file
}

// AnalyzeJavaModule analyzes a Java module located at the specified path and returns a structured representation of the module.
//
// This function recursively traverses the directory tree starting from the given modulePath. It identifies Java source files
// and submodules, parsing the Java files and organizing them into a hierarchical structure.
//
// Parameters:
//   - modulePath: The root directory path of the Java module to analyze.
//
// Returns:
//   - *types.Module: A pointer to the structured representation of the module, containing its name, files, and submodules.
//   - error: An error if any issues occur during the analysis, or nil if the analysis is successful.
//
// Behavior:
//   - Skips hidden directories (those starting with a dot).
//   - Parses files with the ".java" extension using a Tree-Sitter-based Java parser.
//   - Recursively analyzes subdirectories as submodules, except for the root directory itself.
//
// Example:
//   module, err := AnalyzeJavaModule("/path/to/java/module")
//   if err != nil {
//       log.Fatalf("Failed to analyze module: %v", err)
//   }
//   fmt.Printf("Module Name: %s\n", module.Name)
func AnalyzeJavaModule(modulePath string) (*types.Module, error) {
    module := &types.Module{
        Name:       filepath.Base(modulePath),
        Files:      []types.File{},
        SubModules: []types.Module{},
    }

    err := filepath.Walk(modulePath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if info.IsDir() && info.Name()[0] == '.' {
            return filepath.SkipDir
        }

        if !info.IsDir() && filepath.Ext(info.Name()) == ".java" {
            parser := NewTreeSitterJavaParser()
            file, err := parser.ParseFile(path)
            if err != nil {
                return err
            }
            module.Files = append(module.Files, *file)
        }

        if info.IsDir() && path != modulePath {
            subModule, err := AnalyzeJavaModule(path)
            if err != nil {
                return err
            }
            module.SubModules = append(module.SubModules, *subModule)
        }

        return nil
    })

    if err != nil {
        return &types.Module{}, err
    }

    return module, nil
}
