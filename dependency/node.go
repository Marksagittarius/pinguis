package dependency

// FileNode represents a node in a file tree structure.
// It contains information about the file name, file type,
// its children nodes, parent node, and any dependencies.
//
// FileName: The name of the file or directory.
// FileType: The type of the file (e.g., "file", "directory").
// Children: A slice of pointers to FileNode representing the children of this node.
// Parent: A pointer to the parent FileNode.
// Dependencies: A slice of pointers to FileNode representing the dependencies of this node.
type FileNode struct {
	FileName string  `json:"file_name"`
	FileType string	 `json:"file_type"`
	Children []*FileNode `json:"children"`
	Parent   *FileNode `json:"parent"`
	Dependencies []*FileNode `json:"dependencies"`
}

func NewFileNode(fileName string, fileType string) *FileNode {
	return &FileNode{
		FileName: fileName,
		FileType: fileType,
		Children: []*FileNode{},
		Parent:   nil,
		Dependencies: []*FileNode{},
	}
}

func (n *FileNode) AddChild(child *FileNode) {
	n.Children = append(n.Children, child)
	child.Parent = n
}

func (n *FileNode) AddDependency(dependency *FileNode) {
	n.Dependencies = append(n.Dependencies, dependency)
}
