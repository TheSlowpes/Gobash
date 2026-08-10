package lexer

import (
	"testing"

	"github.com/TheSlowpes/gobash/token"
)

func lexAll(t *testing.T, input string) []token.Token {
	t.Helper()
	l := New(input)
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.EOF || len(toks) > 1000 {
			break
		}
	}
	return toks
}

func assertTypes(t *testing.T, input string, want []token.TokenType) {
	t.Helper()
	toks := lexAll(t, input)
	if len(toks) != len(want) {
		types := make([]token.TokenType, len(toks))
		for i, tok := range toks {
			types[i] = tok.Type
		}
		t.Fatalf("input %q: got %d tokens %v, wand %d %v", input, len(toks), types, len(want), want)
	}
	for i, w := range want {
		if toks[i].Type != w {
			t.Errorf("input %q: token[%d] = %s, want %s", input, i, toks[i].Type, w)
		}
	}
}

// --- Simple Words ---

func TestNextToken_SimpleWords(t *testing.T) {
	assertTypes(t, "echo hello world", []token.TokenType{token.WORD, token.WORD, token.WORD, token.EOF})
}

func TestNextToken_EmptyInput(t *testing.T) {
	assertTypes(t, "", []token.TokenType{token.EOF})
}

func TestNextToken_WhitespaceOnly(t *testing.T) {
	assertTypes(t, "  \t ", []token.TokenType{token.EOF})
}

// --- Operators ---
func TestNextToken_Operators(t *testing.T) {
	tests := []struct {
		input string
		want  []token.TokenType
	}{
		{"a && b", []token.TokenType{token.WORD, token.AND_IF, token.WORD, token.EOF}},
		{"a || b", []token.TokenType{token.WORD, token.OR_IF, token.WORD, token.EOF}},
		{"a ; b", []token.TokenType{token.WORD, token.SEMI, token.WORD, token.EOF}},
		{"a ;; b", []token.TokenType{token.WORD, token.DSEMI, token.WORD, token.EOF}},
		{"a | b", []token.TokenType{token.WORD, token.PIPE, token.WORD, token.EOF}},
		{"a & b", []token.TokenType{token.WORD, token.AMP, token.WORD, token.EOF}},
		{"(a)", []token.TokenType{token.LPAREN, token.WORD, token.RPAREN, token.EOF}},
		{"a < b", []token.TokenType{token.WORD, token.LESS, token.WORD, token.EOF}},
		{"a > b", []token.TokenType{token.WORD, token.GREAT, token.WORD, token.EOF}},
		{"a << b", []token.TokenType{token.WORD, token.DLESS, token.WORD, token.EOF}},
		{"a <<- b", []token.TokenType{token.WORD, token.DLESSDASH, token.WORD, token.EOF}},
		{"a >> b", []token.TokenType{token.WORD, token.DGREAT, token.WORD, token.EOF}},
		{"a <& b", []token.TokenType{token.WORD, token.LESSAND, token.WORD, token.EOF}},
		{"a >& b", []token.TokenType{token.WORD, token.GREATAND, token.WORD, token.EOF}},
		{"a <> b", []token.TokenType{token.WORD, token.LESSGREAT, token.WORD, token.EOF}},
		{"a >| b", []token.TokenType{token.WORD, token.CLOBBER, token.WORD, token.EOF}},
	}

	for _, tt := range tests {
		assertTypes(t, tt.input, tt.want)
	}
}

// --- IO_NUMBER ---

func TestNextToken_IONumberBeforeRedirect(t *testing.T) {
	toks := lexAll(t, "2>&1")
	want := []token.TokenType{token.IO_NUMBER, token.GREATAND, token.WORD, token.EOF}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(toks), len(want))
	}
	for i, w := range want {
		if toks[i].Type != w {
			t.Errorf("token[%d] = %s, want %s", i, toks[i].Type, w)
		}
	}
	if toks[0].Literal != "2" {
		t.Errorf("IO_NUMBER literal = %q, want \"2\"", toks[0].Literal)
	}
	if toks[2].Literal != "1" {
		t.Errorf("WORD literal = %q, want \"1\"", toks[2].Literal)
	}
}

func TestNextToken_DigitsNotFollowedByRedirectAreWords(t *testing.T) {
	assertTypes(t, "123abc", []token.TokenType{token.WORD, token.EOF})
	assertTypes(t, "5 > file", []token.TokenType{token.WORD, token.GREAT, token.WORD, token.EOF})
}

// --- Assignment words ---
func TestNextToken_AssignmentWord(t *testing.T) {
	toks := lexAll(t, "FOO=bar")
	if toks[0].Type != token.ASSIGNMENT_WORD || toks[0].Literal != "FOO=bar" {
		t.Errorf("got %+v, want ASSIGNMENT_WORD(FOO=bar)", toks[0])
	}
}

func TestNextToken_AssignmentWordEmptyValue(t *testing.T) {
	toks := lexAll(t, "FOO=")
	if toks[0].Type != token.ASSIGNMENT_WORD || toks[0].Literal != "FOO=" {
		t.Errorf("got %+v, want ASSIGNMENT_WORD(FOO=)", toks[0])
	}
}

func TestNextToken_LeadingDigitIsNotAssignment(t *testing.T) {
	toks := lexAll(t, "2FOO=bar")
	if toks[0].Type != token.WORD || toks[0].Literal != "2FOO=bar" {
		t.Errorf("got %+v, want WORD(2FOO=bar)", toks[0])
	}
}

// --- Quoting ---
func TestNextToken_SingleQuotePreservesSpaces(t *testing.T) {
	toks := lexAll(t, `'hello world'`)
	if toks[0].Type != token.WORD || toks[0].Literal != `'hello world'` {
		t.Errorf("got %+v, want WORD('hello world')", toks[0])
	}
	if toks[1].Type != token.EOF {
		t.Errorf("expected single WORD token followed by EOF, got %v", toks)
	}
}

func TestNextToken_DoubleQuoteWithVarSyntax(t *testing.T) {
	toks := lexAll(t, `echo "hello $world"`)
	assertTypes(t, `echo "hello $world"`, []token.TokenType{token.WORD, token.WORD, token.EOF})
	if toks[1].Literal != `"hello $world"` {
		t.Errorf("got literal %q, want %q", toks[1].Literal, `"hello $world"`)
	}
}

func TestNextToken_NestedCommandSubstitutionInDOubleQuotes(t *testing.T) {
	input := `"$(echo $(date))"`
	toks := lexAll(t, input)
	if toks[0].Type != token.WORD || toks[0].Literal != input {
		t.Errorf("got %+v, want WORD(%s)", toks[0], input)
	}
}

func TestNextToken_ParameterExpansionWithBraces(t *testing.T) {
	input := `${name:-default}`
	toks := lexAll(t, input)
	if toks[0].Type != token.WORD || toks[0].Literal != input {
		t.Errorf("got %+v, want WORD(%s)", toks[0], input)
	}
}

// --- Unterminated constructs (error path) ---
func TestNextToken_UnterminatedSingleQuote(t *testing.T) {
	l := New("'hello")
	tok := l.NextToken()
	if tok.Type != token.ILLEGAL {
		t.Fatalf("got %v, want ILLEGAL", tok)
	}
	lexErr := l.LastError()
	if lexErr == nil {
		t.Fatal("expected LastError to be set")
	}
	if lexErr.Message != "unterminated single quote" {
		t.Errorf("message = %q, want %q", lexErr.Message, "unterminated single quote")
	}
	if lexErr.Pos.Line != 1 || lexErr.Pos.Col != 1 {
		t.Errorf("pos = %+v, want opening quote at line 1 col 1", lexErr.Pos)
	}
}

func TestNextToken_UnterminatedDoubleQuote(t *testing.T) {
	l := New(`"hello`)
	tok := l.NextToken()
	if tok.Type != token.ILLEGAL {
		t.Fatalf("got %v, want ILLEGAL", tok)
	}
	lexErr := l.LastError()
	if lexErr == nil {
		t.Fatal("expected LastError to be set")
	}
	if lexErr.Message != "unterminated double quote" {
		t.Errorf("message = %q, want %q", lexErr.Message, "unterminated double quote")
	}
	if lexErr.Pos.Line != 1 || lexErr.Pos.Col != 1 {
		t.Errorf("pos = %+v, want opening quote at line 1 col 1", lexErr.Pos)
	}
}

func TestNextToken_UnterminatedCommandSubstitution(t *testing.T) {
	l := New("$(echo hello")
	tok := l.NextToken()
	if tok.Type != token.ILLEGAL {
		t.Fatalf("got %v, want ILLEGAL", tok)
	}
	lexErr := l.LastError()
	if lexErr == nil {
		t.Fatal("expected LastError to be set")
	}
	if lexErr.Message != "unterminated $( ) command substitution" {
		t.Errorf("message = %q, want %q", lexErr.Message, "unterminated $( ) command substitution")
	}
	if lexErr.Pos.Line != 1 || lexErr.Pos.Col != 1 {
		t.Errorf("pos = %+v, want opening $( at line 1 col 1", lexErr.Pos)
	}
}

func TestNextToken_UnterminatedParameterExpansion(t *testing.T) {
	l := New("${name:-default")
	tok := l.NextToken()
	if tok.Type != token.ILLEGAL {
		t.Fatalf("got %v, want ILLEGAL", tok)
	}
	lexErr := l.LastError()
	if lexErr == nil {
		t.Fatal("expected LastError to be set")
	}
	if lexErr.Message != "unterminated ${ } parameter expansion" {
		t.Errorf("message = %q, want %q", lexErr.Message, "unterminated ${ } parameter expansion")
	}
	if lexErr.Pos.Line != 1 || lexErr.Pos.Col != 1 {
		t.Errorf("pos = %+v, want opening ${ at line 1 col 1", lexErr.Pos)
	}
}

func TestNextToken_UnterminatedBacktick(t *testing.T) {
	l := New("`echo hello")
	tok := l.NextToken()
	if tok.Type != token.ILLEGAL {
		t.Fatalf("got %v, want ILLEGAL", tok)
	}
	lexErr := l.LastError()
	if lexErr == nil {
		t.Fatal("expected LastError to be set")
	}
	if lexErr.Message != "unterminated backtick" {
		t.Errorf("message = %q, want %q", lexErr.Message, "unterminated backtick")
	}
	if lexErr.Pos.Line != 1 || lexErr.Pos.Col != 1 {
		t.Errorf("pos = %+v, want opening backtick at line 1 col 1", lexErr.Pos)
	}
}

func TestNextToken_UnterminatedNestedReportsInnermost(t *testing.T) {
	l := New("$(echo $(date)")
	tok := l.NextToken()
	if tok.Type != token.ILLEGAL {
		t.Fatalf("got %v, want ILLEGAL", tok)
	}
	lexErr := l.LastError()
	if lexErr == nil {
		t.Fatal("expected LastError to be set")
	}
	if lexErr.Message != "unterminated $( ) command substitution" {
		t.Errorf("message = %q, want %q", lexErr.Message, "unterminated $( ) command substitution")
	}
}
