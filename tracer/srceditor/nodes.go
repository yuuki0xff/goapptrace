package srceditor

import (
	"bytes"
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
	var pos token.Pos

	sort.Sort(NodeSorter(nl.list))

	buf.Grow(len(nl.OrigSrc) + DefaultBufferCap)
	for _, node_ := range nl.list {
		switch node := node_.(type) {
		case *InsertNode:
			buf.Write(nl.OrigSrc[pos:node.Pos])
			buf.Write(node.Src)
			pos = node.Pos
		case *DeleteNode:
			pos = node.End
		default:
			panic("Unreachable")
		}
	}
	buf.Write(nl.OrigSrc[pos:])

	return format.Source(buf.Bytes())
}
