package srceditor

import (
	"bytes"
	"go/format"
	"go/token"
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

func (nl *NodeList) Add(nodes ...Node) {
	if nl.list == nil {
		nl.list = make([]Node, 0, DefaultArrayCap)
	}
	nl.list = append(nl.list, nodes...)
}

func (nl *NodeList) Format() ([]byte, error) {
	var buf bytes.Buffer
	var pos token.Pos

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
