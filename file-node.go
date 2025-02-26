package main

import (
	"archive/tar"
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/phayes/permbits"
)

const (
	AttributeFormat = "%s%s %11s %10s "
)

type FileNode struct {
	Tree     *FileTree
	Parent   *FileNode
	Size     int64 // memoized total size of file or directory
	Name     string
	Data     NodeData
	Children map[string]*FileNode
	path     string
}

// NewNode creates a new FileNode relative to the given parent node with a payload.
func NewNode(parent *FileNode, name string, data FileInfo) (node *FileNode) {
	node = new(FileNode)
	node.Name = name
	node.Data = *NewNodeData()
	node.Data.FileInfo = *data.Copy()
	node.Size = -1 // signal lazy load later

	node.Children = make(map[string]*FileNode)
	node.Parent = parent
	if parent != nil {
		node.Tree = parent.Tree
	}

	return node
}
func (node *FileNode) String() string {
	var display string
	if node == nil {
		return ""
	}

	display = node.Name
	if node.Data.FileInfo.TypeFlag == tar.TypeSymlink || node.Data.FileInfo.TypeFlag == tar.TypeLink {
		display += " → " + node.Data.FileInfo.Linkname
	}
	return diffTypeColor[node.Data.DiffType].Sprint(display)
}

func (node *FileNode) GetSize() int64 {
	if 0 <= node.Size {
		return node.Size
	}
	var sizeBytes int64

	if node.IsLeaf() {
		sizeBytes = node.Data.FileInfo.Size
	} else {
		sizer := func(curNode *FileNode) error {

			if curNode.Data.DiffType != Removed || node.Data.DiffType == Removed {
				sizeBytes += curNode.Data.FileInfo.Size
			}
			return nil
		}
		err := node.VisitDepthChildFirst(sizer, nil, nil)
		if err != nil {
			fmt.Println("unable to propagate node for metadata: ", err)
		}
	}
	node.Size = sizeBytes
	return node.Size
}

func (node *FileNode) IsLeaf() bool {
	return len(node.Children) == 0
}

type Visitor func(*FileNode) error

type VisitEvaluator func(*FileNode) bool

func (node *FileNode) VisitDepthChildFirst(visitor Visitor, evaluator VisitEvaluator, sorter OrderStrategy) error {
	if sorter == nil {
		sorter = GetSortOrderStrategy(ByName)
	}
	keys := sorter.orderKeys(node.Children)
	for _, name := range keys {
		child := node.Children[name]
		err := child.VisitDepthChildFirst(visitor, evaluator, sorter)
		if err != nil {
			return err
		}
	}
	// never visit the root node
	if node == node.Tree.Root {
		return nil
	} else if evaluator != nil && evaluator(node) || evaluator == nil {
		return visitor(node)
	}

	return nil
}

func (node *FileNode) AddChild(name string, data FileInfo) (child *FileNode) {
	// never allow processing of purely whiteout flag files (for now)
	if strings.HasPrefix(name, doubleWhiteoutPrefix) {
		return nil
	}

	child = NewNode(node, name, data)
	if node.Children[name] != nil {
		// tree node already exists, replace the payload, keep the children
		node.Children[name].Data.FileInfo = *data.Copy()
	} else {
		node.Children[name] = child
		node.Tree.Size++
	}

	return child
}

func (node *FileNode) Path() string {
	if node.path == "" {
		var path []string
		curNode := node
		for {
			if curNode.Parent == nil {
				break
			}

			name := curNode.Name
			if curNode == node {
				// white out prefixes are fictitious on leaf nodes
				name = strings.TrimPrefix(name, whiteoutPrefix)
			}

			path = append([]string{name}, path...)
			curNode = curNode.Parent
		}
		node.path = "/" + strings.Join(path, "/")
	}
	return strings.Replace(node.path, "//", "/", -1)
}

func (node *FileNode) IsWhiteout() bool {
	return strings.HasPrefix(node.Name, whiteoutPrefix)
}

func (node *FileNode) Copy(parent *FileNode) *FileNode {
	newNode := NewNode(parent, node.Name, node.Data.FileInfo)
	newNode.Data.ViewInfo = node.Data.ViewInfo
	newNode.Data.DiffType = node.Data.DiffType
	for name, child := range node.Children {
		newNode.Children[name] = child.Copy(newNode)
		child.Parent = newNode
	}
	return newNode
}

func (node *FileNode) MetadataString() string {
	if node == nil {
		return ""
	}

	fileMode := permbits.FileMode(node.Data.FileInfo.Mode).String()
	dir := "-"
	if node.Data.FileInfo.IsDir {
		dir = "d"
	}
	user := node.Data.FileInfo.Uid
	group := node.Data.FileInfo.Gid
	userGroup := fmt.Sprintf("%d:%d", user, group)

	// don't include file sizes of children that have been removed (unless the node in question is a removed dir,
	// then show the accumulated size of removed files)
	sizeBytes := node.GetSize()

	size := humanize.Bytes(uint64(sizeBytes))

	return diffTypeColor[node.Data.DiffType].Sprint(fmt.Sprintf(AttributeFormat, dir, fileMode, userGroup, size))
}

var diffTypeColor = map[DiffType]*color.Color{
	Added:      color.New(color.FgGreen),
	Removed:    color.New(color.FgRed),
	Modified:   color.New(color.FgYellow),
	Unmodified: color.New(color.Reset),
}

func (node *FileNode) renderTreeLine(spaces []bool, last bool, collapsed bool) string {
	var otherBranches string
	for _, space := range spaces {
		if space {
			otherBranches += noBranchSpace
		} else {
			otherBranches += branchSpace
		}
	}

	thisBranch := middleItem
	if last {
		thisBranch = lastItem
	}

	collapsedIndicator := uncollapsedItem
	if collapsed {
		collapsedIndicator = collapsedItem
	}

	return otherBranches + thisBranch + collapsedIndicator + node.String() + newLine
}
