# Toy Filesystem — Go Implementation Snippets

## Phase 1 — On-Disk Structs (`fs.go`)

```go
// fs.go

package main

import (
    "bytes"
    "encoding/binary"
    "os"
)

const (
    BlockSize       = 4096
    Magic           = 0xDEADC0DE
    MaxInodes       = 128
    MaxDirectBlocks = 12

    SuperblockLBA  = 0
    BitmapLBA      = 1
    InodeTableLBA  = 2
    DataStartLBA   = 3
)

// Block 0 — filesystem metadata
type Superblock struct {
    Magic       uint32
    TotalBlocks uint32
    InodeCount  uint32
    FreeBlocks  uint32
    FreeInodes  uint32
    BlockSize   uint32
}

// One entry in the inode table (block 2)
// Must be a fixed size — calculate yours: 4+4+4+4+8+8+8+(12*4)+4 = 96 bytes
type Inode struct {
    Mode   uint32   // S_IFREG | S_IFDIR | permission bits
    Size   uint32   // file size in bytes
    UID    uint32
    GID    uint32
    ATime  uint64
    MTime  uint64
    CTime  uint64
    Blocks [MaxDirectBlocks]uint32 // LBAs of data blocks
    InUse  uint32   // 1 = allocated, 0 = free
}

// One slot inside a directory's data block
// 64 bytes total → 4096/64 = 64 entries per directory block
type DirEntry struct {
    InodeNum uint32
    Name     [60]byte // null-terminated
}

// --- Disk helpers ---

var disk *os.File

func readBlock(lba uint32, v any) error {
    buf := make([]byte, BlockSize)
    _, err := disk.ReadAt(buf, int64(lba)*BlockSize)
    if err != nil {
        return err
    }
    return binary.Read(bytes.NewReader(buf), binary.LittleEndian, v)
}

func writeBlock(lba uint32, v any) error {
    buf := new(bytes.Buffer)
    if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
        return err
    }
    // Pad to full block size
    padded := make([]byte, BlockSize)
    copy(padded, buf.Bytes())
    _, err := disk.WriteAt(padded, int64(lba)*BlockSize)
    return err
}

func readInode(num uint32) (Inode, error) {
    offset := int64(InodeTableLBA)*BlockSize + int64(num)*int64(binary.Size(Inode{}))
    buf := make([]byte, binary.Size(Inode{}))
    _, err := disk.ReadAt(buf, offset)
    if err != nil {
        return Inode{}, err
    }
    var ino Inode
    binary.Read(bytes.NewReader(buf), binary.LittleEndian, &ino)
    return ino, nil
}

func writeInode(num uint32, ino Inode) error {
    buf := new(bytes.Buffer)
    binary.Write(buf, binary.LittleEndian, ino)
    _, err := disk.WriteAt(buf.Bytes(), int64(InodeTableLBA)*BlockSize+int64(num)*int64(binary.Size(Inode{})))
    return err
}
```

---

## Phase 2 — mkfs (`mkfs/main.go`)

```go
// mkfs/main.go

package main

import (
    "encoding/binary"
    "log"
    "os"
    "syscall"
)

func main() {
    f, err := os.OpenFile("mydisk.img", os.O_RDWR, 0644)
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    writeBlock(f, SuperblockLBA, Superblock{
        Magic:       Magic,
        TotalBlocks: 2560,
        InodeCount:  MaxInodes,
        FreeBlocks:  2556,       // blocks 0-3 reserved
        FreeInodes:  MaxInodes - 1, // root inode claimed
        BlockSize:   BlockSize,
    })

    // Bitmap: all 0xFF (free), except first byte marks blocks 0-3 used
    bitmap := [BlockSize]byte{}
    for i := range bitmap {
        bitmap[i] = 0xFF
    }
    bitmap[0] = 0b11110000 // blocks 0,1,2,3 used
    writeBlock(f, BitmapLBA, bitmap)

    // Root inode — inode 0, directory, data in block 3
    root := Inode{
        Mode:  syscall.S_IFDIR | 0755,
        Size:  BlockSize,
        InUse: 1,
    }
    root.Blocks[0] = DataStartLBA
    writeInode(f, 0, root)

    // Root directory data — "." and ".."
    type dirBlock struct {
        Entries [64]DirEntry
    }
    var dir dirBlock
    dir.Entries[0] = DirEntry{InodeNum: 0}
    copy(dir.Entries[0].Name[:], ".")
    dir.Entries[1] = DirEntry{InodeNum: 0}
    copy(dir.Entries[1].Name[:], "..")
    writeBlock(f, DataStartLBA, dir)

    log.Println("mkfs done")
}
```

---

## Phase 3 — FUSE Driver: Read-Only (`toyfs.go`, `main.go`)

```go
// toyfs.go

package main

import (
    "context"
    "syscall"

    "github.com/hanwen/go-fuse/v2/fs"
    "github.com/hanwen/go-fuse/v2/fuse"
)

// ToyNode represents any file or directory in our filesystem.
// The embedded fs.Inode is go-fuse's bookkeeping — don't touch it directly.
type ToyNode struct {
    fs.Inode
    inodeNum uint32
}

// Getattr — called by stat(). Fill in fuse.AttrOut from your on-disk Inode.
func (n *ToyNode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
    ino, err := readInode(n.inodeNum)
    if err != nil {
        return syscall.EIO
    }
    out.Mode  = ino.Mode
    out.Size  = uint64(ino.Size)
    out.Uid   = ino.UID
    out.Gid   = ino.GID
    out.Mtime = ino.MTime
    out.Atime = ino.ATime
    out.Ctime = ino.CTime
    return 0
}

// Lookup — called when resolving a path component inside a directory.
// e.g. for "/foo/bar", VFS calls Lookup("bar") on the node for "/foo".
func (n *ToyNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
    inodeNum, err := lookupInDir(n.inodeNum, name)  // scan dir_entry blocks
    if err != nil {
        return nil, syscall.ENOENT
    }

    ino, _ := readInode(inodeNum)
    child := n.NewInode(ctx, &ToyNode{inodeNum: inodeNum}, fs.StableAttr{
        Mode: ino.Mode,
        Ino:  uint64(inodeNum),
    })

    out.Mode = ino.Mode
    out.Size = uint64(ino.Size)
    return child, 0
}

// Readdir — called by ls. Add one entry per DirEntry in the directory's blocks.
func (n *ToyNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
    entries, err := readDirEntries(n.inodeNum)
    if err != nil {
        return nil, syscall.EIO
    }
    var result []fuse.DirEntry
    for _, e := range entries {
        if e.InodeNum == 0 {
            continue
        }
        ino, _ := readInode(e.InodeNum)
        result = append(result, fuse.DirEntry{
            Name: nullTermToString(e.Name[:]),
            Ino:  uint64(e.InodeNum),
            Mode: ino.Mode,
        })
    }
    return fs.NewListDirStream(result), 0
}
```

```go
// main.go

package main

import (
    "log"
    "os"

    "github.com/hanwen/go-fuse/v2/fs"
)

func main() {
    diskPath   := os.Args[1]  // mydisk.img
    mountPoint := os.Args[2]  // /mnt/toy

    var err error
    disk, err = os.OpenFile(diskPath, os.O_RDWR, 0644)
    if err != nil {
        log.Fatal(err)
    }
    defer disk.Close()

    root := &ToyNode{inodeNum: 0}

    server, err := fs.Mount(mountPoint, root, &fs.Options{
        MountOptions: fuse.MountOptions{
            Debug: true,   // remove when it's working
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    server.Wait()
}
```

---

## Phase 4 — Reading Files

```go
// ToyFileHandle is returned by Open and carries state for Read/Write.
type ToyFileHandle struct {
    inodeNum uint32
}

func (n *ToyNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
    ino, err := readInode(n.inodeNum)
    if err != nil || ino.InUse == 0 {
        return nil, 0, syscall.ENOENT
    }
    return &ToyFileHandle{inodeNum: n.inodeNum}, fuse.FOPEN_DIRECT_IO, 0
}

func (fh *ToyFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
    ino, err := readInode(fh.inodeNum)
    if err != nil {
        return nil, syscall.EIO
    }

    blockIdx := uint32(off) / BlockSize
    blockOff := uint32(off) % BlockSize
    lba      := ino.Blocks[blockIdx]

    buf := make([]byte, BlockSize)
    disk.ReadAt(buf, int64(lba)*BlockSize)

    n := copy(dest, buf[blockOff:])
    return fuse.ReadResultData(dest[:n]), 0
}
```

---

## Phase 5 — Writing Files

```go
func (n *ToyNode) Create(ctx context.Context, name string, flags uint32,
    mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {

    inodeNum, err := allocInode()
    if err != nil {
        return nil, nil, 0, syscall.ENOSPC
    }
    blockNum, err := allocBlock()
    if err != nil {
        return nil, nil, 0, syscall.ENOSPC
    }

    newInode := Inode{Mode: mode | syscall.S_IFREG, InUse: 1}
    newInode.Blocks[0] = blockNum
    writeInode(inodeNum, newInode)

    insertDirEntry(n.inodeNum, name, inodeNum)

    out.Mode = newInode.Mode
    child := n.NewInode(ctx, &ToyNode{inodeNum: inodeNum}, fs.StableAttr{
        Mode: newInode.Mode,
        Ino:  uint64(inodeNum),
    })
    return child, &ToyFileHandle{inodeNum: inodeNum}, fuse.FOPEN_DIRECT_IO, 0
}

func (fh *ToyFileHandle) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
    ino, _ := readInode(fh.inodeNum)

    blockIdx := uint32(off) / BlockSize
    blockOff := uint32(off) % BlockSize
    lba      := ino.Blocks[blockIdx]

    // Read-modify-write the target block
    buf := make([]byte, BlockSize)
    disk.ReadAt(buf, int64(lba)*BlockSize)
    n := copy(buf[blockOff:], data)
    disk.WriteAt(buf, int64(lba)*BlockSize)

    // Update size if we grew the file
    newSize := uint32(off) + uint32(n)
    if newSize > ino.Size {
        ino.Size = newSize
        writeInode(fh.inodeNum, ino)
    }
    return uint32(n), 0
}
```

---

## Phase 6 — Full Shell Usable

```go
// Implement these methods on ToyNode:
func (n *ToyNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno)
func (n *ToyNode) Rmdir(ctx context.Context, name string) syscall.Errno
func (n *ToyNode) Unlink(ctx context.Context, name string) syscall.Errno
func (n *ToyNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno

// Implement on the root fs object for df support:
func (n *ToyNode) Statfs(ctx context.Context, out *fuse.StatfsOut) syscall.Errno
```

---

## Stretch — Indirect Block Pointers

```go
type Inode struct {
    ...
    Blocks   [MaxDirectBlocks]uint32
    Indirect uint32   // points to a [1024]uint32 block of further LBAs
}

// In Read/Write, after exhausting direct blocks:
if blockIdx >= MaxDirectBlocks {
    var indirectTable [BlockSize / 4]uint32
    readBlock(ino.Indirect, &indirectTable)
    lba = indirectTable[blockIdx - MaxDirectBlocks]
}
```

---

## Go-Specific Gotchas

```go
func init() {
    if binary.Size(Inode{}) != 96 { // whatever your expected size is
        panic("Inode size mismatch — on-disk layout is broken")
    }
}
```
