package dao

import (
	"encoding/json"
	"fmt"

	"github.com/Marksagittarius/pinguis/types"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
)

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
