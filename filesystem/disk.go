package main

import (
	"bytes"
	"encoding/binary"
	"os"
)

const (
	MaxBlocks   = 12
	BlockOffset = 4096
	BlockSize   = 4096
)

// These are raw on disk structures
type Superblock struct {
	MagicNumber uint32
	BlockSize   uint32
	InodeOffset uint32
	TotalBlocks uint32
	TotalInodes uint32
}

/*
size calculation:
4 * 4 + 8 * 4 + 15 * 4 + 4 = 112
*/
type Inode struct {
	Mode uint32
	Size uint32
	UID  uint32
	GID  uint32

	MTime uint64
	ATime uint64
	CTime uint64
	BTime uint64

	Blocks [15]uint32 // Tripple indirect support
	InUse  uint32
}

type DEntry struct {
	// max 255 bytes filenames is linux std
	FileName    [256]byte
	INodeNumber uint64
}

type FileSystem interface {
	ReadBlock(blockNum uint32) ([]byte, error)
	WriteBlock(blockNum uint32, data []byte) error
	ReadInode(inodeNum uint32) (Inode, error)
	WriteInode(inodeNum uint32, inode Inode) error
	ReadDEntry(inodeNum uint32) ([]DEntry, error)
	WriteDEntry(inodeNum uint32, dentry []DEntry) error
}

type FileSystem_v1 struct {
	filePath string
	sb       Superblock
}

func Serialize[T any](v *T) ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Deserialize[T any](data []byte, v *T) error {
	return binary.Read(bytes.NewReader(data), binary.LittleEndian, v)
}

func (fs *FileSystem_v1) ReadBlock(blockNum uint32) ([]byte, error) {
	f, err := os.OpenFile(fs.filePath, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data := make([]byte, BlockSize)
	if n, err := f.ReadAt(data, int64(blockNum*BlockSize)); n != BlockSize || err != nil {
		return nil, err
	}

	return data, nil
}

func (fs *FileSystem_v1) WriteBlock(blockNum uint32, data []byte) error {
	f, err := os.OpenFile(fs.filePath, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if n, err := f.WriteAt(data, int64(blockNum*BlockSize)); n != BlockSize || err != nil {
		return err
	}
	return nil
}

// func (fs *FileSystem_v1) ReadInode(inodeNum uint32) (Inode, error) {}
// func (fs *FileSystem_v1) WriteInode(inodeNum uint32, inode Inode) (error) {}
// func (fs *FileSystem_v1) ReadDEntry(inodeNum uint32) ([]DEntry, error) {}
// func (fs *FileSystem_v1) WriteDEntry(inodeNum uint32, dentry []DEntry) (error) {}
