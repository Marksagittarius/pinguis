package python

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"

    "github.com/Marksagittarius/pinguis/types"
)

func GetFileMetaData(filePath string) (*types.File, error) {
    baseFileName := filepath.Base(filePath)
    jsonFileName := strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName)) + ".json"
    
    jsonFilePath := filepath.Join(filepath.Dir(filePath), jsonFileName)
    
    _, currentFile, _, ok := runtime.Caller(0)
    if !ok {
        fmt.Println("Failed to get current file path")
        return nil, fmt.Errorf("failed to get current file path")
    }
    
    scriptPath := filepath.Join(filepath.Dir(currentFile), "gen_metadata.py")
    
    cmd := exec.Command(
        "python", 
        scriptPath,
        filePath,
        "-o", jsonFilePath,
    )
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Printf("Failed to run gen_metadata.py: %v\nOutput: %s\n", err, output)
        return nil, fmt.Errorf("failed to run gen_metadata.py: %v", err)
    }
    
    fileData, err := types.LoadFromJSON[types.File](jsonFilePath)
    if err != nil {
        fmt.Printf("Failed to load JSON file %s: %v\n", jsonFilePath, err)
        return nil, fmt.Errorf("failed to load JSON file %s: %v", jsonFilePath, err)
    }
    
    if err := os.Remove(jsonFilePath); err != nil {
        fmt.Printf("Warning: Failed to delete temporary JSON file %s: %v\n", jsonFilePath, err)
    }
    
    return &fileData, nil
}
