package java

import "github.com/Marksagittarius/pinguis/types"

type JavaParser interface {
	ParseFile(filePath string) (*types.File, error)
	ParseModule(modulePath string) (*types.Module, error)
}
