package sampler

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func Print(w io.Writer, s *Sampler) {
	var t trie
	for _, safePoint := range s.samples {
		stackIds := make([]int, 0, len(safePoint))
		for stackID := range safePoint {
			stackIds = append(stackIds, int(stackID))
		}
		sort.Ints(stackIds)
		for _, stackID := range stackIds {
			sample := safePoint[int32(stackID)]
			t.put(reverse(sample.Stack.InstructionPointers64()), sample.Count)
		}
	}
	t.walk(func(i int, n *node) {
		frameName, _ := s.sym.resolve(n.data)
		_, _ = fmt.Fprintf(w, "%s[%d] %s\n", padding(i), n.count, frameName)
	})
}

func reverse(s []uint64) []uint64 {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func padding(x int) string {
	var s strings.Builder
	for i := 0; i < x; i++ {
		s.WriteString("\t")
	}
	return s.String()
}

type trie struct {
	nodes []*node
}

type node struct {
	data  uint64
	count uint64
	trie
}

func (t *trie) put(b []uint64, c uint64) {
	if len(b) == 0 {
		return
	}
	for _, n := range t.nodes {
		if n.data == b[0] {
			n.count += c
			if len(b) > 1 {
				n.trie.put(b[1:], c)
			}
			return
		}
	}
	n := &node{data: b[0], count: c}
	if len(b) > 1 {
		n.trie.put(b[1:], c)
	}
	t.nodes = append(t.nodes, n)
}

func (t *trie) walk(f func(int, *node)) {
	for i, n := range t.nodes {
		if i > 0 {
			f(i-1, n)
		} else {
			f(i, n)
		}
		n.trie.walk(func(i int, n *node) {
			f(i+1, n)
		})
	}
}
