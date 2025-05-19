package dependency

type FileTree struct {
	Root *FileNode `json:"root"`
}

// NewFileTree creates a new FileTree with the given root FileNode.
// It returns a pointer to the newly created FileTree.
//
// Parameters:
//   - root: A pointer to the root FileNode of the FileTree.
//
// Returns:
//   - A pointer to the newly created FileTree.
func NewFileTree(root *FileNode) *FileTree {
	return &FileTree{
		Root: root,
	}
}
