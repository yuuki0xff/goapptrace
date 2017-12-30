package srceditor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"sort"
)

const DefaultArrayCap = 1000
const DefaultBufferCap = 1 << 18 // about 256KiB

type Node interface{}

type InsertNode struct {
	Node
	Pos token.Pos
	Src []byte
}

type DeleteNode struct {
	Node
	Pos token.Pos
	End token.Pos
}

type NodeList struct {
	OrigSrc []byte
	File    *ast.File
	list    []Node
}

type NodeSorter []Node

func (n NodeSorter) Len() int {
	return len(n)
}
func (n NodeSorter) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}
func (n NodeSorter) Less(i, j int) bool {
	return n.topos(i) < n.topos(j)
}
func (n NodeSorter) topos(i int) int {
	switch node := n[i].(type) {
	case *InsertNode:
		return int(node.Pos)
	case *DeleteNode:
		return int(node.Pos)
	}
	panic("Unreachable")
}

func (nl *NodeList) Add(nodes ...Node) {
	if nl.list == nil {
		nl.list = make([]Node, 0, DefaultArrayCap)
	}
	nl.list = append(nl.list, nodes...)
}

func (nl *NodeList) Format() ([]byte, error) {
	var buf bytes.Buffer
	pos := token.Pos(1)

	sort.Sort(NodeSorter(nl.list))

	buf.Grow(len(nl.OrigSrc) + DefaultBufferCap)
	for _, node_ := range nl.list {
		switch node := node_.(type) {
		case *InsertNode:
			buf.Write(nl.srcByRange2(pos, node.Pos))
			buf.Write(node.Src)
			pos = node.Pos
		case *DeleteNode:
			pos = node.End
		default:
			panic(fmt.Sprintf("Unreachable: %+v", node))
		}
	}
	buf.Write(nl.srcByRange1(pos))

	return format.Source(buf.Bytes())
}

func (nl *NodeList) srcByRange1(a token.Pos) []byte {
	basePos := nl.File.Pos() - 1
	start := basePos + a - 1
	return nl.OrigSrc[start:]
}

func (nl *NodeList) srcByRange2(a, b token.Pos) []byte {
	basePos := nl.File.Pos() - 1
	start := basePos + a - 1
	end := basePos + b - 1
	return nl.OrigSrc[start:end]
}
