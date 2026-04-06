# Toy Filesystem in Go — Implementation Roadmap

A ground-up guide to building a FUSE-based ext2-like filesystem in Go.
Uses [`github.com/hanwen/go-fuse/v2`](https://github.com/hanwen/go-fuse) — the most actively maintained Go FUSE library.

---

## Project Layout

```
toyfs/
├── go.mod
├── main.go          — mounts the filesystem, wires up the FUSE server
├── fs.go            — on-disk structs + disk read/write helpers
├── mkfs/
│   └── main.go      — mkfs tool, run once to format the image
└── toyfs.go         — FUSE interface implementation
```

```bash
go mod init toyfs
go get github.com/hanwen/go-fuse/v2
```

---

## Checklist

**Phase 0 — Virtual Disk**
- [ ] Create a 10MB blank image with `dd`
- [ ] Attach as a loop device with `losetup`
- [ ] Verify with `lsblk` and `stat`

**Phase 1 — On-Disk Format**
- [ ] Define `Superblock`, `Inode`, `DirEntry` structs with fixed-size types only
- [ ] Implement `readBlock` / `writeBlock` helpers
- [ ] Implement `readInode` / `writeInode` helpers
- [ ] Assert `binary.Size(Inode{})` matches expected byte count

**Phase 2 — mkfs**
- [ ] Write superblock to block 0
- [ ] Write bitmap to block 1 (mark reserved blocks used)
- [ ] Write root inode (inode 0) to inode table
- [ ] Write root directory data block with `.` and `..` entries
- [ ] Verify with `xxd` that magic number and entries are correct

**Phase 3 — FUSE Driver: Read-Only**
- [ ] Implement `Getattr` (stat support)
- [ ] Implement `Lookup` (path resolution)
- [ ] Implement `Readdir` (ls support)
- [ ] Mount and verify `ls /mnt/toy` works

**Phase 4 — Reading Files**
- [ ] Implement `Open` returning a `ToyFileHandle`
- [ ] Implement `Read` on `ToyFileHandle`
- [ ] Plant a test file on the raw image and verify `cat` works

**Phase 5 — Writing Files**
- [ ] Implement `allocBlock()` (scan bitmap, flip bit, persist)
- [ ] Implement `allocInode()` (scan inode table for free slot)
- [ ] Implement `insertDirEntry()` (add entry to parent dir block)
- [ ] Implement `Create` on `ToyNode`
- [ ] Implement `Write` on `ToyFileHandle` (read-modify-write + size update)
- [ ] Verify `echo "hello" > /mnt/toy/test.txt` and `cat` round-trips

**Phase 6 — Full Shell Usable**
- [ ] Implement `Mkdir`
- [ ] Implement `Rmdir`
- [ ] Implement `Unlink`
- [ ] Implement `Rename`
- [ ] Implement `Statfs` (for `df` support)
- [ ] Smoke test: mkdir, echo, cat, mv, rm, rmdir, df all pass

**Stretch Goals**
- [ ] Indirect block pointers (files > 48KB)
- [ ] Journaling (crash-safe metadata writes)
- [ ] `cmd/fsck` tool (walk inodes, verify bitmap consistency)

---

## Phase 0 — Virtual Disk

> Goal: a fake "disk" the OS believes is real. No code yet.

```bash
# Create a 10MB blank image
dd if=/dev/zero of=mydisk.img bs=4K count=2560

# Attach as a loop device (Linux only — macOS skip this, use the file directly)
sudo losetup /dev/loop0 mydisk.img

# Verify
lsblk | grep loop0
stat mydisk.img
```

**Debugging tools:**

| Tool | What it shows |
|---|---|
| `xxd mydisk.img \| head -4` | Raw hex — all zeros before mkfs |
| `stat mydisk.img` | Confirm exactly 10MB |
| `lsblk` | Loop device visible as block device |

---

## Phase 1 — On-Disk Format

> Goal: define your data structures. These map directly to raw bytes on disk.

The key constraint in Go: structs written to disk must be fixed-size and use
`encoding/binary` for serialisation. No pointers, no slices, no strings — only
fixed-size integer types.

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

**Note on `binary.Size`:** call `binary.Size(Inode{})` once at startup and assert
it equals what you expect. If you add a field and forget to check, your inode
offsets will silently be wrong.

---

## Phase 2 — mkfs

> Goal: write initial structures onto the image. Run once before mounting.

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

**Debugging tools:**

```bash
go run ./mkfs

# Check magic number at byte 0 (expect DE C0 AD DE in little-endian)
xxd mydisk.img | head -4

# Inspect inode table (block 2)
dd if=mydisk.img bs=4096 skip=2 count=1 | xxd | head -8

# Inspect root dir (block 3) — look for "." and ".."
dd if=mydisk.img bs=4096 skip=3 count=1 | xxd | head -4

# Parse superblock with Go directly
go run ./cmd/inspect  # worth writing a small tool for this
```

---

## Phase 3 — FUSE Driver: Read-Only

> Goal: `ls /mnt/toy` works. Nothing else yet.

`go-fuse` works by embedding `fs.Inode` and implementing method interfaces.
You only need to implement the methods for what you want to support.

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

**Mount and test:**

```bash
mkdir -p /mnt/toy
go run . mydisk.img /mnt/toy

# In another terminal:
ls -la /mnt/toy
stat /mnt/toy

# Unmount
fusermount -u /mnt/toy
```

**Debugging tools:**

```bash
# go-fuse Debug:true prints every operation to stdout — leave it on while building
go run . mydisk.img /mnt/toy  # watch the output as you run ls

# strace from the application side
strace ls /mnt/toy 2>&1 | grep -E "openat|getdents|statx"

# dlv attach for stepping through your FUSE handler
dlv attach $(pgrep toyfs)
```

---

## Phase 4 — Reading Files

> Goal: `cat /mnt/toy/hello.txt` works.

Implement `Open` and `Read` on `ToyNode`. In `go-fuse`, `Read` lives on a
`FileHandle` object you return from `Open` — this is where you stash the inode
number so `Read` doesn't have to re-walk the path.

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

**Debugging tools:**

```bash
# Plant a raw file on the image to test reading before write() is implemented
echo -n "hello toyfs" | dd of=mydisk.img bs=1 seek=$((4096*4)) conv=notrunc
# Then manually update the inode for that file via a Go helper or the inspect tool

cat /mnt/toy/hello.txt

# strace confirms read() goes all the way through
strace -e trace=read cat /mnt/toy/hello.txt
```

---

## Phase 5 — Writing Files

> Goal: `echo "hello" > /mnt/toy/test.txt` works.

Add `Create` and `Write`. This phase forces you to write:

- `allocBlock()` — scan bitmap for a free block, flip the bit, write bitmap back
- `allocInode()` — scan inode table for `InUse == 0`
- `insertDirEntry()` — scan a dir's data block for an empty slot, write the entry

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

**Debugging tools:**

```bash
echo "hello" > /mnt/toy/test.txt
cat /mnt/toy/test.txt

# Verify bytes landed at the right LBA on disk
dd if=mydisk.img bs=4096 skip=<data_lba> count=1 | xxd | head

# Verify inode size field updated
dd if=mydisk.img bs=4096 skip=2 count=1 | xxd | head -16

# Verify bitmap bit flipped for new block
dd if=mydisk.img bs=4096 skip=1 count=1 | xxd | head -4

# df reads your Statfs
df /mnt/toy
```

---

## Phase 6 — Full Shell Usable

> Goal: mkdir, rm, mv all work.

```go
// Implement these methods on ToyNode:
func (n *ToyNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno)
func (n *ToyNode) Rmdir(ctx context.Context, name string) syscall.Errno
func (n *ToyNode) Unlink(ctx context.Context, name string) syscall.Errno
func (n *ToyNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno

// Implement on the root fs object for df support:
func (n *ToyNode) Statfs(ctx context.Context, out *fuse.StatfsOut) syscall.Errno
```

**Smoke test:**

```bash
mkdir /mnt/toy/subdir
echo "world" > /mnt/toy/subdir/file.txt
cat /mnt/toy/subdir/file.txt
mv /mnt/toy/subdir/file.txt /mnt/toy/moved.txt
rm /mnt/toy/moved.txt
rmdir /mnt/toy/subdir
df /mnt/toy
```

---

## Stretch Goals

### Indirect block pointers

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

### Journaling

Before any write that modifies metadata (bitmap, inode, directory block), append
the intended operation to a journal block. On `Mount`, replay incomplete
transactions from the journal before making the filesystem available.

### `fsck` tool

```bash
go run ./cmd/fsck mydisk.img
```

Walk all inodes. For each live one, verify:
- All `Blocks[]` entries are marked used in the bitmap
- The inode is reachable from the root directory tree
- `Inode.Size` is consistent with the number of allocated blocks

---

## Quick Reference — Debugging Cheatsheet

```bash
# Inspect any raw block
dd if=mydisk.img bs=4096 skip=<N> count=1 | xxd | head

# Parse superblock from Go
go run ./cmd/inspect mydisk.img

# Watch every FUSE operation (set Debug: true in MountOptions)
go run . mydisk.img /mnt/toy

# strace application-side syscalls
strace -e trace=openat,read,write,getdents64 ls /mnt/toy

# Attach delve to a running FUSE process
dlv attach $(pgrep toyfs)

# Clean unmount (always before re-running mkfs)
fusermount -u /mnt/toy      # Linux
umount /mnt/toy             # macOS

# Check it's mounted
cat /proc/mounts | grep toy

# Inspect superblock with Python (no Go needed)
python3 - <<'EOF'
import struct
with open("mydisk.img", "rb") as f:
    raw = f.read(24)
magic, total, inodes, free_b, free_i, bsize = struct.unpack("<IIIIII", raw)
print(f"magic:        0x{magic:08X}")
print(f"total_blocks: {total}")
print(f"inode_count:  {inodes}")
print(f"free_blocks:  {free_b}")
print(f"free_inodes:  {free_i}")
print(f"block_size:   {bsize}")
EOF
```

---

## Go-Specific Gotchas

**`binary.Read` requires fixed-size types only.** No `string`, no `[]byte`, no
`bool`. Use `[N]byte` for names, `uint32`/`uint64` for everything else.

**Check your struct sizes.** Add this to an `init()` or a test:
```go
func init() {
    if binary.Size(Inode{}) != 96 { // whatever your expected size is
        panic("Inode size mismatch — on-disk layout is broken")
    }
}
```

**`go-fuse` calls your methods concurrently.** Your disk reads/writes need a
`sync.RWMutex`. Take a read lock for reads, write lock when modifying the bitmap,
inode table, or directory blocks.

**`fuse.FOPEN_DIRECT_IO`** tells the kernel not to cache your file data. Use this
while building — it means every read goes through your code and you can see what's
happening. Remove it (or switch to `FOPEN_KEEP_CACHE`) once things are working.

---

## Build Order Summary

| Phase | What you implement | Milestone |
|---|---|---|
| 0 | `dd`, `losetup` | Blank disk exists |
| 1 | `fs.go` structs | Schema defined, `binary.Size` checked |
| 2 | `mkfs/main.go` | `xxd` shows magic number |
| 3 | `Getattr`, `Lookup`, `Readdir` | `ls /mnt/toy` works |
| 4 | `Open`, `Read` | `cat` works |
| 5 | `Create`, `Write`, `allocBlock`, `allocInode` | `echo >` works |
| 6 | `Mkdir`, `Rmdir`, `Unlink`, `Rename`, `Statfs` | Full shell usable |
| S1 | Indirect block pointers | Files > 48KB |
| S2 | Journal | Crash safe |
| S3 | `cmd/fsck` | Self-healing |
