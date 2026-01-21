package eule

import "fmt"

type tokenType string

const (
	tokenLeftParen    tokenType = "("
	tokenRightParen   tokenType = ")"
	tokenLeftBrace    tokenType = "{"
	tokenRightBrace   tokenType = "}"
	tokenLeftBracket  tokenType = "["
	tokenRightBracket tokenType = "]"
	tokenLeftAngle    tokenType = "<"
	tokenRightAngle   tokenType = ">"

	tokenSemicolon tokenType = ";"
	tokenComma     tokenType = ","
	tokenDot       tokenType = "."
	tokenBang      tokenType = "!"
	tokenQuestion  tokenType = "?"
	tokenColon     tokenType = ":"
	tokenEqual     tokenType = "="

	tokenPlusPlus   tokenType = "++"
	tokenMinusMinus tokenType = "--"

	tokenMinusRightAngle tokenType = "->"
	tokenEqualRightAngle tokenType = "=>"

	tokenPlus    tokenType = "+"
	tokenMinus   tokenType = "-"
	tokenStar    tokenType = "*"
	tokenSlash   tokenType = "/"
	tokenPercent tokenType = "%"

	tokenPlusEqual    tokenType = "+="
	tokenMinusEqual   tokenType = "-="
	tokenStarEqual    tokenType = "*="
	tokenSlashEqual   tokenType = "/="
	tokenPercentEqual tokenType = "%="

	tokenEqualEqual      tokenType = "=="
	tokenBangEqual       tokenType = "!="
	tokenLeftAngleEqual  tokenType = "<="
	tokenRightAngleEqual tokenType = ">="

	tokenPipePipe   tokenType = "||"
	tokenAmperAmper tokenType = "&&"
	tokenColonColon tokenType = "::"

	tokenPipePipeEqual   tokenType = "||="
	tokenAmperAmperEqual tokenType = "&&="

	tokenDotDotDot tokenType = "..."

	tokenIdentifier tokenType = "identifier"
	tokenNumber     tokenType = "number"
	tokenString     tokenType = "string"

	tokenNihil    tokenType = "nihil"
	tokenVariable tokenType = "variable"
	tokenFunction tokenType = "function"

	tokenFalse    tokenType = "false"
	tokenTrue     tokenType = "true"
	tokenIf       tokenType = "if"
	tokenElse     tokenType = "else"
	tokenWhile    tokenType = "while"
	tokenDo       tokenType = "do"
	tokenFor      tokenType = "for"
	tokenBreak    tokenType = "break"
	tokenContinue tokenType = "continue"
	tokenReturn   tokenType = "return"
	tokenSwitch   tokenType = "switch"
	tokenCase     tokenType = "case"
	tokenDefault  tokenType = "default"
	tokenAnd      tokenType = "and"
	tokenOr       tokenType = "or"
	tokenNot      tokenType = "not"
	tokenUnless   tokenType = "unless"
	tokenUntil    tokenType = "until"
	tokenForEach  tokenType = "foreach"
	tokenIn       tokenType = "in"
	tokenThen     tokenType = "then"
	tokenTry      tokenType = "try"

	tokenTypeOf tokenType = "typeof"

	tokenNewLine tokenType = "__new_line"

	tokenError tokenType = "__error"
	tokenEof   tokenType = "__eof"
)

type token struct {
	tokenType
	literal string
	line    int
}

func (t token) String() string {
	return fmt.Sprintf(
		"[ln %d] %-12s %s",
		t.line,
		t.tokenType,
		shortString(t.literal, 32),
	)
}
