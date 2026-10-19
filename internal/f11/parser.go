package f11

import "fmt"

type Parser struct {
	ts []Token
	i  int
}

func Parse(ts []Token) (*Program, error) {
	p := &Parser{ts: ts}
	prog := &Program{}
	for !p.at(TokEOF) {
		f, err := p.parseFunc()
		if err != nil { return nil, err }
		prog.Funcs = append(prog.Funcs, f)
	}
	return prog, nil
}
func (p *Parser) cur() Token        { return p.ts[p.i] }
func (p *Parser) at(k TokKind) bool { return p.cur().Kind == k }
func (p *Parser) take() Token {
	t := p.cur()
	if p.i < len(p.ts)-1 { p.i++ }
	return t
}
func (p *Parser) want(k TokKind, what string) (Token, error) {
	if !p.at(k) {
		t := p.cur()
		return Token{}, fmt.Errorf("%d:%d: expected %s, got %q", t.Pos.Line, t.Pos.Col, what, t.Lit)
	}
	return p.take(), nil
}
func (p *Parser) bindOp() bool { return p.at(TokEquals) || p.at(TokDefEq) }
func (p *Parser) wantBind() error {
	if !p.bindOp() {
		t := p.cur()
		return fmt.Errorf("%d:%d: expected =, got %q", t.Pos.Line, t.Pos.Col, t.Lit)
	}
	p.take()
	return nil
}

func (p *Parser) parseFunc() (*FuncDecl, error) {
	st, err := p.want(TokFunc, "func")
	if err != nil { return nil, err }
	n, err := p.want(TokIdent, "function name")
	if err != nil { return nil, err }
	if _, err = p.want(TokLParen, "("); err != nil { return nil, err }
	var ps []Param
	if !p.at(TokRParen) {
		for {
			pn, err := p.want(TokIdent, "parameter name")
			if err != nil { return nil, err }
			if _, err = p.want(TokColon, ":"); err != nil { return nil, err }
			ty, err := p.parseType()
			if err != nil { return nil, err }
			ps = append(ps, Param{Name: pn.Lit, Ty: ty, Pos: pn.Pos})
			if p.at(TokComma) { p.take(); continue }
			break
		}
	}
	if _, err = p.want(TokRParen, ")"); err != nil { return nil, err }
	if _, err = p.want(TokArrow, "->"); err != nil { return nil, err }
	ret, err := p.parseType()
	if err != nil { return nil, err }

	var body Expr
	if p.at(TokLBlock) {
		b, err := p.parseBlock()
		if err != nil { return nil, err }
		body = &BlockExpr{Block: b, Pos: b.Pos}
	} else {
		if _, err = p.want(TokEquals, "{ or ="); err != nil { return nil, err }
		body, err = p.parseExpr(0)
		if err != nil { return nil, err }
		if p.at(TokSemi) { p.take() }
	}
	return &FuncDecl{Name: n.Lit, Params: ps, Ret: ret, Body: body, Pos: st.Pos}, nil
}

func (p *Parser) parseType() (Type, error) {
	var m byte
	if p.at(TokAmp) {
		p.take(); m = '&'
	} else if p.at(TokBang) {
		p.take(); m = '!'
	} else if p.at(TokStar) {
		p.take(); m = '*'
	}
	t, err := p.want(TokIdent, "type")
	if err != nil { return Type{}, err }
	return Type{Name: t.Lit, Mode: m}, nil
}

func (p *Parser) parseWorld() (Expr, error) {
	st := p.take()
	n, err := p.want(TokIdent, "world name")
	if err != nil { return nil, err }
	if _, err = p.want(TokLWorld, "["); err != nil { return nil, err }
	var admits []string
	for !p.at(TokRWorld) {
		a, err := p.want(TokIdent, "admitted name")
		if err != nil { return nil, err }
		admits = append(admits, a.Lit)
		if p.at(TokComma) { p.take() }
	}
	if _, err = p.want(TokRWorld, "]"); err != nil { return nil, err }
	if _, err = p.want(TokArrow, "->"); err != nil { return nil, err }
	ret, err := p.parseType()
	if err != nil { return nil, err }
	body, err := p.parseBlock()
	if err != nil { return nil, err }
	w := &WorldExpr{Name: n.Lit, Admits: admits, Ret: ret, Body: body, Pos: st.Pos}
	if p.at(TokEsc) {
		p.take()
		rv, err := p.want(TokIdent, "recov binding")
		if err != nil { return nil, err }
		w.RecovVar = rv.Lit
		rb, err := p.parseBlock()
		if err != nil { return nil, err }
		w.Recov = rb
	}
	return w, nil
}

func (p *Parser) parseIf() (Expr, error) {
	st := p.take()
	cond, err := p.parseExpr(0)
	if err != nil { return nil, err }
	th, err := p.parseBlock()
	if err != nil { return nil, err }
	if _, err = p.want(TokElse, "else"); err != nil { return nil, err }
	el, err := p.parseBlock()
	if err != nil { return nil, err }
	return &IfExpr{Cond: cond, Then: th, Else: el, Pos: st.Pos}, nil
}

func (p *Parser) parseBlock() (*Block, error) {
	st, err := p.want(TokLBlock, "{")
	if err != nil { return nil, err }
	b := &Block{Pos: st.Pos}
	for !p.at(TokRBlock) {
		if p.at(TokEOF) {
			t := p.cur()
			return nil, fmt.Errorf("%d:%d: unterminated block", t.Pos.Line, t.Pos.Col)
		}
		if p.at(TokK) || p.at(TokJ) || p.at(TokN) {
			r := p.take()
			n, err := p.want(TokIdent, "binding name")
			if err != nil { return nil, err }
			if err = p.wantBind(); err != nil { return nil, err }
			x, err := p.parseExpr(0)
			if err != nil { return nil, err }
			if _, err = p.want(TokSemi, ";"); err != nil { return nil, err }
			role := byte('K')
			switch r.Kind {
			case TokJ: role = 'J'
			case TokN: role = 'N'
			}
			b.Stmts = append(b.Stmts, &BindStmt{Role: role, Name: n.Lit, X: x, Pos: r.Pos})
			continue
		}
		if p.at(TokVar) {
			st := p.take()
			n, err := p.want(TokIdent, "mutable name")
			if err != nil { return nil, err }
			var ty Type
			if p.at(TokColon) {
				p.take()
				ty, err = p.parseType()
				if err != nil { return nil, err }
			}
			if err = p.wantBind(); err != nil { return nil, err }
			x, err := p.parseExpr(0)
			if err != nil { return nil, err }
			if _, err = p.want(TokSemi, ";"); err != nil { return nil, err }
			b.Stmts = append(b.Stmts, &VarStmt{Name: n.Lit, Ty: ty, X: x, Pos: st.Pos})
			continue
		}
		if p.at(TokWhile) {
			st := p.take()
			cond, err := p.parseExpr(0)
			if err != nil { return nil, err }
			body, err := p.parseBlock()
			if err != nil { return nil, err }
			b.Stmts = append(b.Stmts, &WhileStmt{Cond: cond, Body: body, Pos: st.Pos})
			if p.at(TokSemi) { p.take() }
			continue
		}
		if p.at(TokIdent) && p.i+1 < len(p.ts) && p.ts[p.i+1].Kind == TokAssign {
			n := p.take()
			p.take()
			x, err := p.parseExpr(0)
			if err != nil { return nil, err }
			if _, err = p.want(TokSemi, ";"); err != nil { return nil, err }
			b.Stmts = append(b.Stmts, &AssignStmt{Name: n.Lit, X: x, Pos: n.Pos})
			continue
		}
		if p.at(TokIdent) && p.i+1 < len(p.ts) && (p.ts[p.i+1].Kind == TokEquals || p.ts[p.i+1].Kind == TokDefEq) {
			n := p.take()
			p.take()
			x, err := p.parseExpr(0)
			if err != nil { return nil, err }
			if _, err = p.want(TokSemi, ";"); err != nil { return nil, err }
			b.Stmts = append(b.Stmts, &BindStmt{Name: n.Lit, X: x, Pos: n.Pos})
			continue
		}
		x, err := p.parseExpr(0)
		if err != nil { return nil, err }
		if p.at(TokSemi) {
			p.take()
			if p.at(TokRBlock) {
				b.Tail = x
				break
			}
			b.Stmts = append(b.Stmts, &ExprStmt{X: x, Pos: x.GetPos()})
			continue
		}
		b.Tail = x
		break
	}
	if _, err = p.want(TokRBlock, "}"); err != nil { return nil, err }
	return b, nil
}

func prec(k TokKind) int {
	switch k {
	case TokOr: return 1
	case TokAnd: return 2
	case TokEqEq, TokNE: return 3
	case TokLT, TokLE, TokGT, TokGE: return 4
	case TokPlus, TokMinus: return 5
	case TokStar, TokSlash, TokPercent: return 6
	}
	return 0
}

func (p *Parser) parseExpr(min int) (Expr, error) {
	var left Expr
	t := p.cur()
	switch t.Kind {
	case TokInt:
		p.take()
		left = &IntExpr{Value: t.Lit, Pos: t.Pos}
	case TokTrue, TokFalse:
		p.take()
		left = &BoolExpr{Value: t.Kind == TokTrue, Pos: t.Pos}
	case TokDot:
		p.take()
		n, err := p.want(TokIdent, "tag name")
		if err != nil { return nil, err }
		left = &TagExpr{Name: n.Lit, Pos: t.Pos}
	case TokIdent:
		p.take()
		if p.at(TokLParen) {
			p.take()
			var args []Expr
			if !p.at(TokRParen) {
				for {
					a, err := p.parseExpr(0)
					if err != nil { return nil, err }
					args = append(args, a)
					if p.at(TokComma) { p.take(); continue }
					break
				}
			}
			if _, err := p.want(TokRParen, ")"); err != nil { return nil, err }
			left = &CallExpr{Name: t.Lit, Args: args, Pos: t.Pos}
		} else {
			left = &VarExpr{Name: t.Lit, Pos: t.Pos}
		}
	case TokMinus, TokBang:
		p.take()
		x, err := p.parseExpr(7)
		if err != nil { return nil, err }
		left = &UnaryExpr{Op: t.Kind, X: x, Pos: t.Pos}
	case TokLParen:
		p.take()
		x, err := p.parseExpr(0)
		if err != nil { return nil, err }
		if _, err = p.want(TokRParen, ")"); err != nil { return nil, err }
		left = x
	case TokIf:
		return p.parseIf()
	case TokF11:
		return p.parseWorld()
	case TokEsc:
		p.take()
		var target string
		if p.at(TokIdent) && p.i+1 < len(p.ts) &&
			(p.ts[p.i+1].Kind == TokInt || p.ts[p.i+1].Kind == TokDot ||
				p.ts[p.i+1].Kind == TokIdent || p.ts[p.i+1].Kind == TokTrue ||
				p.ts[p.i+1].Kind == TokFalse) {
			target = p.take().Lit
		}
		x, err := p.parseExpr(7)
		if err != nil { return nil, err }
		left = &EscExpr{Target: target, Value: x, Pos: t.Pos}
	default:
		return nil, fmt.Errorf("%d:%d: expected expression, got %q", t.Pos.Line, t.Pos.Col, t.Lit)
	}
	for {
		op := p.cur()
		pr := prec(op.Kind)
		if pr == 0 || pr < min { break }
		p.take()
		right, err := p.parseExpr(pr + 1)
		if err != nil { return nil, err }
		left = &BinaryExpr{Op: op.Kind, L: left, R: right, Pos: op.Pos}
	}
	return left, nil
}
