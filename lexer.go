package lexer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type StateFunc func(*L) StateFunc

type TokenType int

const (
	EOFToken TokenType = -1
	ErrorToken TokenType = 0
)

type Token struct {
	Type  TokenType
	Value string
	Start int
	End int
}

type L struct {
	Input          string
	Start, Position int
	StartState      StateFunc
	Err             error
	Tokens          chan Token
	ErrorHandler    func(e string)
	Rewind          runeStack
	StateRecord     stateStack
}

func (t Token) String() string {
	switch t.Type {
	case EOFToken:
		return "EOF"
	case ErrorToken:
		return t.Value
	}
	if len(t.Value) > 10 {
		return fmt.Sprintf("%.10q...", t.Value)
	} else {
		return fmt.Sprintf("%q", t.Value)
	}
}

// New creates a returns a lexer ready to parse the given Input code.
func New(src string, Start StateFunc) *L {
	l := &L{
		Input:     src,
		StartState: Start,
		Start:      0,
		Position:   0,
		Rewind:     NewRuneStack(),
		StateRecord: NewStateStack(),
	}
	return l
}

// Start begins executing the Lexer in an asynchronous manner (using a goroutine).
func (l *L) RunLexer() {
	// Take half the string length as a buffer size.
	buffSize := len(l.Input) / 2
	if buffSize <= 0 {
		buffSize = 1
	}
	l.Tokens = make(chan Token, buffSize)
	go l.run()
}

func (l *L) RunLexerSync() {
	// Take half the string length as a buffer size.
	buffSize := len(l.Input) / 2
	if buffSize <= 0 {
		buffSize = 1
	}
	l.Tokens = make(chan Token, buffSize)
	l.run()
}

// Current returns the value being analyzed at this moment.
func (l *L) Current() string {
	return l.Input[l.Start:l.Position]
}

// Emit will receive a token type and push a new token with the current analyzed
// value into the Tokens channel.
func (l *L) Emit(t TokenType) {
	tok := Token{
		Type:  t,
		Value: l.Current(),
		Start: l.Start,
		End: l.Position,
	}
	l.Tokens <- tok
	l.Start = l.Position
	l.Rewind.Clear()
}

// Ignore clears the Rewind stack and then sets the current beginning Position
// to the current Position in the Input, which effectively ignores the section
// of the Input being analyzed.
func (l *L) Ignore() {
	l.Start = l.Position
	l.Rewind.Clear()
}

// IgnoreCharacter removes the current character from the output
func (l *L) IgnoreCharacter() {
	r := l.Rewind.Pop()
	width := utf8.RuneLen(r)
	l.Input = l.Input[:l.Position - width] + l.Input[l.Position:]
	l.Position -= width
}

// Peek performs a Next operation immediately followed by a Backup returning the
// peeked rune.
func (l *L) Peek() rune {
	r := l.Next()
	l.Backup()

	return r
}

// PeekMany performs n Next operations immediately followed by n Backup operations
// returning the last peeked rune.
func (l *L) PeekMany(n int) rune {
	var r rune
	for i := n; i > 0; i-- {
		r = l.Next()
	}
	for i := n; i > 0; i-- {
		l.Backup()
	}

	return r
}

// Backup will take the last rune read (if any) and back up. Backups can
// occur more than once per call to Next, but you can never Backup past the
// last point a token was emitted.
func (l *L) Backup() bool {
	r := l.Rewind.Pop()
	if r > rune(EOFToken) {
		size := utf8.RuneLen(r)
		l.Position -= size
		if l.Position < l.Start {
			l.Position = l.Start
			return true
		}
	}
	return false
}

// Next pulls the next rune from the Lexer and returns it, moving the Position
// forward in the Input.
func (l *L) Next() rune {
	var (
		r rune
		s int
	)
	str := l.Input[l.Position:]
	if len(str) == 0 {
		r, s = rune(EOFToken), 0
	} else {
		r, s = utf8.DecodeRuneInString(str)
	}
	l.Position += s
	l.Rewind.Push(r)

	return r
}

// Take receives a string containing all acceptable characters and will take the next rune
// if it matches an acceptable character
func (l *L) Take(chars string) bool {
	if strings.ContainsRune(chars, l.Next()) {
		return true
	}
	l.Backup()
	return false
}

// TakeMany receives a string containing all acceptable characters and will continue
// over each rune until it finds an unacceptable rune
func (l *L) TakeMany(chars string) {
	r := l.Next()
	for strings.ContainsRune(chars, r) {
		r = l.Next()
	}
	l.Backup() // last next wasn't a match
}

// TakePattern receives a regex pattern and will take the next rune if it matches the pattern
func (l *L) TakePattern(p *regexp.Regexp) bool {
	r := l.Next()
	if p.MatchString(string(r)) {
		return true
	}
	l.Backup()
	return false
}

// TakeManyPattern receives a regex pattern and will continue over each rune until
// a non-match is found
func (l *L) TakeManyPattern(p *regexp.Regexp) {
	r := l.Next()
	for p.MatchString(string(r)) {
		r = l.Next()
	}
	l.Backup()
}

// NextToken returns the next token from the lexer and a value to denote whether
// or not the token is finished.
func (l *L) NextToken() (*Token, bool) {
	if tok, ok := <-l.Tokens; ok {
		return &tok, false
	} else {
		return nil, true
	}
}

// Partial yyLexer implementation

func (l *L) Error(e string) {
	if l.ErrorHandler != nil {
		l.Err = errors.New(e)
		l.ErrorHandler(e)
	} else {
		panic(e)
	}
}

// Private methods

func (l *L) run() {
	state := l.StartState
	for state != nil {
		state = state(l)
	}
	close(l.Tokens)
}
