package types

import (
    "encoding/json"
    "fmt"
    "os"
)

// LoadFromJSON reads a JSON file from the specified file path and unmarshals its content
// into a generic type T. If the unmarshaling fails and the target type is determined to
// be a Module or a pointer to a Module, it attempts to unmarshal the JSON into a slice
// of Modules and returns the first element if available.
//
// Type Parameters:
//   - T: The type into which the JSON content will be unmarshaled.
//
// Parameters:
//   - filePath: The path to the JSON file to be read.
//
// Returns:
//   - T: The unmarshaled result of type T.
//   - error: An error if the file cannot be read, the JSON cannot be unmarshaled, or
//            if there is a type conversion issue.
//
// Errors:
//   - Returns an error if the file cannot be read.
//   - Returns an error if the JSON cannot be unmarshaled into the target type T.
//   - Returns an error if the JSON represents an empty array of Modules when T is a Module.
//   - Returns an error if there is a type conversion issue when handling Modules.
func LoadFromJSON[T any](filePath string) (T, error) {
    var result T
    
    data, err := os.ReadFile(filePath)
    if err != nil {
        return result, fmt.Errorf("failed to read file: %w", err)
    }

    if err := json.Unmarshal(data, &result); err != nil {
        var isModule bool
        if _, ok := any(result).(Module); ok {
            isModule = true
        } else if _, ok := any(result).(*Module); ok {
            isModule = true
        }
        
        if isModule {
            var modules []Module
            if err := json.Unmarshal(data, &modules); err != nil {
                return result, fmt.Errorf("failed to unmarshal JSON: %w", err)
            }
            if len(modules) > 0 {
                moduleResult, ok := any(modules[0]).(T)
                if !ok {
                    return result, fmt.Errorf("type conversion error")
                }
                return moduleResult, nil
            }
            return result, fmt.Errorf("empty module array in JSON")
        }
        
        return result, fmt.Errorf("failed to unmarshal JSON: %w", err)
    }

    return result, nil
}

// SaveToJSON saves the provided data of any type to a JSON file at the specified file path.
// The data is marshaled into a pretty-printed JSON format.
//
// Type Parameters:
//   - T: The type of the data to be saved.
//
// Parameters:
//   - filePath: The path to the file where the JSON data will be written.
//   - data: The data to be marshaled and saved as JSON.
//
// Returns:
//   - error: An error if the marshaling or file writing fails, otherwise nil.
func SaveToJSON[T any](filePath string, data T) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON data to file: %w", err)
	}

	return nil
}
