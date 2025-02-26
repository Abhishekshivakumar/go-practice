package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash"
)

type FileInfo struct {
	Path     string
	TypeFlag byte
	Linkname string
	hash     uint64
	Size     int64
	Mode     os.FileMode
	Uid      int
	Gid      int
	IsDir    bool
}

// NewFileInfoFromTarHeader extracts the metadata from a tar header and file contents and generates a new FileInfo object.
func NewFileInfoFromTarHeader(reader *tar.Reader, header *tar.Header, path string) FileInfo {
	var hash uint64
	if header.Typeflag != tar.TypeDir {
		hash = getHashFromReader(reader)
	}

	return FileInfo{
		Path:     path,
		TypeFlag: header.Typeflag,
		Linkname: header.Linkname,
		hash:     hash,
		Size:     header.FileInfo().Size(),
		Mode:     header.FileInfo().Mode(),
		Uid:      header.Uid,
		Gid:      header.Gid,
		IsDir:    header.FileInfo().IsDir(),
	}
}

func getHashFromReader(reader io.Reader) uint64 {
	h := xxhash.New()

	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println(err)
		}
		if n == 0 {
			break
		}

		_, err = h.Write(buf[:n])
		if err != nil {
			fmt.Println(err)
		}
	}

	return h.Sum64()
}

// Copy duplicates a FileInfo
func (data *FileInfo) Copy() *FileInfo {
	if data == nil {
		return nil
	}
	return &FileInfo{
		Path:     data.Path,
		TypeFlag: data.TypeFlag,
		Linkname: data.Linkname,
		hash:     data.hash,
		Size:     data.Size,
		Mode:     data.Mode,
		Uid:      data.Uid,
		Gid:      data.Gid,
		IsDir:    data.IsDir,
	}
}
