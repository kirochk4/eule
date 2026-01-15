package eule

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

type compiler struct {
	*tokenReader
	fn *Function
	fnType
	locals []localVariable
	*loop
	enclosing *compiler
	scope     int
	prefix    int
}

func newCompiler(source []byte) *compiler {
	return &compiler{
		tokenReader: newTokenReader(source),
		fn:          NewFunction(),
		fnType:      fnTypeScript,
		locals:      []localVariable{},
		loop:        nil,
		enclosing:   nil,
		scope:       0,
		prefix:      0,
	}
}

func (c *compiler) newFunctionCompiler(t fnType) *compiler {
	return &compiler{
		tokenReader: c.tokenReader,
		fn:          NewFunction(),
		fnType:      t,
		locals:      []localVariable{},
		loop:        nil,
		enclosing:   c,
		scope:       1,
	}
}

func (c *compiler) compile() *Function {
	for !c.match(tokenEof) {
		c.declaration()
	}
	if c.hadError {
		return nil
	}
	c.emitReturn()
	return c.fn
}

func (c *compiler) declaration() {
	switch {
	case c.match(tokenVariable):
		c.variableDeclaration()
	default:
		c.statement()
	}

	if c.panic {
		c.synchronize()
	}
}

func (c *compiler) statement() {
	switch {
	case c.match(tokenSemicolon), c.match(tokenNewLine):
		/* pass */
	case c.match(tokenLeftBrace):
		c.beginScope()
		c.block()
		c.endScope()
	case c.match(tokenIf):
		c.ifStatement()
	case c.match(tokenWhile):
		c.whileStatement()
	case c.match(tokenDo):
		c.doStatement()
	case c.match(tokenFor):
		c.forStatement()
	case c.match(tokenBreak):
		c.breakStatement()
	case c.match(tokenContinue):
		c.continueStatement()
	case c.match(tokenReturn):
		c.returnStatement()
	default:
		c.expressionStatement()
	}
}

/* ==  statement ============================================================ */

func (c *compiler) variableDeclaration() {
	var needSemicolon bool
	for {
		nameIndex := c.declareVariable("Expect variable name.")
		if c.match(tokenEqual) {
			c.expressionComma()
			needSemicolon = true
		} else if c.match(tokenLeftParen) {
			isArrow := c.function()
			needSemicolon = isArrow
		} else {
			c.emitBytes(opNihil)
			needSemicolon = true
		}
		c.defineVariable(nameIndex)
		if !c.match(tokenComma) {
			break
		}
	}
	if needSemicolon {
		c.consumeSemicolon("Expect ';' after variable declaration.")
	}
}

func (c *compiler) block() {
	for !c.check(tokenRightBrace) && !c.check(tokenEof) {
		c.declaration()
	}
	c.consume(tokenRightBrace, "Expect '}' after block.")
}

func (c *compiler) ifStatement() {
	c.consume(tokenLeftParen, "Expect '(' after 'if'.")
	c.expression()
	c.consume(tokenRightParen, "Expect ')' after condition.")

	thenJump := c.emitJump(opJumpIfFalse)

	c.emitBytes(opPop)

	c.ignoreNewLine()
	c.statement()

	elseJump := c.emitJump(opJump)

	c.patchJump(thenJump)

	c.emitBytes(opPop)

	if c.match(tokenElse) {
		c.statement()
	}

	c.patchJump(elseJump)
}

func (c *compiler) whileStatement() {
	c.beginLoop()
	loopStart := len(c.fn.Code)

	c.consume(tokenLeftParen, "Expect '(' after 'while'.")
	c.expression()
	c.consume(tokenRightParen, "Expect ')' after condition.")

	exitJump := c.emitJump(opJumpIfFalse)
	c.emitBytes(opPop)

	c.ignoreNewLine()
	c.statement()

	c.emitJumpBack(loopStart)

	c.patchJump(exitJump)
	c.emitBytes(opPop)

	c.endLoop()
}

func (c *compiler) doStatement() {
	c.beginLoop()
	loopStart := len(c.fn.Code)

	c.ignoreNewLine()
	c.statement()

	c.consume(tokenWhile, "Expect 'while' after do statement.")
	c.consume(tokenLeftParen, "Expect '(' after 'while'.")

	c.expression()
	exitJump := c.emitJump(opJumpIfFalse)

	c.emitBytes(opPop)
	c.emitJumpBack(loopStart)

	c.consume(tokenRightParen, "Expect ')' after condition.")
	c.consumeSemicolon("Expect ';' after while.")

	c.patchJump(exitJump)
	c.emitBytes(opPop)

	c.endLoop()
}

func (c *compiler) forStatement() {
	c.beginScope()
	c.consume(tokenLeftParen, "Expect '(' after 'for'.")
	if c.match(tokenSemicolon) {

	} else if c.match(tokenVariable) {
		c.variableDeclaration()
	} else {
		c.expressionStatement()
	}

	c.beginLoop()
	loopStart := len(c.fn.Code)
	exitJump := -1
	if !c.match(tokenSemicolon) {
		c.expression()
		c.consumeSemicolon("Expect ';' after loop condition.")

		exitJump = c.emitJump(opJumpIfFalse)
		c.emitBytes(opPop)
	}

	if !c.match(tokenRightParen) {
		bodyJump := c.emitJump(opJump)
		incrementStart := len(c.fn.Code)
		c.expression()
		c.emitBytes(opPop)
		c.consume(tokenRightParen, "Expect ')' after for clauses.")

		c.emitJumpBack(loopStart)
		c.loop.start = incrementStart
		loopStart = incrementStart
		c.patchJump(bodyJump)
	}

	c.ignoreNewLine()
	c.statement()
	c.emitJumpBack(loopStart)

	if exitJump != -1 {
		c.patchJump(exitJump)
		c.emitBytes(opPop)
	}

	c.endLoop()
	c.endScope()
}

func (c *compiler) breakStatement() {
	if c.loop == nil {
		c.errorAtPrevious("Can't use 'break' outside of a loop.")
		return
	}
	c.consumeSemicolon("Expect ';' after 'break'.")
	c.loop.addBreak(c.emitJump(opJump))
}

func (c *compiler) continueStatement() {
	if c.loop == nil {
		c.errorAtPrevious("Can't use 'continue' outside of a loop.")
		return
	}
	c.consumeSemicolon("Expect ';' after 'continue'.")
	c.emitJumpBack(c.loop.start)
}

func (c *compiler) returnStatement() {
	if c.fnType == fnTypeScript {
		c.errorAtPrevious("Can't return from top-level code.")
	}

	if c.match(tokenSemicolon) {
		c.emitReturn()
	} else {
		c.expression()
		c.consumeSemicolon("Expect ';' after return value.")
		c.emitBytes(opReturn)
	}
}

func (c *compiler) expressionStatement() {
	c.expression()
	c.consumeSemicolon("Expect ';' after expression!.")
	c.emitBytes(opPop)
}

/* ==  expression =========================================================== */

func (c *compiler) expression() {
	c.precedence(precComma)
}

func (c *compiler) expressionComma() {
	c.precedence(precAssign)
}

func (c *compiler) precedence(prec precedence) {
	c.advance()
	nudFn := c.nud()
	if nudFn == nil {
		c.errorAtPrevious("Expect expression.")
		return
	}

	canAssign := prec <= precAssign
	nudFn(canAssign)

	for prec <= precedences[c.current.tokenType] {
		c.advance()
		ledFn := c.led()
		ledFn(canAssign)
	}

	if canAssign && mapHas(assignTokens, c.current.tokenType) {
		c.errorAtPrevious("Invalid assignment target.")
	}
}

var assignTokens = map[tokenType]empty{
	tokenEqual:      {},
	tokenPlusEqual:  {},
	tokenMinusEqual: {},
	tokenStarEqual:  {},
	tokenSlashEqual: {},
}

func (c *compiler) nud() parseFn {
	switch c.previous.tokenType {
	case tokenLeftParen:
		return c.parseGroup
	case tokenIdentifier:
		return c.parseVariable
	case tokenNihil, tokenFalse, tokenTrue:
		return c.parseLiteral
	case tokenNumber:
		return c.parseNumber
	case tokenString:
		return c.parseString
	case tokenLeftBrace:
		return c.parseTable
	case tokenFunction:
		return c.parseFunction
	case tokenPlus, tokenMinus, tokenBang, tokenTypeOf,
		tokenPlusPlus, tokenMinusMinus:
		return c.parsePrefix
	default:
		return nil
	}
}

func (c *compiler) parseGroup(canAssign bool) {
	c.expression()
	c.consume(tokenRightParen, "Expect ')' after expression.")
}

func (c *compiler) parseVariable(canAssign bool) {
	c.namedVariable(c.previous.literal, canAssign)
}

func (c *compiler) namedVariable(name string, canAssign bool) {
	var getOp, setOp uint8

	var index int
	if idx, ok := c.resolveLocal(name); ok {
		index = idx
		getOp = opLoadLocal
		setOp = opStoreLocal
	} else {
		index = int(c.makeConstant(String(name)))
		getOp = opLoadGlobal
		setOp = opStoreGlobal
	}

	c.assign(
		func() { c.emitBytes(setOp, uint8(index)) },
		func() { c.emitBytes(getOp, uint8(index)) },
		func() { c.emitBytes(getOp, uint8(index)) },
		canAssign,
	)
}

func (c *compiler) resolveLocal(name string) (int, bool) {
	for i := len(c.locals) - 1; i >= 0; i-- {
		local := c.locals[i]
		if local.name == name && local.isInitialized {
			return i, true
		}
	}
	return 0, false
}

func (c *compiler) parseLiteral(canAssign bool) {
	switch c.previous.tokenType {
	case tokenNihil:
		c.emitBytes(opNihil)
	case tokenFalse:
		c.emitBytes(opFalse)
	case tokenTrue:
		c.emitBytes(opTrue)
	default:
		return
	}
}

func (c *compiler) parseNumber(canAssign bool) {
	value, err := strconv.ParseFloat(c.previous.literal, 64)
	if err != nil {
		panic(err)
	}
	c.emitNumber(value)
}

func (c *compiler) parseString(canAssign bool) {
	literal := c.previous.literal
	c.emitConstant(String(literal[1 : len(literal)-1]))
}

func (c *compiler) parseTable(canAssign bool) {
	var index float64 = 0
	c.emitBytes(opTable)
	if !c.check(tokenRightBrace) {
		for {
			if c.match(tokenDot) {
				c.consumeIdentifierConstant("identifier expected")
				if c.match(tokenEqual) {
					c.expressionComma()
				} else if c.match(tokenLeftParen) {
					c.function()
				} else {
					c.namedVariable(c.previous.literal, false)
				}
			} else if c.match(tokenLeftBracket) {
				c.expression()
				c.consume(tokenRightBracket, "agagagaga")
				c.consume(tokenEqual, "agagagaga")
				c.expressionComma()
			} else {
				c.emitNumber(index)
				index++
				c.expressionComma()
			}
			c.emitBytes(opDefineKey)
			if !c.match(tokenComma) {
				break
			}
			if c.check(tokenRightBrace) {
				break
			}
		}
	}
	c.consume(tokenRightBrace, "Expect '}' after map pairs.")
}

func (c *compiler) parseFunction(canAssign bool) {
	c.consume(tokenLeftParen, "'(' expected")
	c.function()
}

func (c *compiler) function() bool {
	fc := c.newFunctionCompiler(fnTypeSync)

	fc.parameterList()

	isArrow := false
	fc.consume(tokenLeftBrace, "Expect '{' before function body.")
	fc.block()
	fc.emitReturn()

	c.emitConstant(fc.fn)

	return isArrow
}

func (c *compiler) parsePrefix(canAssign bool) {
	opType := c.previous.tokenType
	switch opType {
	case tokenPlusPlus:
		c.prefix = 1
	case tokenMinusMinus:
		c.prefix = -1
	}
	c.precedence(precUn)
	switch opType {
	case tokenBang:
		c.emitBytes(opNot)
	case tokenPlus:
		c.emitBytes(opPos)
	case tokenMinus:
		c.emitBytes(opNeg)
	case tokenTypeOf:
		c.emitBytes(opTypeOf)
	case tokenPlusPlus, tokenMinusMinus:
		/* pass */
	default:
		panic(unreachable)
	}
}

func (c *compiler) led() parseFn {
	switch c.previous.tokenType {
	case tokenComma:
		return c.parseComma
	case tokenPlus, tokenMinus, tokenStar, tokenSlash,
		tokenEqualEqual, tokenBangEqual,
		tokenLeftAngle, tokenLeftAngleEqual,
		tokenRightAngle, tokenRightAngleEqual:
		return c.parseInfix
	case tokenPipePipe:
		return c.parseOr
	case tokenAmperAmper:
		return c.parseAnd
	case tokenQuestion:
		return c.parseTernary
	case tokenLeftParen:
		return c.parseCall
	case tokenLeftBracket:
		return c.parseKey
	case tokenDot:
		return c.parseDot
	default:
		panic(unreachable)
	}
}

func (c *compiler) parseComma(canAssign bool) {
	c.emitBytes(opPop)
	c.expression()
}

func (c *compiler) parseInfix(canAssign bool) {
	opType := c.previous.tokenType
	c.precedence(precedences[opType] + 1)
	switch opType {
	case tokenBangEqual:
		c.emitBytes(opEq, opNot)
	case tokenEqualEqual:
		c.emitBytes(opEq)
	case tokenLeftAngle:
		c.emitBytes(opLt)
	case tokenLeftAngleEqual:
		c.emitBytes(opLt)
	case tokenRightAngle:
		c.emitBytes(opLe, opNot)
	case tokenRightAngleEqual:
		c.emitBytes(opLt, opNot)
	case tokenPlus:
		c.emitBytes(opAdd)
	case tokenMinus:
		c.emitBytes(opSub)
	case tokenStar:
		c.emitBytes(opMul)
	case tokenSlash:
		c.emitBytes(opDiv)
	default:
		panic(unreachable)
	}
}

func (c *compiler) parseOr(canAssign bool) {
	elseJump := c.emitJump(opJumpIfFalse)
	endJump := c.emitJump(opJump)
	c.patchJump(elseJump)
	c.emitBytes(opPop)
	c.precedence(precOr)
	c.patchJump(endJump)
}

func (c *compiler) parseAnd(canAssign bool) {
	endJump := c.emitJump(opJumpIfFalse)
	c.emitBytes(opPop)
	c.precedence(precAnd)
	c.patchJump(endJump)
}

func (c *compiler) parseTernary(canAssign bool) {
	thenJump := c.emitJump(opJumpIfFalse)
	c.emitBytes(opPop)
	c.expression()
	elseJump := c.emitJump(opJump)
	c.consume(tokenColon, "Expect ':' after expression.")
	c.patchJump(thenJump)
	c.emitBytes(opPop)
	c.expression()
	c.patchJump(elseJump)
}

func (c *compiler) parseCall(canAssign bool) {
	argCount := c.argumentList()
	c.emitBytes(opCall, argCount)
}

func (c *compiler) parseKey(canAssign bool) {
	c.expression()
	c.consume(tokenRightBracket, "Expect ']' after index.")
	c.assign(
		func() { c.emitBytes(opStoreKey) },
		func() { c.emitBytes(opLoadKey) },
		func() { c.emitBytes(opLoadKeyNoPop) },
		canAssign,
	)
}

func (c *compiler) parseDot(canAssign bool) {
	c.consumeIdentifierConstant("agagagagagga")
	c.assign(
		func() { c.emitBytes(opStoreKey) },
		func() { c.emitBytes(opLoadKey) },
		func() { c.emitBytes(opLoadKeyNoPop) },
		canAssign,
	)
}

/* == utilities ============================================================= */

func (c *compiler) assign(set, get, getNoPop func(), canAssign bool) {
	if c.prefix != 0 && precedences[c.current.tokenType] <= precUn {
		if c.prefix > 0 {
			getNoPop()
			c.emitNumber(1)
			c.emitBytes(opAdd)
			set()
		} else {
			getNoPop()
			c.emitNumber(1)
			c.emitBytes(opSub)
			set()
		}
		c.prefix = 0
	} else if c.match(tokenPlusPlus) {
		getNoPop()
		c.emitBytes(opStoreTemp)
		c.emitNumber(1)
		c.emitBytes(opAdd)
		set()
		c.emitBytes(opLoadTemp)
	} else if c.match(tokenMinusMinus) {
		getNoPop()
		c.emitBytes(opStoreTemp)
		c.emitNumber(1)
		c.emitBytes(opSub)
		set()
		c.emitBytes(opLoadTemp)
	} else if canAssign {
		switch {
		case c.match(tokenEqual):
			c.expression()
			set()
		case c.match(tokenPlusEqual):
			getNoPop()
			c.expression()
			c.emitBytes(opAdd)
			set()
		case c.match(tokenMinusEqual):
			getNoPop()
			c.expression()
			c.emitBytes(opSub)
			set()
		case c.match(tokenStarEqual):
			getNoPop()
			c.expression()
			c.emitBytes(opMul)
			set()
		case c.match(tokenSlashEqual):
			getNoPop()
			c.expression()
			c.emitBytes(opDiv)
			set()
		default:
			get()
		}
	} else {
		get()
	}
}

func (c *compiler) declareVariable(message string) uint8 {
	c.consume(tokenIdentifier, message)
	c.declareName()
	if c.scope == 0 {
		return c.makeConstant(String(c.previous.literal))
	} else {
		return 0
	}
}

func (c *compiler) defineVariable(nameIndex uint8) {
	if c.scope == 0 {
		c.emitBytes(opDefineGlobal, nameIndex)
	} else {
		c.markInitialized()
	}
}

func (c *compiler) declareName() {
	if c.scope == 0 {
		return
	}

	name := c.previous.literal
	for i := len(c.locals) - 1; i >= 0; i-- {
		local := &c.locals[i]
		if local.depth < c.scope {
			break
		}
		if local.name == name {
			c.errorAtPrevious("Already a variable with this name in this scope.")
		}
	}

	c.addLocal(name)
}

func (c *compiler) addLocal(name string) {
	if c.scope == 0 {
		return
	}

	if len(c.locals) == uint8Count {
		c.errorAtPrevious("Too many local variables in function.")
	}
	c.locals = append(c.locals, localVariable{name, c.scope, false})
}

func (c *compiler) markInitialized() {
	if c.scope == 0 {
		return
	}

	c.locals[len(c.locals)-1].isInitialized = true
}

func (c *compiler) makeConstant(value Value) uint8 {
	index := c.fn.addConstant(value)
	if index > int(uint8Max) {
		c.errorAtPrevious("Too many constants in one chunk.")
		return 0
	}
	return uint8(index)
}

func (c *compiler) emitBytes(b ...uint8) {
	for _, b := range b {
		c.fn.writeCode(b, c.previous.line)
	}
}

func (c *compiler) emitNumber(num float64) {
	if 0 <= num && num <= float64(uint8Max) &&
		math.Floor(num) == num {
		c.emitBytes(opSmallInteger, uint8(num))
	} else {
		c.emitConstant(Number(num))
	}
}

func (c *compiler) emitConstant(value Value) {
	c.emitBytes(opConstant, c.makeConstant(value))
}

func (c *compiler) consumeIdentifierConstant(message string) {
	c.consume(tokenIdentifier, message)
	c.emitConstant(String(c.previous.literal))
}

func (c *compiler) emitReturn() {
	c.emitBytes(opNihil, opReturn)
}

func (c *compiler) emitJump(instruction uint8) int {
	c.emitBytes(instruction)
	c.emitBytes(0xff)
	c.emitBytes(0xff)
	return len(c.fn.Code) - 2
}

func (c *compiler) patchJump(offset int) {
	jump := len(c.fn.Code) - offset - 2

	if jump > int(uint16Max) {
		c.errorAtPrevious("Too much code to jump over.")
	}

	c.fn.Code[offset] = uint8((jump >> 8) & 0xff)
	c.fn.Code[offset+1] = uint8(jump & 0xff)
}

func (c *compiler) emitJumpBack(loopStart int) {
	c.emitBytes(opJumpBack)

	offset := len(c.fn.Code) - loopStart + 2
	if offset > int(uint16Max) {
		c.errorAtPrevious("Loop body too large.")
	}

	c.emitBytes(uint8((offset >> 8) & 0xff))
	c.emitBytes(uint8(offset & 0xff))
}

func (c *compiler) beginScope() { c.scope++ }

func (c *compiler) endScope() {
	c.scope--

	for len(c.locals) > 0 && c.locals[len(c.locals)-1].depth > c.scope {
		c.emitBytes(opPop)
		c.locals = c.locals[:len(c.locals)-1]
	}
}

func (c *compiler) beginLoop() {
	c.loop = &loop{len(c.fn.Code), nil, c.loop}
}

func (c *compiler) endLoop() {
	for _, breakJump := range c.loop.breaks {
		c.patchJump(breakJump)
	}
	c.loop = c.loop.enclosing
}

func (c *compiler) argumentList() uint8 {
	var argCount uint8 = 0
	if !c.check(tokenRightParen) {
		for {
			c.expressionComma()
			if argCount == 255 {
				c.errorAtPrevious("Can't have more than 255 arguments.")
			}
			argCount++
			if !c.match(tokenComma) {
				break
			}
			if c.check(tokenRightParen) {
				break
			}
		}
	}
	c.consume(tokenRightParen, "Expect ')' after arguments.")
	return argCount
}

func (c *compiler) parameterList() {
	if !c.check(tokenRightParen) {
		for {
			c.fn.Arity++
			if c.fn.Arity > 255 {
				c.errorAtCurrent("Can't have more than 255 parameters.")
			}
			parameterIndex := c.declareVariable("Expect parameter name.")
			c.defineVariable(parameterIndex)
			if !c.match(tokenComma) {
				break
			}
			if c.check(tokenRightParen) {
				break
			}
		}
	}
	c.consume(tokenRightParen, "Expect ')' after parameters.")
}

// == precedence ============================================================ */

type precedence int

const (
	precLowest precedence = iota
	precComma             // ,
	precAssign            // =
	precTern              // ? : then else
	precOr                // || or
	precAnd               // && and
	precBor               // |
	precBxor              // ^
	precBand              // &
	precEq                // == !=
	precComp              // < > <= >=
	precShift             // << >> >>>
	precTerm              // + -
	precFact              // * / %
	precUn                // ! not + - ~ ++ -- typeof
	precCall              // . () [] -> ++ --
	precHighest
)

var precedences = map[tokenType]precedence{
	tokenComma: precComma,

	tokenEqual: precAssign,

	tokenQuestion: precTern,

	tokenPipePipe: precOr,

	tokenAmperAmper: precAnd,

	tokenEqualEqual: precEq,
	tokenBangEqual:  precEq,

	tokenLeftAngle:       precComp,
	tokenRightAngle:      precComp,
	tokenLeftAngleEqual:  precComp,
	tokenRightAngleEqual: precComp,

	tokenPlus:  precTerm,
	tokenMinus: precTerm,

	tokenStar:  precFact,
	tokenSlash: precFact,

	tokenLeftParen:   precCall,
	tokenLeftBracket: precCall,
	tokenDot:         precCall,
}

/* == token reader ========================================================== */

type tokenReader struct {
	scanner
	current  token
	previous token
	hadError bool
	panic    bool
}

func newTokenReader(source []byte) *tokenReader {
	p := &tokenReader{scanner: newScanner(source)}
	p.advance()
	return p
}

func (r *tokenReader) errorAt(token *token, message string) {
	if r.panic {
		return
	}
	r.panic = true
	fmt.Fprintf(os.Stderr, "[ln %d] %s", token.line, message)

	switch token.tokenType {
	case tokenEof:
		fmt.Fprintf(os.Stderr, " at end")
	case tokenError:
	default:
		fmt.Fprintf(os.Stderr, " at '%s'", token.literal)
	}
	fmt.Fprintln(os.Stderr)

	r.hadError = true
}

func (r *tokenReader) errorAtPrevious(message string) {
	r.errorAt(&r.previous, message)
}

func (r *tokenReader) errorAtCurrent(message string) {
	r.errorAt(&r.current, message)
}

func (r *tokenReader) advance() {
	r.previous = r.current
	for {
		r.current = r.scanner.scan()
		if r.current.tokenType != tokenError {
			break
		}
		r.errorAtCurrent(r.current.literal)
	}
}

func (r *tokenReader) consume(t tokenType, message string) {
	if r.current.tokenType == t {
		r.advance()
		return
	}
	r.errorAtCurrent(message)
}

func (r *tokenReader) consumeSemicolon(message string) {
	r.consume(tokenSemicolon, message)
}

func (r *tokenReader) ignoreNewLine() { r.match(tokenNewLine) }

func (r *tokenReader) check(t tokenType) bool {
	return r.current.tokenType == t
}

func (r *tokenReader) match(t tokenType) bool {
	if !r.check(t) {
		return false
	}
	r.advance()
	return true
}

func (r *tokenReader) synchronize() {
	r.panic = false

	for r.current.tokenType != tokenEof {
		if r.previous.tokenType == tokenSemicolon {
			return
		}

		if _, ok := safeTokens[r.current.tokenType]; ok {
			return
		}

		r.advance()
	}
}

var safeTokens = map[tokenType]empty{
	tokenVariable: {},
	tokenIf:       {},
	tokenWhile:    {},
	tokenDo:       {},
	tokenFor:      {},
	tokenBreak:    {},
	tokenContinue: {},
	tokenReturn:   {},
}

/* == additional ============================================================ */

type parseFn func(canAssign bool)

type localVariable struct {
	name          string
	depth         int
	isInitialized bool
}

type loop struct {
	start     int
	breaks    []int
	enclosing *loop
}

func (l *loop) addBreak(position int) {
	l.breaks = append(l.breaks, position)
}

type fnType int

const (
	fnTypeScript fnType = iota
	fnTypeSync
)
