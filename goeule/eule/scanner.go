package eule

import "fmt"

const eofByte = 0

type scanner struct {
	source []byte
	cursor int
	start  int
	line   int
	nl     bool
}

func newScanner(source []byte) scanner {
	s := scanner{
		source: source,
		cursor: 0,
		line:   1,
	}
	s.skipShebang()
	return s
}

func (s *scanner) scan() token {
	line := s.line

skipWhite:
	for {
		if s.current() == '#' {
			s.skipLineComment()
		} else if s.current() == '/' && s.peek() == '*' {
			s.advance()
			s.advance()
			for s.current() != '*' && s.peek() != '/' {
				if s.current() == '\n' {
					s.line++
				}
				if s.isAtEnd() {
					return s.makeToken("unfinished block comment")
				}
				s.advance()
			}
			s.advance()
			s.advance()
		}

		switch s.current() {
		case '\n':
			s.line++
			fallthrough
		case ' ', '\r', '\t':
			s.advance()
		default:
			break skipWhite
		}
	}

	s.start = s.cursor

	if modeAutoSemicolons {
		if s.nl && line < s.line {
			return s.makeToken(tokenNewLine)
		}
	}

	if s.isAtEnd() {
		return s.makeToken(tokenEof)
	}

	char := s.advance()

	if trio, ok := trio[[3]byte{char, s.current(), s.peek()}]; ok {
		s.advance()
		s.advance()
		return s.makeToken(trio)
	}

	if duo, ok := duo[[2]byte{char, s.current()}]; ok {
		s.advance()
		return s.makeToken(duo)
	}

	if solo, ok := solo[char]; ok {
		return s.makeToken(solo)
	}

	if isAlpha(char) {
		return s.identifier()
	}

	if isNumeric(char, 10) {
		return s.number()
	}

	if char == '"' {
		return s.string()
	}

	return s.errorToken("unexpected symbol '%c'", char)
}

func (s *scanner) skipShebang() {
	if s.current() == '#' && s.peek() == '!' {
		s.skipLineComment()
	}
}

func (s *scanner) skipLineComment() {
	for s.current() != '\n' && !s.isAtEnd() {
		s.advance()
	}
}

func (s *scanner) literal() string {
	return string(s.source[s.start:s.cursor])
}

func (s *scanner) identifierType() tokenType {
	if t, ok := keywords[s.literal()]; ok {
		return t
	}
	return tokenIdentifier
}

func (s *scanner) identifier() token {
	for isAlpha(s.current()) || isNumeric(s.current(), 10) {
		s.advance()
	}
	return s.makeToken(s.identifierType())
}

func (s *scanner) number() token {
	var allowUnderscore bool
	read := func() {
		for isNumeric(s.current(), 10) ||
			(allowUnderscore && s.current() == '_') {
			allowUnderscore = s.current() != '_'
			s.advance()
		}
	}

	allowUnderscore = true
	read()
	if !allowUnderscore || isAlpha(s.current()) {
		return s.errorToken("malformed number")
	}

	if s.current() == '.' && isNumeric(s.peek(), 10) {
		s.advance()
		allowUnderscore = false
		read()
		if !allowUnderscore || isAlpha(s.current()) {
			return s.errorToken("malformed number")
		}
	}

	return s.makeToken(tokenNumber)
}

func (s *scanner) string() token {
	char := s.advance()
	for char != '"' {
		if char == '\n' || s.isAtEnd() {
			return s.errorToken("unfinished string")
		}
		char = s.advance()
	}
	return s.makeToken(tokenString)
}

func (s *scanner) isAtEnd() bool {
	return s.current() == eofByte
}

func (s *scanner) current() byte {
	if s.cursor >= len(s.source) {
		return eofByte
	}
	return s.source[s.cursor]
}

func (s *scanner) peek() byte {
	if s.cursor+1 >= len(s.source) {
		return eofByte
	}
	return s.source[s.cursor+1]
}

func (s *scanner) advance() byte {
	char := s.current()
	s.cursor++
	return char
}

func (s *scanner) makeToken(t tokenType) token {
	s.nl = mapHas(insertNewLineAfter, t)
	literal := string(s.source[s.start:s.cursor])
	tk := token{t, literal, s.line}
	if debugPrintTokens {
		fmt.Println(tk)
	}
	return tk
}

func (s *scanner) errorToken(format string, a ...any) token {
	return token{tokenError, fmt.Sprintf(format, a...), s.line}
}

func isAlpha(char byte) bool {
	lChar := lowerChar(char)
	return 'a' <= lChar && lChar <= 'z' || char == '_'
}

func isNumeric(char byte, base int) bool {
	if base <= 10 {
		return '0' <= char && char <= '0'+byte(base)-1
	}
	lChar := lowerChar(char)
	return ('0' <= char && char <= '9') ||
		('a' <= lChar && lChar <= 'a'+byte(base)-1)
}

func lowerChar(char byte) byte {
	return ('a' - 'A') | char
}

var insertNewLineAfter = map[tokenType]empty{
	tokenRightParen:   {},
	tokenRightBrace:   {},
	tokenRightBracket: {},
	tokenPlusPlus:     {},
	tokenMinusMinus:   {},
	tokenIdentifier:   {},
	tokenNumber:       {},
	tokenString:       {},
	tokenNihil:        {},
	tokenFalse:        {},
	tokenTrue:         {},
	tokenDotDotDot:    {},
	tokenBreak:        {},
	tokenContinue:     {},
	tokenReturn:       {},
}

var solo = map[byte]tokenType{
	'(': tokenLeftParen,
	')': tokenRightParen,
	'{': tokenLeftBrace,
	'}': tokenRightBrace,
	'[': tokenLeftBracket,
	']': tokenRightBracket,
	'<': tokenLeftAngle,
	'>': tokenRightAngle,

	';': tokenSemicolon,
	',': tokenComma,
	'.': tokenDot,
	'!': tokenBang,
	'?': tokenQuestion,
	':': tokenColon,
	'=': tokenEqual,

	'+': tokenPlus,
	'-': tokenMinus,
	'*': tokenStar,
	'/': tokenSlash,
	'%': tokenPercent,
}

var duo = map[[2]byte]tokenType{
	{'=', '='}: tokenEqualEqual,
	{'!', '='}: tokenBangEqual,
	{'<', '='}: tokenLeftAngleEqual,
	{'>', '='}: tokenRightAngleEqual,
	
	{'+', '+'}: tokenPlusPlus,
	{'-', '-'}: tokenMinusMinus,
	
	{'-', '>'}: tokenMinusRightAngle,
	{'=', '>'}: tokenEqualRightAngle,
	
	{'+', '='}: tokenPlusEqual,
	{'-', '='}: tokenMinusEqual,
	{'*', '='}: tokenStarEqual,
	{'/', '='}: tokenSlashEqual,
	{'%', '='}: tokenPercentEqual,
	
	{'|', '|'}: tokenPipePipe,
	{'&', '&'}: tokenAmperAmper,
	{':', ':'}: tokenColonColon,
}

var trio = map[[3]byte]tokenType{
	{'.', '.', '.'}: tokenDotDotDot,

	{'|', '|', '='}: tokenPipePipeEqual,
	{'&', '&', '='}: tokenAmperAmperEqual,
}

var keywords = map[string]tokenType{
	"false": tokenFalse,
	"true":  tokenTrue,

	nihilLiteral:    tokenNihil,
	variableLiteral: tokenVariable,
	functionLiteral: tokenFunction,

	"if":       tokenIf,
	"else":     tokenElse,
	"while":    tokenWhile,
	"do":       tokenDo,
	"for":      tokenFor,
	"break":    tokenBreak,
	"continue": tokenContinue,
	"return":   tokenReturn,

	"switch":  tokenSwitch,
	"case":    tokenCase,
	"default": tokenDefault,
	"and":     tokenAnd,
	"or":      tokenOr,
	"not":     tokenNot,
	"unless":  tokenUnless,
	"until":   tokenUntil,
	"foreach": tokenForEach,
	"in":      tokenIn,
	"then":    tokenThen,
	"try":     tokenTry,

	"typeof": tokenTypeOf,
}
