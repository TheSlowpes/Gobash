package token

import "fmt"

type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF

	WORD            // generic word (raw, unexpended, quotes preserved)
	ASSIGNMENT_WORD // NAME=value, unquoted NAME prefix
	IO_NUMBER       // digits immediately preceding < or >
	NEWLINE

	// Multicharacter operators
	AND_IF    // &&
	OR_IF     // ||
	DSEMI     // ;;
	DLESS     // <<
	DGREAT    // >>
	LESSAND   // <&
	GREATAND  // >&
	LESSGREAT // <>
	DLESSDASH // <<-
	CLOBBER   // >

	// Single-character operators
	AMP    // &
	SEMI   // ;
	PIPE   // |
	LPAREN // (
	RPAREN // )
	LESS   // <
	GREAT  // >
)

var typeNames = map[TokenType]string{
	ILLEGAL:         "ILLEGAL",
	EOF:             "EOF",
	WORD:            "WORD",
	ASSIGNMENT_WORD: "ASSIGNMENT_WORD",
	IO_NUMBER:       "IO_NUMBER",
	NEWLINE:         "NEWLINE",
	AND_IF:          "AND_IF",
	OR_IF:           "OR_IF",
	DSEMI:           "DSEMI",
	DLESS:           "DLESS",
	DGREAT:          "DGREAT",
	LESSAND:         "LESSAND",
	DLESSDASH:       "DLESSDASH",
	CLOBBER:         "CLOBBER",
	AMP:             "AMP",
	SEMI:            "SEMI",
	PIPE:            "PIPE",
	LPAREN:          "LPAREN",
	RPAREN:          "RPAREN",
	LESS:            "LESS",
	GREAT:           "GREAT",
}

func (t TokenType) String() string {
	if name, ok := typeNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

type Position struct {
	Line   int
	Col    int
	Offset int
}

type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%s)", t.Type.String(), t.Literal)
}

const (
	KwIf       = "if"
	KwThen     = "then"
	KwElse     = "else"
	KwElif     = "elif"
	KwFi       = "fi"
	KwDo       = "do"
	KwDone     = "done"
	KwCase     = "case"
	KwEsac     = "esac"
	KwWhile    = "while"
	KwUntil    = "until"
	KwFor      = "for"
	KwIn       = "in"
	KwFunction = "function"
	KwLbrace   = "{"
	KwRbrace   = "}"
	KwBang     = "!"
)

var ReservedWords = map[string]bool{
	KwIf: true, KwThen: true, KwElse: true, KwElif: true, KwFi: true,
	KwDo: true, KwDone: true, KwCase: true, KwEsac: true, KwWhile: true,
	KwUntil: true, KwFor: true, KwIn: true, KwFunction: true, KwLbrace: true,
	KwRbrace: true, KwBang: true,
}

func IsReservedWord(word string) bool {
	return ReservedWords[word]
}
