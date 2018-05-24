package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/lpar/bytesize"
)

// DirInfo stores the amount of file bloat under a single directory
type DirInfo struct {
	Path  string
	Bytes int64
}

// Bloat stores the amount of bloat found
type Bloat struct {
	DirMap map[string]*DirInfo
	Dirs   []*DirInfo
	Abs    bool
}

// NewBloat returns
func NewBloat(absmode bool) *Bloat {
	return &Bloat{DirMap: make(map[string]*DirInfo), Abs: absmode}
}

// Sort sorts the data in the DirMap map and places it in the Dirs slice,
// with the biggest bloatiest directories at the top
func (b *Bloat) Sort() {
	b.Dirs = make([]*DirInfo, 0, len(b.DirMap))
	for _, info := range b.DirMap {
		b.Dirs = append(b.Dirs, info)
	}
	sort.Slice(b.Dirs, func(x, y int) bool { return b.Dirs[x].Bytes > b.Dirs[y].Bytes })
}

// AddBloat adds the specified number of bytes of bloat to the total for the specified
// directory, adding new map entries to the DirMap as necessary.
func (b *Bloat) AddBloat(dir string, bytes int64) {
	info, ok := b.DirMap[dir]
	if !ok {
		info = &DirInfo{Path: dir, Bytes: bytes}
		b.DirMap[dir] = info
		return
	}
	info.Bytes += bytes
}

// AddFile adds the bloat from a single file to the total for the file's directory
// and all parent directores of that directory
func (b *Bloat) AddFile(path string, bytes int64) {
	dir := path
	ldir := dir
	for {
		dir = filepath.Dir(dir)
		if ldir == dir {
			break
		}
		b.AddBloat(dir, bytes)
		ldir = dir
	}
}

// Scan walks all files under the specified base dir, and totals their sizes into the Bloat
func (b *Bloat) Scan(basedir string) {
	werr := filepath.Walk(basedir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path)
		var fdir string
		var perr error
		if b.Abs {
			fdir, perr = filepath.Abs(path)
		} else {
			fdir, perr = filepath.Rel(basedir, path)
		}
		if perr != nil {
			fmt.Fprintf(os.Stderr, "can't process %s: %v\n", path, perr)
			panic(err)
		}
		b.AddFile(fdir, f.Size())
		return nil
	})
	if werr != nil {
		fmt.Fprintf(os.Stderr, "error scanning %s: %v\n", basedir, werr)
	}
}

// Report outputs the results of the scan
func (b *Bloat) Report() {
	for _, info := range b.Dirs {
		bs := bytesize.FormatBytes(info.Bytes, 10, 0)
		fmt.Printf("%6s %s\n", bs, info.Path)
	}
}

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}
	helpp := os.Args[1]
	if helpp == "--help" || helpp == "-h" || helpp == "/?" {
		help()
		return
	}
	absmode := len(os.Args) > 2
	bloat := NewBloat(absmode)
	for _, arg := range os.Args[1:] {
		bloat.Scan(arg)
	}
	bloat.Sort()
	bloat.Report()
}

func help() {
	fmt.Printf("Usage: %s [DIR]...\n", filepath.Base(os.Args[0]))
	fmt.Println("Summarize disk space in use under the specified directory or directories.")
	fmt.Println("Each directory is output along with the total size of all files under that directory.")
	fmt.Println("The most bloated directories are reported first.")
	fmt.Println("With a single DIR, output is displayed as relative directory paths.")
	fmt.Println("With multiple DIRs, all dir paths are made absolute for output, but only data under the")
	fmt.Println("specified DIRs counts towards the totals displayed.")
	fmt.Println("If the DIRs overlap or are repeated, you will get inaccurate output because\nfiles will be counted multiple times.")
	fmt.Println("\nExample invocation:\n\n    bloat ~/Downloads | head -n 10")
}
