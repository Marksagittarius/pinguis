package fileio

import "os"

type FileIO interface {
	Read(filePath string) ([]byte, error)
	Write(filePath string, data []byte) error
}

type SimpleFileIO struct{}

func (sio *SimpleFileIO) Read(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func (sio *SimpleFileIO) Write(filePath string, data []byte) error {
	return os.WriteFile(filePath, data, 0644)
}
