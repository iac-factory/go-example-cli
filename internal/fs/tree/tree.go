package tree

import (
	"cli/internal/fs/checksum"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

type Descriptor string

type Exception error

var (
	ExceptionNilNode              Exception = errors.New("nil node")
	ExceptionInvalidFileNode      Exception = errors.New("invalid file node")
	ExceptionInvalidDirectoryNode Exception = errors.New("invalid directory node")

	ExceptionInvalidDirectory Exception = errors.New("invalid directory")
)

const (
	File      Descriptor = "FILE"
	Directory Descriptor = "DIRECTORY"
	Symbolic  Descriptor = "SYMBOLIC"
)

type Node struct {
	parent *Node            `json:"-" yaml:"-"`
	table  map[string]*Node `json:"-" yaml:"-"`
	depth  int              `json:"-" yaml:"-"`

	content []byte `json:"-" yaml:"-"`

	Path     string     `json:"path" yaml:"path"`
	Dirname  string     `json:"dirname" yaml:"dirname"`
	Name     string     `json:"name" yaml:"name"`
	Type     Descriptor `json:"type" yaml:"type"`
	Checksum *string    `json:"checksum,omitempty" yaml:"checksum,omitempty"`
	Nodes    []Node     `json:"nodes,omitempty" yaml:"nodes,omitempty"`
}

func (n *Node) String() string {
	return n.JSON()
}

func (n *Node) JSON() string {
	buffer, e := json.MarshalIndent(n, "", "    ")
	if e != nil {
		panic(e)
	}

	return string(buffer)
}

func (n *Node) YAML() string {
	buffer, e := yaml.Marshal(n)
	if e != nil {
		panic(e)
	}

	return string(buffer)
}

func (n *Node) Root() *Node {
	if n.parent == nil {
		return n
	} else {
		r := n.parent

		if r != nil {
			return r.Root()
		} else {
			return r
		}
	}
}

func (n *Node) Parent() *Node {
	return n.parent
}

func (n *Node) Permissions() os.FileMode {
	info, e := os.Stat(n.Path)
	if e != nil {
		panic(e)
	}

	return info.Mode().Perm()
}

func (n *Node) Files() []*Node {
	var partials = make([]*Node, 0)
	for _, node := range n.Table() {
		if node.Type == File {
			partials = append(partials, node)
		}
	}

	return partials
}

func (n *Node) Directories() []*Node {
	var partials = make([]*Node, 0)
	for _, node := range n.Table() {
		if node.Type == Directory {
			partials = append(partials, node)
		}
	}

	return partials
}

// URI returns the full-system, absolute path of the Node instance.
func (n *Node) URI() (path string) {
	path, e := filepath.Abs(n.Path)
	if e != nil {
		panic("Invalid Path - Unable to Calculate Full-System, Absolute Path")
	}

	return
}

// Map returns a hash-map of all nodes from the node's absolute root.
func (n *Node) Map() map[string]*Node {
	return n.Root().table
}

// Table returns the current node's hash-map of child nodes.
func (n *Node) Table() map[string]*Node {
	return n.table
}

// Search will search for matching file-system descriptors, and return
// substring matches.
//
//   - Note that the search function will only evaluate the current Node instance's table.
func (n *Node) Search(descriptor string) (nodes []*Node) {
	table := n.Table()
	for key, node := range table {
		if strings.Contains(key, descriptor) {
			nodes = append(nodes, node)
		}
	}

	return
}

// Contents returns a Node of Type File's file contents.
func (n *Node) Contents() ([]byte, error) {
	if n == nil {
		return nil, ExceptionNilNode
	} else if n.Type != File {
		return nil, ExceptionInvalidFileNode
	} else {
		n.read()
	}

	return n.content, nil
}

// Copy will copy the Node instance's directories and files to the destination.
//
//   - Copy will not overwrite existing files.
//   - Copy will not overwrite existing directory or file permissions.
func (n *Node) Copy(destination string) {
	directories := n.Directories()
	files := n.Files()

	for _, directory := range directories {
		target := filepath.Join(destination, directory.Path)
		if e := os.MkdirAll(target, directory.Permissions()); e != nil {
			panic(e)
		}
	}

	for _, file := range files {
		target := filepath.Join(destination, file.Path)
		if _, exception := os.Stat(target); errors.Is(exception, os.ErrNotExist) {
			contents, e := file.Contents()
			if e != nil {
				panic(e)
			}

			if e := os.WriteFile(target, contents, file.Permissions()); e != nil {
				panic(e)
			}
		}
	}
}

// Replicate will copy the Node instance's directories and files to the destination.
//
//   - Replicate will overwrite existing files.
//   - Replicate will not overwrite existing directory or file permissions.
func (n *Node) Replicate(destination string) {
	directories := n.Directories()
	files := n.Files()

	for _, directory := range directories {
		target := filepath.Join(destination, directory.Path)
		if e := os.MkdirAll(target, directory.Permissions()); e != nil {
			panic(e)
		}
	}

	for _, file := range files {
		target := filepath.Join(destination, file.Path)
		contents, e := file.Contents()
		if e != nil {
			panic(e)
		}

		if e := os.WriteFile(target, contents, file.Permissions()); e != nil {
			panic(e)
		}
	}
}

// Replace will copy the Node instance's directories and files to the destination.
//
//   - Replace will overwrite existing files.
//   - Replace will overwrite existing directory and file permissions.
func (n *Node) Replace(destination string) {
	if exists(destination) {
		if e := os.RemoveAll(destination); e != nil {
			panic(e)
		}
	}

	directories := n.Directories()
	files := n.Files()

	for _, directory := range directories {
		target := filepath.Join(destination, directory.Path)
		if e := os.MkdirAll(target, directory.Permissions()); e != nil {
			panic(e)
		}
	}

	for _, file := range files {
		target := filepath.Join(destination, file.Path)
		contents, e := file.Contents()
		if e != nil {
			panic(e)
		}

		if e := os.WriteFile(target, contents, file.Permissions()); e != nil {
			panic(e)
		}
	}
}

// read will read-in the Node file-contents if of Type File.
func (n *Node) read() {
	if n != nil && n.Type == File && n.content == nil {
		buffer, e := os.ReadFile(n.URI())
		if e != nil {
			panic(e)
		}

		n.content = buffer
	}
}

func (n *Node) add(child *Node) {
	child.parent = n
	child.depth = n.depth + 1
	child.table = map[string]*Node{}

	if child.Type == Directory {
		child.walk()
	} else if child.Type == File {
		child.Checksum = checksum.SHA256(child.URI())
	}

	// update root table
	rt := n.Root().table
	if _, valid := rt[child.Path]; !(valid) {
		rt[child.Path] = child
	}

	// update current node table
	nt := n.table
	if _, valid := nt[child.Path]; !(valid) {
		nt[child.Path] = child
	}

	n.Nodes = append(n.Nodes, *child)
}

func (n *Node) walk() {
	entries, e := os.ReadDir(n.Path)
	if e != nil {
		fmt.Printf("error reading %s: %s\n", n.Path, e.Error())
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(n.Path, name)
		dirname := filepath.Dir(path)

		var child = &Node{
			Name:    name,
			Dirname: dirname,
			Path:    path,
			Nodes:   make([]Node, 0),
		}

		if (entry.Type() & os.ModeSymlink) == os.ModeSymlink {
			child.Type = Symbolic
			// dereference, e := os.Readlink(filepath.Join(n.Path, entry.Name()))
			// if e != nil {
			// 	fmt.Printf("error reading link: %s\n", e.Error())
			// } else {
			// 	child.Path = dereference
			// }
		} else if entry.IsDir() {
			child.Type = Directory
		} else {
			child.Type = File
		}

		n.add(child)
	}
}

// exists returns whether the given file or directory exists
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}

	return false
}

func New(path string) *Node {
	descriptor, e := os.Stat(path)
	if e != nil || !(descriptor.IsDir()) {
		panic(ExceptionInvalidDirectory)
	}

	dirname := filepath.Dir(descriptor.Name())
	root := &Node{
		table:  map[string]*Node{},
		parent: nil,
		depth:  0,

		Dirname: dirname,
		Name:    descriptor.Name(),
		Path:    path,
		Type:    Directory,
		Nodes:   make([]Node, 0),
	}

	root.walk()

	return root
}
