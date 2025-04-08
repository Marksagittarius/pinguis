package dao

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Marksagittarius/pinguis/types"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
)

func FileInfoHandler(weaviate *Weaviate, code string, fileName string) string {
    file, err := FileInfoGetter(weaviate, code, fileName)
    if err != nil {
        return ""
    }

    var prompt strings.Builder
    
    prompt.WriteString(fmt.Sprintf("You are analyzing a file named '%s'", file.Path))
    if file.Module != "" {
        prompt.WriteString(fmt.Sprintf(" from the module '%s'", file.Module))
    }
    prompt.WriteString(".\n\n")
    
    if len(file.Classes) > 0 {
        prompt.WriteString(fmt.Sprintf("The file contains %d classes:\n\n", len(file.Classes)))
        
        for _, class := range file.Classes {
            prompt.WriteString(fmt.Sprintf("- Class '%s':\n", class.Name))
            
			prompt.WriteString("  Fields:\n")
			for _, field := range class.Fields {
				prompt.WriteString(fmt.Sprintf("  - %s: %s\n", field.Name, field.Type))
			}
			prompt.WriteString("\n")
            
            if len(class.Methods) > 0 {
                prompt.WriteString("  Methods:\n")
                for _, method := range class.Methods {
                    prompt.WriteString(fmt.Sprintf("  - %s(", method.Func.Name))
                    
                    paramStrs := make([]string, len(method.Func.Parameters))
                    for i, param := range method.Func.Parameters {
                        paramStrs[i] = fmt.Sprintf("%s: %s", param.Name, param.Type)
                    }
                    prompt.WriteString(strings.Join(paramStrs, ", "))
                    prompt.WriteString(")")
                    
                    if len(method.Func.ReturnTypes) > 0 {
                        prompt.WriteString(" -> ")
                        prompt.WriteString(strings.Join(method.Func.ReturnTypes, ", "))
                    }
                    prompt.WriteString("\n")
                }
                prompt.WriteString("\n")
            }
        }
    }
    
    if len(file.Interfaces) > 0 {
        prompt.WriteString(fmt.Sprintf("The file contains %d interfaces:\n\n", len(file.Interfaces)))
        
        for _, iface := range file.Interfaces {
            prompt.WriteString(fmt.Sprintf("- Interface '%s':\n", iface.Name))
            
            if len(iface.Methods) > 0 {
                prompt.WriteString("  Methods:\n")
                for _, method := range iface.Methods {
                    prompt.WriteString(fmt.Sprintf("  - %s(", method.Name))
                    
                    paramStrs := make([]string, len(method.Parameters))
                    for i, param := range method.Parameters {
                        paramStrs[i] = fmt.Sprintf("%s: %s", param.Name, param.Type)
                    }
                    prompt.WriteString(strings.Join(paramStrs, ", "))
                    prompt.WriteString(")")
                    
                    if len(method.ReturnTypes) > 0 {
                        prompt.WriteString(" -> ")
                        prompt.WriteString(strings.Join(method.ReturnTypes, ", "))
                    }
                    prompt.WriteString("\n")
                }
                prompt.WriteString("\n")
            }
        }
    }
    
    if len(file.Functions) > 0 {
        prompt.WriteString(fmt.Sprintf("The file contains %d standalone functions:\n\n", len(file.Functions)))
        
        for _, function := range file.Functions {
            prompt.WriteString(fmt.Sprintf("- %s(", function.Name))
            
            paramStrs := make([]string, len(function.Parameters))
            for i, param := range function.Parameters {
                paramStrs[i] = fmt.Sprintf("%s: %s", param.Name, param.Type)
            }
            prompt.WriteString(strings.Join(paramStrs, ", "))
            prompt.WriteString(")")
            
            if len(function.ReturnTypes) > 0 {
                prompt.WriteString(" -> ")
                prompt.WriteString(strings.Join(function.ReturnTypes, ", "))
            }
            prompt.WriteString("\n")    
        }
    }
    
    prompt.WriteString("\nPlease analyze this code structure and provide insights or answer questions about it.")
    
    return prompt.String()
}

func FileInfoGetter(weaviate *Weaviate, code string, fileName string) (*types.File, error) {
    client := weaviate.GetClient()
    res, err := client.GraphQL().Get().WithClassName("File").WithFields(ToFields(types.File{})...).
        WithWhere(filters.Where().WithPath([]string{"path"}).WithOperator(filters.Equal).WithValueText(fileName)).
        Do(weaviate.GetContext())
    if err != nil {
        return nil, fmt.Errorf("weaviate query failed: %w", err)
    }

    getMap, ok := res.Data["Get"].(map[string]any)
    if !ok {
        return nil, fmt.Errorf("invalid response format: missing 'Get' key")
    }

    fileArray, ok := getMap["File"].([]any)
    if !ok || len(fileArray) == 0 {
        return nil, fmt.Errorf("no file found with path: %s", fileName)
    }

    data, ok := fileArray[0].(map[string]any)
    if !ok {
        return nil, fmt.Errorf("invalid file data format")
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal data: %w", err)
    }

    var file types.File
    if err := json.Unmarshal(jsonData, &file); err != nil {
        return nil, fmt.Errorf("failed to unmarshal data to File struct: %w", err)
    }

    return &file, nil
}
