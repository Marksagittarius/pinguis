package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

type SymPromptWorker struct {
	*DeepWorker
	fileIO FileIO
}

type FileIO interface {
	Read(filePath string) ([]byte, error)
	Write(filePath string, data []byte) error
}

func NewSymPromptWorker(config *DeepWorkerConfig, fileIO FileIO) *SymPromptWorker {
	return &SymPromptWorker{
		DeepWorker: NewDeepWorker(config),
		fileIO:     fileIO,
	}
}

func (sw *SymPromptWorker) SubmitSymTask(sourcePath string) error {
	codeBytes, err := sw.fileIO.Read(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read code: %w", err)
	}
	code := string(codeBytes)

	parser := tree_sitter.NewParser()
	parser.SetLanguage(tree_sitter.NewLanguage(tree_sitter_python.Language()))
	tree := parser.Parse([]byte(code), nil)
	root := tree.RootNode()

	var funcNodes []*tree_sitter.Node
	var funcNames []string
	var collectFuncs func(node *tree_sitter.Node)
	collectFuncs = func(node *tree_sitter.Node) {
		if node == nil {
			return
		}
		if node.Kind() == "function_definition" {
			funcNodes = append(funcNodes, node)
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				funcNames = append(funcNames, string(code[nameNode.StartByte():nameNode.EndByte()]))
			} else {
				funcNames = append(funcNames, "unknown")
			}
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			collectFuncs(node.NamedChild(uint(i)))
		}
	}
	collectFuncs(root)

	promptTemplateBytes, err := os.ReadFile("prompt.txt")
	if err != nil {
		return fmt.Errorf("failed to read prompt template: %w", err)
	}
	promptTemplate := string(promptTemplateBytes)

	for idx, fn := range funcNodes {
		var paths [][]string
		bodyNode := fn.ChildByFieldName("body")
		CollectPathsPython(bodyNode, func(n *tree_sitter.Node) string {
			return string(code[n.StartByte():n.EndByte()])
		}, []string{}, &paths)
		minPaths := MinimizePaths(paths)

		nameNode := fn.ChildByFieldName("name")
		funcName := "unknown"
		if nameNode != nil {
			funcName = string(code[nameNode.StartByte():nameNode.EndByte()])
		}
		parametersNode := fn.ChildByFieldName("parameters")
		params := ""
		if parametersNode != nil {
			params = string(code[parametersNode.StartByte():parametersNode.EndByte()])
		}
		returns := ""
		retNode := fn.ChildByFieldName("return_type")
		if retNode != nil {
			returns = string(code[retNode.StartByte():retNode.EndByte()])
		}

		pathDescs := []string{}
		for i, p := range minPaths {
			conds := []string{}
			retVal := ""
			for j, kind := range p {
				if strings.HasPrefix(kind, "if:") {
					condExpr := strings.TrimPrefix(kind, "if:")
					if j+1 < len(p) && strings.HasSuffix(p[j+1], "-else") {
						conds = append(conds, "not("+condExpr+")")
					} else {
						conds = append(conds, condExpr)
					}
				}
				if strings.HasPrefix(kind, "elif:") {
					condExpr := strings.TrimPrefix(kind, "elif:")
					if j+1 < len(p) && strings.HasSuffix(p[j+1], "-else") {
						conds = append(conds, "not("+condExpr+")")
					} else {
						conds = append(conds, condExpr)
					}
				}
				if strings.HasPrefix(kind, "return") {
					retVal = strings.TrimSpace(strings.TrimPrefix(kind, "return"))
				}
			}
			desc := fmt.Sprintf("Testcase %d for %s%s%s:\n", i+1, funcName, params, funcReturnTypeStr(returns))
			if len(conds) > 0 {
				desc += "test case where " + conds[0] + ",\n"
				for k := 1; k < len(conds); k++ {
					desc += "and " + conds[k] + "\n"
				}
			}
			if retVal != "" {
				desc += "returns '" + retVal + "'"
			}
			pathDescs = append(pathDescs, desc)
		}
		promptStr := promptTemplate
		promptStr = strings.ReplaceAll(promptStr, "{path_constraints}", strings.Join(pathDescs, "\n"))
		promptStr = strings.ReplaceAll(promptStr, "{code}", code)
		promptStr = strings.ReplaceAll(promptStr, "{file_name}", sourcePath)

		msg, err := sw.model.Generate(context.Background(), promptStr)
		if err != nil {
			return fmt.Errorf("LLM generate failed: %w", err)
		}
		testCode := extractCodeFromMessage(msg.Content, "python")

		dir := filepath.Dir(sourcePath)
		testFileName := symTestFileName(sourcePath, funcNames[idx], 0)
		testFilePath := filepath.Join(dir, testFileName)
		if err := sw.fileIO.Write(testFilePath, []byte(testCode)); err != nil {
			return fmt.Errorf("failed to write test file: %w", err)
		}

		if sw.callback != nil {
			sw.callback(code, testCode, testFilePath)
		}
	}
	return nil
}

func symTestFileName(sourcePath string, funcName string, idx int) string {
	base := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	return fmt.Sprintf("%s_%s_test_case_%d.py", base, funcName, idx+1)
}

func funcReturnTypeStr(returns string) string {
	if returns == "" {
		return ""
	}
	return " -> " + returns
}

func CollectPathsJava(node *tree_sitter.Node, getNodeText func(*tree_sitter.Node) string, cur []string, paths *[][]string) {
	if node == nil {
		return
	}
	kind := node.Kind()
	cur = append(cur, kind)

	switch kind {
	case "if_statement":
		condNode := node.ChildByFieldName("condition")
		cond := "if"
		if condNode != nil {
			cond += ":" + getNodeText(condNode)
		}
		thenNode := node.ChildByFieldName("consequence")
		thenPath := append(cur, cond+"-then")
		CollectPathsJava(thenNode, getNodeText, thenPath, paths)
		elseNode := node.ChildByFieldName("alternative")
		if elseNode != nil {
			elsePath := append(cur, cond+"-else")
			CollectPathsJava(elseNode, getNodeText, elsePath, paths)
		}
		return
	case "for_statement", "while_statement":
		loopPath := append(cur, kind)
		bodyNode := node.ChildByFieldName("body")
		CollectPathsJava(bodyNode, getNodeText, loopPath, paths)
		return
	case "switch_expression", "switch_statement":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			c := node.NamedChild(uint(i))
			if c.Kind() == "switch_block" || c.Kind() == "switch_block_statement_group" {
				CollectPathsJava(c, getNodeText, append(cur, "switch-case"), paths)
			}
		}
		return
	case "try_statement":
		tryBlock := node.ChildByFieldName("body")
		CollectPathsJava(tryBlock, getNodeText, append(cur, "try"), paths)
		for i := 0; i < int(node.NamedChildCount()); i++ {
			c := node.NamedChild(uint(i))
			if c.Kind() == "catch_clause" {
				CollectPathsJava(c, getNodeText, append(cur, "catch"), paths)
			}
		}
		finallyNode := node.ChildByFieldName("finally_block")
		if finallyNode != nil {
			CollectPathsJava(finallyNode, getNodeText, append(cur, "finally"), paths)
		}
		return
	}

	if node.NamedChildCount() == 0 {
		*paths = append(*paths, cur)
		return
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		CollectPathsJava(node.NamedChild(uint(i)), getNodeText, cur, paths)
	}
}

func CollectPathsPython(node *tree_sitter.Node, getNodeText func(*tree_sitter.Node) string, cur []string, paths *[][]string) {
	if node == nil {
		return
	}
	kind := node.Kind()
	cur = append(cur, kind)

	switch kind {
	case "if_statement":
		condNode := node.ChildByFieldName("condition")
		cond := "if"
		if condNode != nil {
			cond += ":" + getNodeText(condNode)
		}
		thenNode := node.ChildByFieldName("consequence")
		thenPath := append(cur, cond+"-then")
		CollectPathsPython(thenNode, getNodeText, thenPath, paths)
		elseNode := node.ChildByFieldName("alternative")
		if elseNode != nil {
			elsePath := append(cur, cond+"-else")
			CollectPathsPython(elseNode, getNodeText, elsePath, paths)
		}
		return
	case "for_statement", "while_statement":
		loopPath := append(cur, kind)
		bodyNode := node.ChildByFieldName("body")
		CollectPathsPython(bodyNode, getNodeText, loopPath, paths)
		return
	case "try_statement":
		tryBlock := node.ChildByFieldName("body")
		CollectPathsPython(tryBlock, getNodeText, append(cur, "try"), paths)
		for i := 0; i < int(node.NamedChildCount()); i++ {
			c := node.NamedChild(uint(i))
			if c.Kind() == "except_clause" {
				CollectPathsPython(c, getNodeText, append(cur, "except"), paths)
			}
		}
		finallyNode := node.ChildByFieldName("finalbody")
		if finallyNode != nil {
			CollectPathsPython(finallyNode, getNodeText, append(cur, "finally"), paths)
		}
		return
	}

	if node.NamedChildCount() == 0 {
		*paths = append(*paths, cur)
		return
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		CollectPathsPython(node.NamedChild(uint(i)), getNodeText, cur, paths)
	}
}

func MinimizePaths(paths [][]string) [][]string {
	branchKinds := map[string]struct{}{
		"if_statement": {}, "for_statement": {}, "while_statement": {},
		"switch_expression": {}, "switch_statement": {},
		"try_statement": {}, "catch_clause": {}, "except_clause": {}, "finally": {},
	}

	type branch struct {
		kind string
		idx  int
	}

	branches := []branch{}
	branchSet := map[branch]struct{}{}
	for _, path := range paths {
		for i, kind := range path {
			if _, ok := branchKinds[kind]; ok {
				b := branch{kind, i}
				if _, exist := branchSet[b]; !exist {
					branches = append(branches, b)
					branchSet[b] = struct{}{}
				}
			}
		}
	}

	covered := map[branch]struct{}{}
	result := [][]string{}
	used := make([]bool, len(paths))
	for len(covered) < len(branches) {
		maxCover, maxIdx := 0, -1
		var maxNew map[branch]struct{}
		for i, path := range paths {
			if used[i] {
				continue
			}
			newCover := map[branch]struct{}{}
			for j, kind := range path {
				b := branch{kind, j}
				if _, ok := branchKinds[kind]; ok {
					if _, already := covered[b]; !already {
						newCover[b] = struct{}{}
					}
				}
			}
			if len(newCover) > maxCover {
				maxCover = len(newCover)
				maxIdx = i
				maxNew = newCover
			}
		}
		if maxIdx == -1 {
			break
		}
		result = append(result, paths[maxIdx])
		used[maxIdx] = true
		for b := range maxNew {
			covered[b] = struct{}{}
		}
	}
	return result
}
