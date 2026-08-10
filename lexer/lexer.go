package lexer

import (
	"fmt"
	"regexp"

	"github.com/TheSlowpes/gobash/token"
)

// LexError describes a lexical error, such as an unterminated quote or
// substitution. Pos points at the character that opened the unterminated
// construct, not at the EOF where the error was detected.
type LexError struct {
	Message string
	Pos     token.Position
}

func (e *LexError) Error() string {
	return fmt.Sprintf("%s (opened at line %d, col %d)", e.Message, e.Pos.Line, e.Pos.Col)
}

// quoteCtx tracks one open quote/substitution during word scanning.
// kind is one of 'S' (single-quote), 'D' (double-quote), 'K' (backtick),
// 'P' ($()), or 'B' (${...}).
type quoteCtx struct {
	kind byte
	pos  token.Position
}

func unterminatedMessage(kind byte) string {
	switch kind {
	case 'S':
		return "unterminated single quote"
	case 'D':
		return "unterminated double quote"
	case 'K':
		return "unterminated backtick"
	case 'P':
		return "unterminated $( ) command substitution"
	case 'B':
		return "unterminated ${ } parameter expansion"
	default:
		return "unterminated quoted construct"
	}
}

// Lexer scans bash source text into a stream of tokens. It preserves
// quoting and substitution syntax verbatim inside WORD literals.
type Lexer struct {
	input   string
	pos     int //position of ch in input
	readPos int // position after ch
	ch      byte

	line int
	col  int

	lastErr *LexError
}

// New creates a Lexer over the given source text.
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, col: 0}
	l.readChar()
	return l
}

func (l *Lexer) LastError() *LexError {
	return l.lastErr
}

func (l *Lexer) readChar() {
	if l.ch == '\n' {
		l.line++
		l.col = 0
	}
	if l.readPos >= len(l.input) {
		l.ch = 0 // ASCII code for NUL, signifies end of input
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.col++
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) currentPos() token.Position {
	return token.Position{Line: l.line, Col: l.col, Offset: l.pos}
}

func (l *Lexer) skipHorizontalSpace() {
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}
}

// NextToken returns the next token in the stream.
func (l *Lexer) NextToken() token.Token {
	l.skipHorizontalSpace()

	for l.ch == '#' {
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
		l.skipHorizontalSpace()
	}

	pos := l.currentPos()

	switch {
	case l.ch == 0:
		return token.Token{Type: token.EOF, Pos: pos}
	case l.ch == '\n':
		l.readChar()
		return token.Token{Type: token.NEWLINE, Literal: "\n", Pos: pos}
	case l.ch == '\\' && l.peekChar() == '\n':
		l.readChar()
		l.readChar()
		return l.NextToken()
	}

	if tok, ok := l.tryIONumber(pos); ok {
		return tok
	}
	if tok, ok := l.tryOperator(pos); ok {
		return tok
	}

	lit, lexErr := l.scanWord()
	if lexErr != nil {
		l.lastErr = lexErr
		return token.Token{Type: token.ILLEGAL, Literal: lit, Pos: pos}
	}
	if lit == "" {
		ch := l.ch
		l.readChar()
		return token.Token{Type: token.ILLEGAL, Literal: string(ch), Pos: pos}
	}
	if assignmentPattern.MatchString(lit) {
		return token.Token{Type: token.ASSIGNMENT_WORD, Literal: lit, Pos: pos}
	}
	return token.Token{Type: token.WORD, Literal: lit, Pos: pos}
}

var assignmentPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// tryIONumber checks whether the digits starting at the current position
// are immediately followed by < or > with no other characters between,
// per POSIX IO_NUMBER rules (e.g. the "2" in "2>&1").
func (l *Lexer) tryIONumber(pos token.Position) (token.Token, bool) {
	if l.ch < '0' || l.ch > '9' {
		return token.Token{}, false
	}
	i := l.pos
	for i < len(l.input) && l.input[i] >= '0' && l.input[i] <= '9' {
		i++
	}
	if i < len(l.input) && (l.input[i] == '<' || l.input[i] == '>') {
		lit := l.input[l.pos:i]
		for l.pos < i {
			l.readChar()
		}
		return token.Token{Type: token.IO_NUMBER, Literal: lit, Pos: pos}, true
	}
	return token.Token{}, false
}

func (l *Lexer) tryOperator(pos token.Position) (token.Token, bool) {
	two := string([]byte{l.ch, l.peekChar()})
	switch two {
	case "&&":
		l.readChar()
		l.readChar()
		return token.Token{Type: token.AND_IF, Literal: two, Pos: pos}, true
	case "||":
		l.readChar()
		l.readChar()
		return token.Token{Type: token.OR_IF, Literal: two, Pos: pos}, true
	case ";;":
		l.readChar()
		l.readChar()
		return token.Token{Type: token.DSEMI, Literal: two, Pos: pos}, true
	case "<<":
		l.readChar()
		l.readChar()
		if l.ch == '-' {
			l.readChar()
			return token.Token{Type: token.DLESSDASH, Literal: two, Pos: pos}, true
		}
		return token.Token{Type: token.DLESS, Literal: two, Pos: pos}, true
	case ">>":
		l.readChar()
		l.readChar()
		return token.Token{Type: token.DGREAT, Literal: two, Pos: pos}, true
	case "<&":
		l.readChar()
		l.readChar()
		return token.Token{Type: token.LESSAND, Literal: two, Pos: pos}, true
	case ">&":
		l.readChar()
		l.readChar()
		return token.Token{Type: token.GREATAND, Literal: two, Pos: pos}, true
	case "<>":
		l.readChar()
		l.readChar()
		return token.Token{Type: token.LESSGREAT, Literal: two, Pos: pos}, true
	case ">|":
		l.readChar()
		l.readChar()
		return token.Token{Type: token.CLOBBER, Literal: two, Pos: pos}, true
	}

	switch l.ch {
	case '&':
		l.readChar()
		return token.Token{Type: token.AMP, Literal: "&", Pos: pos}, true
	case ';':
		l.readChar()
		return token.Token{Type: token.SEMI, Literal: ";", Pos: pos}, true
	case '|':
		l.readChar()
		return token.Token{Type: token.PIPE, Literal: "|", Pos: pos}, true
	case '(':
		l.readChar()
		return token.Token{Type: token.LPAREN, Literal: "(", Pos: pos}, true
	case ')':
		l.readChar()
		return token.Token{Type: token.RPAREN, Literal: ")", Pos: pos}, true
	case '<':
		l.readChar()
		return token.Token{Type: token.LESS, Literal: "<", Pos: pos}, true
	case '>':
		l.readChar()
		return token.Token{Type: token.GREAT, Literal: ">", Pos: pos}, true
	}
	return token.Token{}, false
}

// scanWord consumes a WORD token, preserving quote/ expansion syntax
// verbatim. It tracks a small context stack so that whitespace and
// operator characters inside quotes or $()/${}/“ substitutions do not
// terminate the word.
func (l *Lexer) scanWord() (string, *LexError) {
	var out []byte
	var stack []quoteCtx

	for l.ch != 0 {
		top := byte(0)
		if len(stack) > 0 {
			top = stack[len(stack)-1].kind
		}

		switch top {
		case 'S': // single-quoted: fully literal
			out = append(out, l.ch)
			if l.ch == '\'' {
				stack = stack[:len(stack)-1]
			}
			l.readChar()
		case 'D': // double-quoted
			switch {
			case l.ch == '\\':
				out = append(out, l.ch)
				l.readChar()
				if l.ch != 0 {
					out = append(out, l.ch)
					l.readChar()
				}
			case l.ch == '"':
				out = append(out, l.ch)
				stack = stack[:len(stack)-1]
				l.readChar()
			case l.ch == '$' && l.peekChar() == '(':
				openingPos := l.currentPos()
				out = append(out, l.ch)
				l.readChar()
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'P', pos: openingPos})
				l.readChar()
			case l.ch == '$' && l.peekChar() == '{':
				openingPos := l.currentPos()
				out = append(out, l.ch)
				l.readChar()
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'B', pos: openingPos})
				l.readChar()
			case l.ch == '`':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'K', pos: l.currentPos()})
				l.readChar()
			default:
				out = append(out, l.ch)
				l.readChar()
			}

		case 'P': // $( ... ) command / arithmetic substitution
			switch l.ch {
			case '\'':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'S', pos: l.currentPos()})
				l.readChar()
			case '"':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'D', pos: l.currentPos()})
				l.readChar()
			case '`':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'K', pos: l.currentPos()})
				l.readChar()
			case '(':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'P', pos: l.currentPos()})
				l.readChar()
			case ')':
				out = append(out, l.ch)
				stack = stack[:len(stack)-1]
				l.readChar()
			default:
				out = append(out, l.ch)
				l.readChar()
			}
		case 'B': // ${ ... } parameter expansion
			switch l.ch {
			case '\'':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'S', pos: l.currentPos()})
				l.readChar()
			case '"':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'D', pos: l.currentPos()})
				l.readChar()
			case '`':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'K', pos: l.currentPos()})
				l.readChar()
			case '{':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'B', pos: l.currentPos()})
				l.readChar()
			case '}':
				out = append(out, l.ch)
				stack = stack[:len(stack)-1]
				l.readChar()
			default:
				out = append(out, l.ch)
				l.readChar()
			}

		case 'K': // legacy backtick command substitution
			out = append(out, l.ch)
			if l.ch == '\\' {
				l.readChar()
				if l.ch != 0 {
					out = append(out, l.ch)
					l.readChar()
				}
			} else {
				if l.ch == '`' {
					stack = stack[:len(stack)-1]
				}
				l.readChar()
			}

		default: // top-level, unquoted
			if isWordBoundary(l.ch) {
				return string(out), nil
			}
			switch l.ch {
			case '\\':
				if l.peekChar() == '\n' {
					l.readChar() // consume backslash
					l.readChar() // consume newline
					continue
				}
				out = append(out, l.ch)
				l.readChar()
				if l.ch != 0 {
					out = append(out, l.ch)
					l.readChar()
				}
			case '\'':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'S', pos: l.currentPos()})
				l.readChar()
			case '"':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'D', pos: l.currentPos()})
				l.readChar()
			case '$':
				openingPos := l.currentPos()
				out = append(out, l.ch)
				l.readChar()
				switch l.ch {
				case '(':
					out = append(out, l.ch)
					stack = append(stack, quoteCtx{kind: 'P', pos: openingPos})
					l.readChar()
				case '{':
					out = append(out, l.ch)
					stack = append(stack, quoteCtx{kind: 'B', pos: openingPos})
					l.readChar()
				}
			case '`':
				out = append(out, l.ch)
				stack = append(stack, quoteCtx{kind: 'K', pos: l.currentPos()})
				l.readChar()
			default:
				out = append(out, l.ch)
				l.readChar()
			}
		}
	}

	if len(stack) > 0 {
		top := stack[len(stack)-1]
		return "", &LexError{Message: unterminatedMessage(top.kind), Pos: top.pos}
	}

	return string(out), nil
}

func isWordBoundary(b byte) bool {
	switch b {
	case 0, ' ', '\t', '\n', '&', '|', ';', '<', '>', '(', ')':
		return true
	}
	return false
}
