package rqlParser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

const (
	// Special tokens
	ILLEGAL Token = iota
	EOF

	// Literals
	IDENT // fields, function names

	// Reserved characters
	SPACE               //
	AMPERSAND           // &
	OPENING_PARENTHESIS // (
	CLOSING_PARENTHESIS // )
	COMMA               // ,
	EQUAL_SIGN          // =
	SLASH               // /
	SEMI_COLON          // ;
	QUESTION_MARK       // ?
	AT_SYMBOL           // @
	PIPE                // |

	// Keywords
	AND
	OR
	EQUAL
	GREATER
	GREATER_OR_EQUAL
	LOWER
	LOWER_OR_EQUAL
	NOT_EQUAL
)

var (
	ReservedRunes []rune = []rune{' ', '&', '(', ')', ',', '=', '/', ';', '?', '@', '|'}
	eof                  = rune(0)
)

type TokenString struct {
	t Token
	s string
}

type Token int

func NewTokenString(t Token, s string) TokenString {
	return TokenString{t: t, s: s}
}

type Scanner struct {
	r *bufio.Reader
}

func NewScanner() *Scanner {
	return &Scanner{}
}

// Scan returns the next token and literal value.
func (s *Scanner) Scan(r io.Reader) (out []TokenString, err error) {
	s.r = bufio.NewReader(r)

	for true {
		tok, lit := s.ScanToken()
		if tok == EOF {
			break
		} else if tok == ILLEGAL {
			return out, fmt.Errorf("Illegal Token : %s", lit)
		} else {
			out = append(out, NewTokenString(tok, lit))
		}
	}

	return
}

func (s *Scanner) ScanToken() (tok Token, lit string) {
	ch := s.read()

	if isReservedRune(ch) {
		s.unread()
		return s.scanReservedRune()
	} else if isIdent(ch) {
		s.unread()
		return s.scanIdent()
	}

	if ch == eof {
		return EOF, ""
	}

	return ILLEGAL, string(ch)
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *Scanner) unread() { _ = s.r.UnreadRune() }

func (s *Scanner) scanReservedRune() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer

	buf.WriteRune(s.read())
	lit = buf.String()

	// Read every subsequent whitespace character into the buffer.
	// Non-whitespace characters and EOF will cause the loop to exit.

	for _, rr := range ReservedRunes {
		if string(rr) == lit {
			switch rr {
			case '&':
				return AMPERSAND, lit
			case '(':
				return OPENING_PARENTHESIS, lit
			case ')':
				return CLOSING_PARENTHESIS, lit
			case ',':
				return COMMA, lit
			case '=':
				return EQUAL_SIGN, lit
			case '/':
				return SLASH, lit
			case ';':
				return SEMI_COLON, lit
			case '?':
				return QUESTION_MARK, lit
			case '@':
				return AT_SYMBOL, lit
			case '|':
				return PIPE, lit
			case eof:
				return EOF, lit
			default:
				return ILLEGAL, lit
			}
		}
	}
	return ILLEGAL, lit
}

func isReservedRune(ch rune) bool {
	for _, rr := range ReservedRunes {
		if ch == rr {
			return true
		}
	}
	return false
}

func isIdent(ch rune) bool {
	return isLetter(ch) || isDigit(ch) || isSpecialChar(ch)
}

func isSpecialChar(ch rune) bool {
	return ch == '*' || ch == '_' || ch == '%' || ch == '+' || ch == '-'
}

// isLetter returns true if the rune is a letter.
func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// isDigit returns true if the rune is a digit.
func isDigit(ch rune) bool { return (ch >= '0' && ch <= '9') }

func (s *Scanner) scanIdent() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent ident character into the buffer.
	// Non-ident characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isIdent(ch) {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	return IDENT, buf.String()
}
