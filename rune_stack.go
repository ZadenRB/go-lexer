package lexer

type runeNode struct {
	r    rune
	next *runeNode
}

type runeStack struct {
	start *runeNode
}

func NewRuneStack() runeStack {
	return runeStack{}
}

func (s *runeStack) Push(r rune) {
	node := &runeNode{r: r}
	if s.start == nil {
		s.start = node
	} else {
		node.next = s.start
		s.start = node
	}
}

func (s *runeStack) Pop() rune {
	if s.start == nil {
		return rune(EOFToken)
	} else {
		n := s.start
		s.start = n.next
		return n.r
	}
}

func (s *runeStack) Clear() {
	s.start = nil
}
