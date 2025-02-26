package main

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
)

const (
	newLine              = "\n"
	noBranchSpace        = "    "
	branchSpace          = "│   "
	middleItem           = "├─"
	lastItem             = "└─"
	whiteoutPrefix       = ".wh."
	doubleWhiteoutPrefix = ".wh..wh.."
	uncollapsedItem      = "─ "
	collapsedItem        = "⊕ "
)

// FileTree represents a set of files, directories, and their relations.
type FileTree struct {
	Root      *FileNode
	Size      int
	FileSize  uint64
	Name      string
	Id        uuid.UUID
	SortOrder SortOrder
}

// NewFileTree creates an empty FileTree
func NewFileTree() (tree *FileTree) {
	tree = new(FileTree)
	tree.Size = 0
	tree.Root = new(FileNode)
	tree.Root.Tree = tree
	tree.Root.Children = make(map[string]*FileNode)
	tree.Id = uuid.New()
	tree.SortOrder = ByName
	return tree
}

func (tree *FileTree) VisitDepthChildFirst(visitor Visitor, evaluator VisitEvaluator) error {
	sorter := GetSortOrderStrategy(tree.SortOrder)
	return tree.Root.VisitDepthChildFirst(visitor, evaluator, sorter)
}

func (tree *FileTree) AddPath(filepath string, data FileInfo) (*FileNode, []*FileNode, error) {
	filepath = path.Clean(filepath)
	if filepath == "." {
		return nil, nil, fmt.Errorf("cannot add relative path '%s'", filepath)
	}
	nodeNames := strings.Split(strings.Trim(filepath, "/"), "/")
	node := tree.Root
	addedNodes := make([]*FileNode, 0)
	for idx, name := range nodeNames {
		if name == "" {
			continue
		}
		// find or create node
		if node.Children[name] != nil {
			node = node.Children[name]
		} else {
			// don't add paths that should be deleted
			if strings.HasPrefix(name, doubleWhiteoutPrefix) {
				return nil, addedNodes, nil
			}

			// don't attach the payload. The payload is destined for the
			// Path's end node, not any intermediary node.
			node = node.AddChild(name, FileInfo{})
			addedNodes = append(addedNodes, node)

			if node == nil {
				// the child could not be added
				return node, addedNodes, fmt.Errorf(fmt.Sprintf("could not add child node: '%s' (path:'%s')", name, filepath))
			}
		}

		// attach payload to the last specified node
		if idx == len(nodeNames)-1 {
			node.Data.FileInfo = data
		}
	}
	return node, addedNodes, nil
}

func StackTreeRange(trees []*FileTree, start, stop int) (*FileTree, []PathError, error) {
	errors := make([]PathError, 0)
	tree := trees[0].Copy()
	for idx := start; idx <= stop; idx++ {
		failedPaths, err := tree.Stack(trees[idx])
		if len(failedPaths) > 0 {
			errors = append(errors, failedPaths...)
		}
		if err != nil {
			fmt.Printf("could not stack tree range: %v", err)
			return nil, nil, err
		}
	}
	return tree, errors, nil
}

func (tree *FileTree) Stack(upper *FileTree) (failed []PathError, stackErr error) {
	graft := func(node *FileNode) error {
		if node.IsWhiteout() {
			err := tree.RemovePath(node.Path())
			if err != nil {
				failed = append(failed, NewPathError(node.Path(), ActionAdd, err))
			}
		} else {
			_, _, err := tree.AddPath(node.Path(), node.Data.FileInfo)
			if err != nil {
				failed = append(failed, NewPathError(node.Path(), ActionRemove, err))
			}
		}
		return nil
	}
	stackErr = upper.VisitDepthChildFirst(graft, nil)
	return failed, stackErr
}

// RemovePath removes a node from the tree given its path.
func (tree *FileTree) RemovePath(path string) error {
	node, err := tree.GetNode(path)
	if err != nil {
		return err
	}
	return node.Remove()
}

func (tree *FileTree) GetNode(path string) (*FileNode, error) {
	nodeNames := strings.Split(strings.Trim(path, "/"), "/")
	node := tree.Root
	for _, name := range nodeNames {
		if name == "" {
			continue
		}
		if node.Children[name] == nil {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}
		node = node.Children[name]
	}
	return node, nil
}

func (node *FileNode) Remove() error {
	if node == node.Tree.Root {
		return fmt.Errorf("cannot remove the tree root")
	}
	for _, child := range node.Children {
		err := child.Remove()
		if err != nil {
			return err
		}
	}
	delete(node.Parent.Children, node.Name)
	node.Tree.Size--
	return nil
}
func (tree *FileTree) String(showAttributes bool) string {
	return tree.renderStringTreeBetween(0, tree.Size, showAttributes)
}

type renderParams struct {
	node          *FileNode
	spaces        []bool
	childSpaces   []bool
	showCollapsed bool
	isLast        bool
}

func (tree *FileTree) renderStringTreeBetween(startRow, stopRow int, showAttributes bool) string {
	// generate a list of nodes to render
	var params = make([]renderParams, 0)
	var result string

	// visit from the front of the list
	var paramsToVisit = []renderParams{{node: tree.Root, spaces: []bool{}, showCollapsed: false, isLast: false}}
	for currentRow := 0; len(paramsToVisit) > 0 && currentRow <= stopRow; currentRow++ {
		// pop the first node
		var currentParams renderParams
		currentParams, paramsToVisit = paramsToVisit[0], paramsToVisit[1:]

		// take note of the next nodes to visit later
		sorter := GetSortOrderStrategy(tree.SortOrder)
		keys := sorter.orderKeys(currentParams.node.Children)

		var childParams = make([]renderParams, 0)
		for idx, name := range keys {
			child := currentParams.node.Children[name]
			// don't visit this node...
			if child.Data.ViewInfo.Hidden || currentParams.node.Data.ViewInfo.Collapsed {
				continue
			}

			// visit this node...
			isLast := idx == (len(currentParams.node.Children) - 1)
			showCollapsed := child.Data.ViewInfo.Collapsed && len(child.Children) > 0

			// completely copy the reference slice
			childSpaces := make([]bool, len(currentParams.childSpaces))
			copy(childSpaces, currentParams.childSpaces)

			if len(child.Children) > 0 && !child.Data.ViewInfo.Collapsed {
				childSpaces = append(childSpaces, isLast)
			}

			childParams = append(childParams, renderParams{
				node:          child,
				spaces:        currentParams.childSpaces,
				childSpaces:   childSpaces,
				showCollapsed: showCollapsed,
				isLast:        isLast,
			})
		}
		// keep the child nodes to visit later
		paramsToVisit = append(childParams, paramsToVisit...)

		// never process the root node
		if currentParams.node == tree.Root {
			currentRow--
			continue
		}

		// process the current node
		if currentRow >= startRow && currentRow <= stopRow {
			params = append(params, currentParams)
		}
	}

	// render the result
	for idx := range params {
		currentParams := params[idx]

		if showAttributes {
			result += currentParams.node.MetadataString() + " "
		}
		result += currentParams.node.renderTreeLine(currentParams.spaces, currentParams.isLast, currentParams.showCollapsed)
	}

	return result
}

func (tree *FileTree) Copy() *FileTree {
	newTree := NewFileTree()
	newTree.Size = tree.Size
	newTree.FileSize = tree.FileSize
	newTree.Root = tree.Root.Copy(newTree.Root)
	newTree.SortOrder = tree.SortOrder

	// update the tree pointers
	err := newTree.VisitDepthChildFirst(func(node *FileNode) error {
		node.Tree = newTree
		return nil
	}, nil)

	if err != nil {
		fmt.Printf("unable to propagate tree on copy(): %+v", err)
	}

	return newTree
}
