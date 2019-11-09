package lexer

type stateNode struct {
	f    StateFunc
	next *stateNode
}

type stateStack struct {
	start *stateNode
}

func NewStateStack() stateStack {
	return stateStack{}
}

func (s *stateStack) Push(f StateFunc) {
	node := &stateNode{f: f}
	if s.start == nil {
		s.start = node
	} else {
		node.next = s.start
		s.start = node
	}
}

func (s *stateStack) Pop() StateFunc {
	if s.start == nil {
		return nil
	} else {
		n := s.start
		s.start = n.next
		return n.f
	}
}

func (s *stateStack) Clear() {
	s.start = nil
}
