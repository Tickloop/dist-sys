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
- [x] Create a 10MB blank image with `dd`
<!-- - [ ] Attach as a loop device with `losetup` -->
<!-- - [ ] Verify with `lsblk` and `stat` -->

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

→ See [`toy-impl.md` — Phase 1](toy-impl.md#phase-1--on-disk-structs-fsgo)

**Note on `binary.Size`:** call `binary.Size(Inode{})` once at startup and assert
it equals what you expect. If you add a field and forget to check, your inode
offsets will silently be wrong.

---

## Phase 2 — mkfs

> Goal: write initial structures onto the image. Run once before mounting.

→ See [`toy-impl.md` — Phase 2](toy-impl.md#phase-2--mkfs-mkmaingo)

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

→ See [`toy-impl.md` — Phase 3](toy-impl.md#phase-3--fuse-driver-read-only-toyfsgo-maingo)

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

→ See [`toy-impl.md` — Phase 4](toy-impl.md#phase-4--reading-files)

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

→ See [`toy-impl.md` — Phase 5](toy-impl.md#phase-5--writing-files)

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

→ See [`toy-impl.md` — Phase 6](toy-impl.md#phase-6--full-shell-usable)

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

→ See [`toy-impl.md` — Stretch: Indirect Block Pointers](toy-impl.md#stretch--indirect-block-pointers)

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
→ See [`toy-impl.md` — Go-Specific Gotchas](toy-impl.md#go-specific-gotchas)

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
