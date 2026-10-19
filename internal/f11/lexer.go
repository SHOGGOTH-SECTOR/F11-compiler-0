package f11

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	src  string
	off  int
	line int
	col  int
}

func Lex(src string) ([]Token, error) {
	l := &Lexer{src: src, line: 1, col: 1}
	var out []Token
	for {
		t, err := l.next()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
		if t.Kind == TokEOF {
			return out, nil
		}
	}
}

func (l *Lexer) pos() Pos  { return Pos{Line: l.line, Col: l.col, Off: l.off} }
func (l *Lexer) eof() bool { return l.off >= len(l.src) }
func (l *Lexer) peek() rune {
	if l.eof() {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.off:])
	return r
}
func (l *Lexer) peekNBytes(n int) string {
	if l.off+n > len(l.src) {
		return l.src[l.off:]
	}
	return l.src[l.off : l.off+n]
}
func (l *Lexer) take() rune {
	r, sz := utf8.DecodeRuneInString(l.src[l.off:])
	l.off += sz
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}
func (l *Lexer) skip() {
	for !l.eof() {
		r := l.peek()
		if unicode.IsSpace(r) {
			l.take()
			continue
		}
		if l.peekNBytes(2) == "//" {
			for !l.eof() && l.take() != '\n' {
			}
			continue
		}
		if r == '#' {
			for !l.eof() && l.take() != '\n' {
			}
			continue
		}
		break
	}
}

func (l *Lexer) next() (Token, error) {
	l.skip()
	p := l.pos()
	if l.eof() {
		return Token{Kind: TokEOF, Pos: p}, nil
	}

	pairs := []struct {
		s string
		k TokKind
	}{
		{"->", TokArrow}, {"<-", TokAssign},
		{"<<", TokLWorld}, {">>", TokRWorld}, {"<{", TokLBlock}, {"}>", TokRBlock},
		{":=", TokDefEq}, {"<=", TokLE}, {">=", TokGE}, {"==", TokEqEq}, {"!=", TokNE},
		{"&&", TokAnd}, {"||", TokOr},
	}
	for _, q := range pairs {
		if l.peekNBytes(len(q.s)) == q.s {
			for range q.s {
				l.take()
			}
			return Token{Kind: q.k, Lit: q.s, Pos: p}, nil
		}
	}

	r := l.take()
	switch r {
	case '[':
		return Token{Kind: TokLWorld, Lit: string(r), Pos: p}, nil
	case ']':
		return Token{Kind: TokRWorld, Lit: string(r), Pos: p}, nil
	case '{':
		return Token{Kind: TokLBlock, Lit: string(r), Pos: p}, nil
	case '}':
		return Token{Kind: TokRBlock, Lit: string(r), Pos: p}, nil
	case '⟪':
		return Token{Kind: TokLWorld, Lit: string(r), Pos: p}, nil
	case '⟫':
		return Token{Kind: TokRWorld, Lit: string(r), Pos: p}, nil
	case '⟨':
		return Token{Kind: TokLBlock, Lit: string(r), Pos: p}, nil
	case '⟩':
		return Token{Kind: TokRBlock, Lit: string(r), Pos: p}, nil
	case '≡':
		return Token{Kind: TokDefEq, Lit: string(r), Pos: p}, nil
	case '←':
		return Token{Kind: TokAssign, Lit: string(r), Pos: p}, nil
	case '≤':
		return Token{Kind: TokLE, Lit: string(r), Pos: p}, nil
	case '≥':
		return Token{Kind: TokGE, Lit: string(r), Pos: p}, nil
	case '≠':
		return Token{Kind: TokNE, Lit: string(r), Pos: p}, nil
	case '∧':
		return Token{Kind: TokAnd, Lit: string(r), Pos: p}, nil
	case '∨':
		return Token{Kind: TokOr, Lit: string(r), Pos: p}, nil
	case '(', ')', ',', ':', ';', '=', '+', '-', '*', '/', '%', '<', '>', '!', '&', '.':
		m := map[rune]TokKind{
			'(': TokLParen, ')': TokRParen, ',': TokComma, ':': TokColon, ';': TokSemi,
			'=': TokEquals, '+': TokPlus, '-': TokMinus, '*': TokStar, '/': TokSlash,
			'%': TokPercent, '<': TokLT, '>': TokGT, '!': TokBang, '&': TokAmp, '.': TokDot,
		}
		return Token{Kind: m[r], Lit: string(r), Pos: p}, nil
	}

	if unicode.IsDigit(r) {
		start := p.Off
		for !l.eof() && unicode.IsDigit(l.peek()) {
			l.take()
		}
		return Token{Kind: TokInt, Lit: l.src[start:l.off], Pos: p}, nil
	}
	if unicode.IsLetter(r) || r == '_' {
		start := p.Off
		for !l.eof() {
			x := l.peek()
			if unicode.IsLetter(x) || unicode.IsDigit(x) || x == '_' {
				l.take()
			} else {
				break
			}
		}
		lit := l.src[start:l.off]
		low := strings.ToLower(lit)
		kw := map[string]TokKind{
			"func": TokFunc, "f11": TokF11, "k": TokK, "j": TokJ, "n": TokN,
			"esc": TokEsc, "true": TokTrue, "false": TokFalse,
		}
		if k, ok := kw[low]; ok {
			return Token{Kind: k, Lit: lit, Pos: p}, nil
		}
		return Token{Kind: TokIdent, Lit: lit, Pos: p}, nil
	}
	return Token{}, fmt.Errorf("%d:%d: unexpected character %q", p.Line, p.Col, r)
}
