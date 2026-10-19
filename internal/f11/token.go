package f11

type TokKind int

const (
	TokEOF TokKind = iota
	TokIdent
	TokInt
	TokFunc
	TokF11
	TokK
	TokJ
	TokN
	TokEsc
	TokVar
	TokWhile
	TokIf
	TokElse
	TokStruct
	TokTrue
	TokFalse
	TokLParen
	TokRParen
	TokComma
	TokColon
	TokSemi
	TokArrow
	TokEquals
	TokDefEq
	TokAssign
	TokLWorld
	TokRWorld
	TokLBlock
	TokRBlock
	TokPlus
	TokMinus
	TokStar
	TokSlash
	TokPercent
	TokLT
	TokLE
	TokGT
	TokGE
	TokEqEq
	TokNE
	TokAnd
	TokOr
	TokBang
	TokAmp
	TokDot
)

type Pos struct {
	Line int
	Col  int
	Off  int
}

type Token struct {
	Kind TokKind
	Lit  string
	Pos  Pos
}
