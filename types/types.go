package types

type manifest struct {
	LayerTarPaths []string
	ConfigPath    string
}

type config struct {
	History []historyEntry
}

type historyEntry struct {
	CreatedBy  string
	EmptyLayer bool
	Size       int64
	ID         string
}

type FileTree struct {
	Name     string
	FileSize uint64
}

type Layer struct {
	Id      string
	Index   int
	Command string
	Size    int64
	Tree    *FileTree
	Names   []string
	Digest  string
}

type FileInfo struct {
	Path string
	Size int64
}
