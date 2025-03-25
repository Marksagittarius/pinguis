package filetree

import (
    "os"
    "path/filepath"
)

type FileTreeBuilder struct{}

func (b *FileTreeBuilder) BuildTree(path string) (*FileTree, error) {
	rootName := filepath.Base(path)
	root := NewFileNode(rootName, "dir")
	tree := NewFileTree(root)

	err := b.buildTreeRecursive(path, root)
	return tree, err
}

// buildTreeRecursive is a recursive function that builds a file tree structure starting from the given path.
// It reads the directory entries at the specified path, creates corresponding FileNode objects, and adds them
// as children to the provided parentNode. If an entry is a directory, the function calls itself recursively
// to process the directory's contents.
//
// Parameters:
//   - path: The file system path to read and build the tree from.
//   - parentNode: The parent FileNode to which the new nodes will be added.
//
// Returns:
//   - error: An error if any occurs during reading the directory or processing its entries.
func (b *FileTreeBuilder) buildTreeRecursive(path string, parentNode *FileNode) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		entryPath := filepath.Join(path, name)

		var nodeType string
		if entry.IsDir() {
			nodeType = "dir"
		} else {
			ext := filepath.Ext(name)
			if ext == "" {
				nodeType = "file"
			} else {
				nodeType = ext[1:]
			}
		}

		node := NewFileNode(name, nodeType)
		parentNode.AddChild(node)

		if nodeType == "dir" {
			err := b.buildTreeRecursive(entryPath, node)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
