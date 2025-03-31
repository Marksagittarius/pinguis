package dao

// Package dao provides data access functionalities for interacting with Weaviate,
// a vector search engine. This file contains the necessary imports for using the
// Weaviate Go client library to perform operations such as data management,
// schema management, and GraphQL queries.
//
// The imported packages include:
// - context: For managing request-scoped values and deadlines.
// - weaviate: The main Weaviate Go client package.
// - weaviate/data: For data-related operations such as CRUD operations on objects.
// - weaviate/graphql: For executing GraphQL queries against the Weaviate instance.
// - weaviate/schema: For managing the schema of the Weaviate instance.
// - models: For working with Weaviate's data models.
import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/data"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/schema"
	"github.com/weaviate/weaviate/entities/models"
)

type Weaviate struct {
	client  *weaviate.Client
	context context.Context
}

// New creates a new instance of the Weaviate struct with the provided configuration
// and context. It initializes a Weaviate client using the given configuration.
//
// Parameters:
//   - config: The configuration settings required to initialize the Weaviate client.
//   - context: The context to be used for the Weaviate instance.
//
// Returns:
//   - *Weaviate: A pointer to the newly created Weaviate instance.
//   - error: An error if the client initialization fails.
func New(config weaviate.Config, context context.Context) (*Weaviate, error) {
	client, err := weaviate.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &Weaviate{client: client, context: context}, nil
}

// GetClient returns the Weaviate client instance associated with the Weaviate object.
// This client can be used to interact with the Weaviate API.
func (w *Weaviate) GetClient() *weaviate.Client {
	return w.client
}

// AddClass adds a new class to the Weaviate schema.
//
// Parameters:
//   - context: The context for the operation, used for managing timeouts and cancellations.
//   - class: A pointer to the models.Class object representing the class to be added.
//
// Returns:
//   - error: An error if the operation fails, or nil if the class is successfully added.
func (w *Weaviate) AddClass(context context.Context, class *models.Class) error {
	return w.client.Schema().ClassCreator().WithClass(class).Do(w.context)
}

// GetClassByName retrieves a class definition from the Weaviate schema by its name.
//
// Parameters:
//   - className: The name of the class to retrieve.
//
// Returns:
//   - A pointer to the models.Class object representing the class definition.
//   - An error if the class cannot be retrieved or if an issue occurs during the request.
func (w *Weaviate) GetClassByName(className string) (*models.Class, error) {
	return w.client.Schema().ClassGetter().WithClassName(className).Do(w.context)
}

// GetSchema retrieves the schema from the Weaviate client.
// It returns a pointer to a schema.Dump object containing the schema
// and an error if the operation fails.
func (w *Weaviate) GetSchema() (*schema.Dump, error) {
	return w.client.Schema().Getter().Do(w.context)
}

// DeleteClass deletes a class from the Weaviate schema.
//
// Parameters:
//   - className: The name of the class to be deleted.
//
// Returns:
//   - An error if the deletion fails, or nil if the operation is successful.
func (w *Weaviate) AddProperties(className string, property *models.Property) error {
	return w.client.Schema().PropertyCreator().WithClassName(className).WithProperty(property).Do(w.context)
}

// AddObjects adds multiple objects to the Weaviate database in a single batch operation.
// It takes a variadic parameter of pointers to models.Object and returns a slice of
// models.ObjectsGetResponse and an error.
//
// Parameters:
//   - objects: A variadic list of pointers to models.Object representing the objects to be added.
//
// Returns:
//   - []models.ObjectsGetResponse: A slice containing the responses for the added objects.
//   - error: An error if the batch operation fails, otherwise nil.
func (w *Weaviate) AddObjects(objects ...*models.Object) ([]models.ObjectsGetResponse, error) {
	return w.client.Batch().ObjectsBatcher().WithObjects(objects...).Do(w.context)
}

// CreateObject creates a new object in Weaviate with the specified class name and properties.
//
// Parameters:
//   - className: The name of the class to which the object belongs.
//   - properties: A map containing the properties of the object to be created.
//
// Returns:
//   - *data.ObjectWrapper: A wrapper containing the created object data.
//   - error: An error if the object creation fails.
func (w *Weaviate) CreateObject(className string, properties map[string]any) (*data.ObjectWrapper, error) {
	return w.client.Data().Creator().WithClassName(className).WithProperties(properties).Do(w.context)
}

// GetObjectsByClass retrieves objects from the Weaviate database based on the specified class name.
// It allows specifying optional GraphQL fields to customize the query.
//
// Parameters:
//   - className: The name of the class to query objects from.
//   - fields: Optional GraphQL fields to include in the query.
//
// Returns:
//   - *models.GraphQLResponse: The response containing the queried objects.
//   - error: An error if the query fails or encounters an issue.
func (w *Weaviate) GetObjectsByClass(className string, fields ...graphql.Field) (*models.GraphQLResponse, error) {
	return w.client.GraphQL().Get().WithClassName(className).WithFields(fields...).Do(w.context)
}

// GetObjectByID retrieves objects of a specified class by their unique ID from the Weaviate database.
//
// Parameters:
//   - className: The name of the class to which the object belongs.
//   - id: The unique identifier of the object to retrieve.
//
// Returns:
//   - []*models.Object: A slice of objects matching the specified class and ID.
//   - error: An error if the retrieval operation fails.
func (w *Weaviate) GetObjectByID(className string, id string) ([]*models.Object, error) {
	return w.client.Data().ObjectsGetter().WithClassName(className).WithID(id).Do(w.context)
}

// UpdateObject updates an object in Weaviate with the specified class name, ID, and properties.
// It uses the Weaviate client's data updater with merge functionality to apply the changes.
//
// Parameters:
//   - className: The name of the class to which the object belongs.
//   - id: The unique identifier of the object to be updated.
//   - properties: A map containing the properties to be updated on the object.
//
// Returns:
//   - error: An error if the update operation fails, otherwise nil.
func (w *Weaviate) UpdateObject(className string, id string, properties map[string]any) error {
	return w.client.Data().Updater().WithMerge().WithID(id).WithClassName(className).WithProperties(properties).Do(w.context)
}

// ReplaceObject replaces an existing object in Weaviate with the specified class name, ID, and properties.
//
// Parameters:
//   - className: The name of the class to which the object belongs.
//   - id: The unique identifier of the object to be replaced.
//   - properties: A map containing the new properties to update the object with.
//
// Returns:
//   - error: An error if the operation fails, or nil if the replacement is successful.
func (w *Weaviate) ReplaceObject(className string, id string, properties map[string]any) error {
	return w.client.Data().Updater().WithID(id).WithClassName(className).WithProperties(properties).Do(w.context)
}

// DeleteObject deletes an object from the Weaviate database based on the specified class name and ID.
//
// Parameters:
//   - className: The name of the class to which the object belongs.
//   - id: The unique identifier of the object to be deleted.
//
// Returns:
//   - error: An error if the deletion fails, or nil if the operation is successful.
func (w *Weaviate) DeleteObject(className string, id string) error {
	return w.client.Data().Deleter().WithID(id).WithClassName(className).Do(w.context)
}

// ToClass converts a given object to a *models.Class representation.
// The function inspects the type of the provided object using reflection
// and generates a class structure with properties based on the object's fields.
//
// Parameters:
//   - object: An interface{} representing the object to be converted.
//
// Returns:
//   - A pointer to a models.Class instance if the object is a struct or a pointer to a struct.
//     Returns nil if the object is not a struct or a pointer to a struct.
//
// Behavior:
//   - The function extracts the name of the struct as the class name.
//   - It iterates over the exported fields of the struct to generate properties.
//   - Field names are converted to lowercase unless overridden by a `json` tag.
//   - Supported field types are mapped to specific data types:
//   - string -> "string"
//   - int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64 -> "int"
//   - float32, float64 -> "number"
//   - Fields with unsupported types or unexported fields are ignored.
func ToClass(object any) *models.Class {
	t := reflect.TypeOf(object)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	class := &models.Class{
		Class:      t.Name(),
		Properties: []*models.Property{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		propName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "-" {
				propName = parts[0]
			}
		}

		if propName == field.Name {
			propName = strings.ToLower(propName)
		}

		var dataType string
		var isArray bool

		fieldType := field.Type
		if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
			isArray = true
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.String:
			dataType = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			dataType = "int"
		case reflect.Float32, reflect.Float64:
			dataType = "number"
		case reflect.Struct:
			dataType = fieldType.Name()

			if len(dataType) > 0 {
				firstChar := dataType[0:1]
				dataType = strings.ToUpper(firstChar) + dataType[1:]
			}
		default:
			continue
		}

		if isArray {
			dataType = dataType + "[]"
		}

		property := &models.Property{
			Name:     propName,
			DataType: []string{dataType},
		}

		class.Properties = append(class.Properties, property)
	}

	return class
}

// ToProperties converts a struct or a pointer to a struct into a map[string]any,
// where the keys are the struct field names (converted to lowercase) or their
// corresponding JSON tag names (if specified). Non-exported fields are ignored.
//
// If the input is not a struct or a pointer to a struct, the function returns nil.
//
// Parameters:
//   - object: The input value to be converted. It must be a struct or a pointer to a struct.
//
// Returns:
//   - A map[string]any containing the struct's field names (or JSON tag names) as keys
//     and their corresponding values. Returns nil if the input is not a struct or a pointer to a struct.
//
// Notes:
//   - Fields with a JSON tag of "-" are ignored.
//   - If a JSON tag is present, its first value is used as the key in the resulting map.
//   - Field names are converted to lowercase if no JSON tag is specified.
func ToProperties(object any) map[string]any {
	if object == nil {
		return nil
	}

	t := reflect.TypeOf(object)
	v := reflect.ValueOf(object)

	if t.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		return ToProperties(v.Elem().Interface())
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	properties := make(map[string]any)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		propName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] == "-" {
				continue
			} else if parts[0] != "" {
				propName = parts[0]
			}
		}

		if propName == field.Name {
			propName = strings.ToLower(propName)
		}

		fieldValue := v.Field(i)
		properties[propName] = processFieldValue(fieldValue)
	}

	return properties
}

func processFieldValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		return processFieldValue(v.Elem())
	}

	switch v.Kind() {
	case reflect.Struct:
		return ToProperties(v.Interface())

	case reflect.Slice, reflect.Array:
		length := v.Len()
		result := make([]any, length)

		for i := 0; i < length; i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Struct ||
				elem.Kind() == reflect.Ptr ||
				elem.Kind() == reflect.Slice ||
				elem.Kind() == reflect.Array ||
				elem.Kind() == reflect.Map {
				result[i] = processFieldValue(elem)
			} else {
				result[i] = elem.Interface()
			}
		}

		return result

	case reflect.Map:
		result := make(map[string]any)

		iter := v.MapRange()
		for iter.Next() {
			key := fmt.Sprintf("%v", iter.Key().Interface())
			val := iter.Value()

			if val.Kind() == reflect.Struct ||
				val.Kind() == reflect.Ptr ||
				val.Kind() == reflect.Slice ||
				val.Kind() == reflect.Array ||
				val.Kind() == reflect.Map {
				result[key] = processFieldValue(val)
			} else {
				result[key] = val.Interface()
			}
		}

		return result

	default:
		return v.Interface()
	}
}

// ToFields converts a Go struct or pointer to a struct into a slice of GraphQL fields.
// It recursively processes nested structs and respects JSON tags for field names.
//
// Parameters:
//   - object: The input object, which can be any type. If the object is not a struct
//             or a pointer to a struct, an empty slice is returned.
//
// Returns:
//   - []graphql.Field: A slice of GraphQL fields representing the structure of the input object.
//
// Behavior:
//   - If the input is nil, an empty slice is returned.
//   - If the input is a pointer, it is dereferenced to access the underlying struct.
//   - If the input is not a struct, an empty slice is returned.
//   - Fields that are unexported or have a JSON tag with "-" are ignored.
//   - JSON tags are used to determine field names, falling back to the struct field name if no tag is present.
//   - Nested structs are processed recursively, creating nested GraphQL fields.
//   - Slice and array fields are handled by inspecting their element type.
//
// Example:
//   Given the following struct:
//     type Example struct {
//         ID   string `json:"id"`
//         Name string
//         Meta struct {
//             CreatedAt string `json:"created_at"`
//         }
//     }
//
//   Calling ToFields(Example{}) would produce:
//     []graphql.Field{
//         {Name: "id"},
//         {Name: "Name"},
//         {
//             Name: "Meta",
//             Fields: []graphql.Field{
//                 {
//                     Name: "... on Meta",
//                     Fields: []graphql.Field{
//                         {Name: "created_at"},
//                     },
//                 },
//             },
//         },
//     }
func ToFields(object any) []graphql.Field {
    t := reflect.TypeOf(object)
    
    if t == nil {
        return []graphql.Field{}
    }
    
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    
    if t.Kind() != reflect.Struct {
        return []graphql.Field{}
    }
    
    var fields []graphql.Field
    
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        
        if !field.IsExported() {
            continue
        }
        
        fieldName := field.Name
        if jsonTag := field.Tag.Get("json"); jsonTag != "" {
            parts := strings.Split(jsonTag, ",")
            if parts[0] == "-" {
                continue
            } else if parts[0] != "" {
                fieldName = parts[0]
            }
        }
        
        fieldType := field.Type
        
        if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
            fieldType = fieldType.Elem()
            
            if fieldType.Kind() == reflect.Ptr {
                fieldType = fieldType.Elem()
            }
        }
        
        if fieldType.Kind() == reflect.Struct {
            nestedInstance := reflect.New(fieldType).Elem().Interface()
            
            typeName := fieldType.Name()
            
            nestedField := graphql.Field{
                Name: fieldName,
                Fields: []graphql.Field{
                    {
                        Name:   "... on " + typeName,
                        Fields: ToFields(nestedInstance), // Recursive call
                    },
                },
            }
            
            fields = append(fields, nestedField)
        } else {
            fields = append(fields, graphql.Field{Name: fieldName})
        }
    }
    
    return fields
}
