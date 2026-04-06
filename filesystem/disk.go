package main

const (
	MaxBlocks = 12
)

/*
size calculation:
4 + 4 + 4 + 4 + 8 + 8 + 12 * 8 + 4 =
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

	Blocks [MaxBlocks]uint32
	InUse  uint32
}

type DEntry struct {
	// max 255 bytes filenames is linux std
	FileName    [256]byte
	INodeNumber uint64
}
