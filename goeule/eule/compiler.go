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
	locals []localVar
	*loop
	enclosing *compiler
	scope     int
	prefix    []bool
}

func newCompiler(source []byte) *compiler {
	return &compiler{
		tokenReader: newTokenReader(source),
		fn:          NewFunction("@"),
		fnType:      fnTypeScript,
		loop:        nil,
		enclosing:   nil,
		scope:       0,
	}
}

func (c *compiler) newFunctionCompiler(t fnType, name string) *compiler {
	return &compiler{
		tokenReader: c.tokenReader,
		fn:          NewFunction(name),
		fnType:      t,
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
		c.ifStatement(false)
	case c.match(tokenUnless):
		c.ifStatement(true)
	case c.match(tokenWhile):
		c.whileStatement("", false)
	case c.match(tokenUntil):
		c.whileStatement("", true)
	case c.match(tokenDo):
		c.doStatement("")
	case c.match(tokenFor):
		c.forStatement("")
	case c.match(tokenForEach):
		c.forEachStatement("")
	case c.match(tokenBreak):
		c.breakStatement()
	case c.match(tokenContinue):
		c.continueStatement()
	case c.match(tokenReturn):
		c.returnStatement()
	case c.check(tokenIdentifier) && c.checkNext(tokenColon):
		c.labelStatement()
	default:
		c.expressionStatement()
	}
}

/* ==  statement ============================================================ */

func (c *compiler) variableDeclaration() {
	var needSemicolon bool
	for {
		nameIndex := c.declareVariable()
		name := c.previous.literal
		if c.match(tokenEqual) {
			c.expression()
			needSemicolon = true
		} else if c.check(tokenLeftParen) ||
			c.check(tokenEqualRightAngle) ||
			c.check(tokenLeftBrace) {
			isArrow := c.function(name)
			needSemicolon = isArrow
		} else {
			c.emit(opNihil)
			needSemicolon = true
		}
		c.defineVariable(nameIndex)
		if !c.match(tokenComma) {
			break
		}
	}
	if needSemicolon {
		c.consumeSemicolon()
	}
}

func (c *compiler) block() {
	for !c.check(tokenRightBrace) && !c.check(tokenEof) {
		c.declaration()
	}
	c.consume(tokenRightBrace)
}

func (c *compiler) ifStatement(reverse bool) {
	c.consume(tokenLeftParen)
	c.expressionAllowComma()
	c.consume(tokenRightParen)

	if reverse {
		c.emit(opNot)
	}
	thenJump := c.emitJump(opJumpIfFalse)

	c.emit(opPop)

	c.ignoreNewLine()
	c.statement()

	elseJump := c.emitJump(opJump)

	c.patchJump(thenJump)

	c.emit(opPop)

	if c.match(tokenElse) {
		c.statement()
	}

	c.patchJump(elseJump)
}

func (c *compiler) whileStatement(label string, reverse bool) {
	loopStart := c.beginLoop(label, loopLoop)

	c.consume(tokenLeftParen)
	c.expressionAllowComma()
	c.consume(tokenRightParen)

	if reverse {
		c.emit(opNot)
	}
	exitJump := c.emitJump(opJumpIfFalse)
	c.emit(opPop)

	c.ignoreNewLine()
	c.statement()

	c.emitJumpBack(loopStart)

	c.patchJump(exitJump)
	c.emit(opPop)

	c.endLoop()
}

func (c *compiler) doStatement(label string) {
	loopStart := c.beginLoop(label, loopLoop)

	c.ignoreNewLine()
	c.statement()

	reverse := false
	if c.match(tokenUntil) {
		reverse = true
	} else {
		c.consume(tokenWhile)
	}
	c.consume(tokenLeftParen)

	c.expressionAllowComma()

	if reverse {
		c.emit(opNot)
	}
	exitJump := c.emitJump(opJumpIfFalse)

	c.emit(opPop)
	c.emitJumpBack(loopStart)

	c.consume(tokenRightParen)
	c.consumeSemicolon()

	c.patchJump(exitJump)
	c.emit(opPop)

	c.endLoop()
}

func (c *compiler) forStatement(label string) {
	c.beginScope()
	c.consume(tokenLeftParen)
	if c.match(tokenSemicolon) {

	} else if c.match(tokenVariable) {
		c.variableDeclaration()
	} else {
		c.expressionStatement()
	}

	loopStart := c.beginLoop(label, loopLoop)
	exitJump := -1
	if !c.match(tokenSemicolon) {
		c.expressionAllowComma()
		c.consumeSemicolon()

		exitJump = c.emitJump(opJumpIfFalse)
		c.emit(opPop)
	}

	if !c.match(tokenRightParen) {
		bodyJump := c.emitJump(opJump)
		incrementStart := len(c.fn.Code)
		c.expressionAllowComma()
		c.emit(opPop)
		c.consume(tokenRightParen)

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
		c.emit(opPop)
	}

	c.endLoop()
	c.endScope()
}

func (c *compiler) forEachStatement(label string) {
	c.beginScope()

	c.consume(tokenLeftParen)
	c.addLocal("@")
	c.defineVariable(c.declareVariable())

	c.consume(tokenIn)
	c.expression()

	loopStart := c.beginLoop(label, loopLoop)

	c.emit(opDup, opCall, 0)
	exitJump := c.emitJump(opJumpIfNihil)
	c.consume(tokenRightParen)

	c.ignoreNewLine()

	c.statement()
	c.emit(opPop)
	c.emitJumpBack(loopStart)

	c.patchJump(exitJump)

	c.endLoop()
	c.endScope()
}

func (c *compiler) breakStatement() {
	if !c.matchSemicolon() {
		c.consume(tokenIdentifier)
		label := c.previous.literal
		loop := c.loop
		for loop != nil {
			if loop.label == label {
				loop.addBreak(c.emitJump(opJump))
				goto end
			}
			loop = loop.enclosing
		}
		c.errorAtPrevious("undefined label")
		return
	} else {
		loop := c.loop
		for loop != nil {
			if loop.loopType == loopLoop || loop.loopType == loopSwitch {
				loop.addBreak(c.emitJump(opJump))
				goto end
			}
			loop = loop.enclosing
		}
		c.errorAtPrevious("break outside loop")
		return
	}
end:
	c.consumeEnd()
}

func (c *compiler) continueStatement() {
	if !c.matchSemicolon() {
		c.consume(tokenIdentifier)
		label := c.previous.literal
		loop := c.loop
		for loop != nil {
			if loop.label == label {
				if loop.loopType != loopLoop {
					c.errorAtPrevious("continue non loop label")
					return
				}
				c.emitJumpBack(loop.start)
				goto end
			}
			loop = loop.enclosing
		}
		c.errorAtPrevious("undefined label")
		return
	} else {
		loop := c.loop
		for loop != nil {
			if loop.loopType == loopLoop {
				c.emitJumpBack(loop.start)
				goto end
			}
			loop = loop.enclosing
		}
		c.errorAtPrevious("continue outside loop")
		return
	}
end:
	c.consumeEnd()
}

func (c *compiler) returnStatement() {
	if c.fnType == fnTypeScript {
		c.errorAtPrevious("return outside function")
	}

	if c.matchSemicolon() {
		c.emitReturn()
	} else {
		c.expression()
		c.consumeEnd()
		c.emit(opReturn)
	}
}

func (c *compiler) labelStatement() {
	label := c.current.literal
	c.advance()
	c.advance()
	switch {
	case c.match(tokenWhile):
		c.whileStatement(label, false)
	case c.match(tokenDo):
		c.doStatement(label)
	case c.match(tokenFor):
		c.forStatement(label)
	case c.match(tokenForEach):
		c.forEachStatement(label)
	case c.match(tokenLeftBrace):
		c.beginLoop(label, loopBlock)
		c.beginScope()
		c.block()
		c.endScope()
		c.endLoop()
	default:
		c.statement()
	}
}

func (c *compiler) expressionStatement() {
	c.expressionAllowComma()
	c.consumeEnd()
	c.emit(opPop)
}

/* ==  expression =========================================================== */

func (c *compiler) expressionAllowComma() {
	c.precedence(precComma)
}

func (c *compiler) expression() {
	c.precedence(precAssign)
}

func (c *compiler) precedence(prec precedence) {
	c.advance()
	nudFn := c.nud()
	if nudFn == nil {
		c.errorAtPrevious("expression expected")
		return
	}

	canAssign := prec <= precAssign
	nudFn(canAssign)

	for prec <= precedences[c.current.tokenType] {
		c.advance()
		ledFn := c.led()
		ledFn(canAssign)
	}

	if mapHas(incTokens, c.current.tokenType) {
		c.errorAtPrevious("invalid postincrement")
	} else if canAssign && mapHas(assignTokens, c.current.tokenType) {
		c.errorAtPrevious("invalid assignment")
	}
}

var assignTokens = map[tokenType]empty{
	tokenEqual:      {},
	tokenPlusEqual:  {},
	tokenMinusEqual: {},
	tokenStarEqual:  {},
	tokenSlashEqual: {},
}

var incTokens = map[tokenType]empty{
	tokenPlusPlus:   {},
	tokenMinusMinus: {},
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
		tokenPlusPlus, tokenMinusMinus, tokenNot:
		return c.parsePrefix
	default:
		return nil
	}
}

func (c *compiler) parseGroup(canAssign bool) {
	c.expressionAllowComma()
	c.consume(tokenRightParen)
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
	} else if idx, ok := c.resolveUpval(name); ok {
		index = idx
		getOp = opLoadUpvalue
		setOp = opStoreUpvalue
	} else {
		index = int(c.makeConstant(String(name)))
		getOp = opLoadGlobal
		setOp = opStoreGlobal
	}

	c.assign(
		func() { c.emit(setOp, uint8(index)) },
		func() { c.emit(getOp, uint8(index)) },
		func() { c.emit(getOp, uint8(index)) },
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

func (c *compiler) resolveUpval(name string) (int, bool) {
	if c.enclosing == nil {
		return 0, false
	} else if local, ok := c.enclosing.resolveLocal(name); ok {
		c.enclosing.locals[local].isCaptured = true
		return c.addUpval(local, true), true
	} else if upval, ok := c.enclosing.resolveUpval(name); ok {
		return c.addUpval(upval, false), true
	}
	return 0, false
}

func (c *compiler) parseLiteral(canAssign bool) {
	switch c.previous.tokenType {
	case tokenNihil:
		c.emit(opNihil)
	case tokenFalse:
		c.emit(opFalse)
	case tokenTrue:
		c.emit(opTrue)
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
	c.emit(opTable)
	if !c.check(tokenRightBrace) {
		for {
			if c.match(tokenDot) {
				c.consumeIdentifierConstant()
				name := c.previous.literal
				if c.match(tokenEqual) {
					c.expression()
				} else if c.check(tokenLeftParen) ||
					c.check(tokenEqualRightAngle) ||
					c.check(tokenLeftBrace) {
					c.function(name)
				} else {
					c.namedVariable(c.previous.literal, false)
				}
				c.emit(opDefineKey)
			} else if c.match(tokenLeftBracket) {
				c.expressionAllowComma()
				c.consume(tokenRightBracket)
				c.consume(tokenEqual)
				c.expression()
				c.emit(opDefineKey)
			} else {
				c.expression()
				if c.match(tokenDotDotDot) {
					c.emit(opDefineKeySpread)
				} else {
					c.emitNumber(index)
					index++
					c.emit(opSwap, opDefineKey)
				}
			}
			if !c.match(tokenComma) {
				break
			}
			if c.check(tokenRightBrace) {
				break
			}
		}
	}
	c.consume(tokenRightBrace)
}

func (c *compiler) parseFunction(canAssign bool) {
	c.function("")
}

func (c *compiler) function(name string) bool {
	fc := c.newFunctionCompiler(fnTypeDefault, name)

	if fc.match(tokenLeftParen) {
		fc.parameterList()
	}

	isArrow := false
	if fc.match(tokenEqualRightAngle) {
		isArrow = true
		fc.expression()
		fc.emit(opReturn)
	} else {
		fc.consume(tokenLeftBrace)
		fc.block()
		fc.emitReturn()
	}

	c.emitConstant(fc.fn)
	if len(fc.fn.Upvals) != 0 {
		c.emit(opClosure)
	}
	return isArrow
}

func (c *compiler) parsePrefix(canAssign bool) {
	opType := c.previous.tokenType
	prefixLen := len(c.prefix)
	switch opType {
	case tokenPlusPlus:
		slicePush(&c.prefix, true)
	case tokenMinusMinus:
		slicePush(&c.prefix, false)
	}
	c.precedence(precUn)
	switch opType {
	case tokenBang, tokenNot:
		c.emit(opNot)
	case tokenPlus:
		c.emit(opPos)
	case tokenMinus:
		c.emit(opNeg)
	case tokenTypeOf:
		c.emit(opTypeOf)
	case tokenPlusPlus, tokenMinusMinus:
		if prefixLen < len(c.prefix) {
			c.prefix = nil
			c.errorAtPrevious("invalid preincrement")
		}
	default:
		panic(unreachable)
	}
}

func (c *compiler) led() parseFn {
	switch c.previous.tokenType {
	case tokenComma:
		return c.parseComma
	case tokenPlus, tokenMinus,
		tokenStar, tokenSlash, tokenPercent,
		tokenEqualEqual, tokenBangEqual,
		tokenLeftAngle, tokenLeftAngleEqual,
		tokenRightAngle, tokenRightAngleEqual:
		return c.parseInfix
	case tokenPipePipe, tokenOr:
		return c.parseOr
	case tokenAmperAmper, tokenAnd:
		return c.parseAnd
	case tokenQuestion, tokenThen:
		return c.parseTernary
	case tokenLeftParen:
		return c.parseCall
	case tokenLeftBracket:
		return c.parseKey
	case tokenDot:
		return c.parseDot
	case tokenMinusRightAngle:
		return c.parseArrow
	default:
		panic(unreachable)
	}
}

func (c *compiler) parseComma(canAssign bool) {
	c.emit(opPop)
	c.expressionAllowComma()
}

func (c *compiler) parseInfix(canAssign bool) {
	opType := c.previous.tokenType
	c.precedence(precedences[opType] + 1)
	switch opType {
	case tokenBangEqual:
		c.emit(opEq, opNot)
	case tokenEqualEqual:
		c.emit(opEq)
	case tokenLeftAngle:
		c.emit(opLt)
	case tokenLeftAngleEqual:
		c.emit(opLt)
	case tokenRightAngle:
		c.emit(opLe, opNot)
	case tokenRightAngleEqual:
		c.emit(opLt, opNot)
	case tokenPlus:
		c.emit(opAdd)
	case tokenMinus:
		c.emit(opSub)
	case tokenStar:
		c.emit(opMul)
	case tokenSlash:
		c.emit(opDiv)
	case tokenPercent:
		c.emit(opMod)
	default:
		panic(unreachable)
	}
}

func (c *compiler) parseOr(canAssign bool) {
	elseJump := c.emitJump(opJumpIfFalse)
	endJump := c.emitJump(opJump)
	c.patchJump(elseJump)
	c.emit(opPop)
	c.precedence(precOr)
	c.patchJump(endJump)
}

func (c *compiler) parseAnd(canAssign bool) {
	endJump := c.emitJump(opJumpIfFalse)
	c.emit(opPop)
	c.precedence(precAnd)
	c.patchJump(endJump)
}

func (c *compiler) parseTernary(canAssign bool) {
	thenJump := c.emitJump(opJumpIfFalse)
	c.emit(opPop)
	c.expressionAllowComma()
	elseJump := c.emitJump(opJump)
	if !c.match(tokenElse) {
		c.consume(tokenColon)
	}
	c.patchJump(thenJump)
	c.emit(opPop)
	c.expression()
	c.patchJump(elseJump)
}

func (c *compiler) parseCall(canAssign bool) {
	argCount, spread := c.argumentList()
	if spread {
		c.emit(opCallSpread, argCount)
	} else {
		c.emit(opCall, argCount)
	}
}

func (c *compiler) parseKey(canAssign bool) {
	c.expressionAllowComma()
	c.consume(tokenRightBracket)
	c.assign(
		func() { c.emit(opStoreKey) },
		func() { c.emit(opLoadKey) },
		func() { c.emit(opDupTwo, opLoadKey) },
		canAssign,
	)
}

func (c *compiler) parseDot(canAssign bool) {
	c.consumeIdentifierConstant()
	c.assign(
		func() { c.emit(opStoreKey) },
		func() { c.emit(opLoadKey) },
		func() { c.emit(opDupTwo, opLoadKey) },
		canAssign,
	)
}

func (c *compiler) parseArrow(canAssign bool) {
	c.emit(opDup)
	c.consumeIdentifierConstant()
	c.emit(opLoadKey, opSwap)
	if c.match(tokenLeftParen) {
		argCount, spread := c.argumentList()
		if spread {
			c.emit(opCallSpread, argCount+1)
		} else {
			c.emit(opCall, argCount+1)
		}
	} else {
		c.assign(
			func() { c.emit(opCall, 2) },
			func() { c.emit(opCall, 1) },
			func() { c.emit(opDupTwo, opCall, 1) },
			canAssign,
		)
	}
}

/* == utilities ============================================================= */

func (c *compiler) assign(set, get, getNoPop func(), canAssign bool) {
	if len(c.prefix) != 0 && precedences[c.current.tokenType] <= precUn {
		if slicePop(&c.prefix) {
			getNoPop()
			c.emitNumber(1)
			c.emit(opAdd)
			set()
		} else {
			getNoPop()
			c.emitNumber(1)
			c.emit(opSub)
			set()
		}
	} else if c.match(tokenPlusPlus) {
		getNoPop()
		c.emit(opStoreTemp)
		c.emitNumber(1)
		c.emit(opAdd)
		set()
		c.emit(opLoadTemp)
	} else if c.match(tokenMinusMinus) {
		getNoPop()
		c.emit(opStoreTemp)
		c.emitNumber(1)
		c.emit(opSub)
		set()
		c.emit(opLoadTemp)
	} else if canAssign {
		switch {
		case c.match(tokenEqual):
			c.expression()
			set()
		case c.match(tokenPlusEqual):
			getNoPop()
			c.expression()
			c.emit(opAdd)
			set()
		case c.match(tokenMinusEqual):
			getNoPop()
			c.expression()
			c.emit(opSub)
			set()
		case c.match(tokenStarEqual):
			getNoPop()
			c.expression()
			c.emit(opMul)
			set()
		case c.match(tokenSlashEqual):
			getNoPop()
			c.expression()
			c.emit(opDiv)
			set()
		case c.match(tokenPercentEqual):
			getNoPop()
			c.expression()
			c.emit(opMod)
			set()
		case c.match(tokenPipePipeEqual):
			getNoPop()
			c.parseOr(false)
			set()
		case c.match(tokenAmperAmperEqual):
			getNoPop()
			c.parseAnd(false)
			set()
		default:
			get()
		}
	} else {
		get()
	}
}

func (c *compiler) declareVariable() uint8 {
	c.consume(tokenIdentifier)
	if c.scope == 0 {
		return c.makeConstant(String(c.previous.literal))
	} else {
		c.declareLocalVariable()
		return 0
	}
}

func (c *compiler) defineVariable(nameIndex uint8) {
	if c.scope == 0 {
		c.emit(opDefineGlobal, nameIndex)
	} else {
		c.markLastInitialized()
	}
}

func (c *compiler) declareLocalVariable() {
	name := c.previous.literal
	for i := len(c.locals) - 1; i >= 0; i-- {
		local := &c.locals[i]
		if local.depth < c.scope {
			break
		}
		if local.name == name {
			c.errorAtPrevious("variable already declared")
		}
	}

	c.addLocal(name)
}

func (c *compiler) addLocal(name string) {
	if len(c.locals) == uint8Count {
		c.errorAtPrevious(
			fmt.Sprintf("too many variables (%d)", uint8Max),
		)
	}
	c.locals = append(c.locals, localVar{name, c.scope, false, false})
}

func (c *compiler) addUpval(index int, isLocal bool) int {
	for i := len(c.fn.Upvals) - 1; i >= 0; i-- {
		if c.fn.Upvals[i].Index == uint8(index) &&
			c.fn.Upvals[i].IsLocal == isLocal {
			return i
		}
	}

	if len(c.fn.Upvals) == uint8Count {
		c.errorAtPrevious(
			fmt.Sprintf("too many upvalues (%d)", uint8Max),
		)
	}

	c.fn.Upvals = append(c.fn.Upvals, compUpval{isLocal, uint8(index)})
	return len(c.fn.Upvals) - 1
}

func (c *compiler) markLastInitialized() {
	c.locals[len(c.locals)-1].isInitialized = true
}

func (c *compiler) makeConstant(value Value) uint8 {
	index := c.fn.addConstant(value)
	if index > int(uint8Max) {
		c.errorAtPrevious(
			fmt.Sprintf("too many constants (%d)", uint8Max),
		)
		return 0
	}
	return uint8(index)
}

func (c *compiler) emit(b ...uint8) {
	for _, b := range b {
		c.fn.writeCode(b, c.previous.line)
	}
}

func (c *compiler) emitNumber(num float64) {
	if useSmallInteger {
		if 0 <= num && num <= float64(uint8Max) &&
			math.Floor(num) == num {
			c.emit(opSmallInteger, uint8(num))
		} else {
			c.emitConstant(Number(num))
		}
	} else {
		c.emitConstant(Number(num))
	}
}

func (c *compiler) emitConstant(value Value) {
	c.emit(opConstant, c.makeConstant(value))
}

func (c *compiler) consumeIdentifierConstant() {
	c.consume(tokenIdentifier)
	c.emitConstant(String(c.previous.literal))
}

func (c *compiler) emitReturn() {
	c.emit(opNihil, opReturn)
}

func (c *compiler) emitJump(instruction uint8) int {
	c.emit(instruction)
	c.emit(0xff)
	c.emit(0xff)
	return len(c.fn.Code) - 2
}

func (c *compiler) patchJump(offset int) {
	jump := len(c.fn.Code) - offset - 2

	if jump > int(uint16Max) {
		c.errorAtPrevious("too long jump")
	}

	c.fn.Code[offset] = uint8((jump >> 8) & 0xff)
	c.fn.Code[offset+1] = uint8(jump & 0xff)
}

func (c *compiler) emitJumpBack(loopStart int) {
	c.emit(opJumpBack)

	offset := len(c.fn.Code) - loopStart + 2
	if offset > int(uint16Max) {
		c.errorAtPrevious("too long jump")
	}

	c.emit(uint8((offset >> 8) & 0xff))
	c.emit(uint8(offset & 0xff))
}

func (c *compiler) beginScope() { c.scope++ }

func (c *compiler) endScope() {
	c.scope--

	for len(c.locals) > 0 && c.locals[len(c.locals)-1].depth > c.scope {
		local := c.locals[len(c.locals)-1]
		if local.isCaptured {
			c.emit(opCloseUpvalue)
		} else {
			c.emit(opPop)
		}
		c.locals = c.locals[:len(c.locals)-1]
	}
}

func (c *compiler) beginLoop(label string, loopType loopType) int {
	c.loop = &loop{label, loopType, len(c.fn.Code), nil, c.loop}
	return len(c.fn.Code)
}

func (c *compiler) endLoop() {
	for _, breakJump := range c.loop.breaks {
		c.patchJump(breakJump)
	}
	c.loop = c.loop.enclosing
}

func (c *compiler) argumentList() (uint8, bool) {
	var argCount uint8 = 0
	isSpread := false
	if !c.check(tokenRightParen) {
		for {
			c.expression()
			if c.match(tokenDotDotDot) {
				isSpread = true
			} else {
				argCount++
			}
			if argCount > fnMaxParams {
				c.errorAtPrevious(
					fmt.Sprintf("too many arguments (%d)", fnMaxParams),
				)
			}
			if !c.match(tokenComma) {
				break
			}
			if c.check(tokenRightParen) || isSpread {
				break
			}
		}
	}
	c.consume(tokenRightParen)
	return argCount, isSpread
}

func (c *compiler) parameterList() {
	if !c.check(tokenRightParen) {
		for {
			if c.match(tokenDotDotDot) {
				c.fn.Vararg = true
			} else {
				c.fn.ParamCount++
			}
			if c.fn.ParamCount > fnMaxParams {
				c.errorAtPrevious(
					fmt.Sprintf("too many parameters (%d)", fnMaxParams),
				)
			}
			c.defineVariable(c.declareVariable())
			if !c.match(tokenComma) {
				break
			}
			if c.check(tokenRightParen) || c.fn.Vararg {
				break
			}
		}
	}
	c.consume(tokenRightParen)
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

	tokenQuestion: precTern,
	tokenThen:     precTern,

	tokenPipePipe: precOr,
	tokenOr:       precOr,

	tokenAmperAmper: precAnd,
	tokenAnd:        precAnd,

	tokenEqualEqual: precEq,
	tokenBangEqual:  precEq,

	tokenLeftAngle:       precComp,
	tokenRightAngle:      precComp,
	tokenLeftAngleEqual:  precComp,
	tokenRightAngleEqual: precComp,

	tokenPlus:  precTerm,
	tokenMinus: precTerm,

	tokenStar:    precFact,
	tokenSlash:   precFact,
	tokenPercent: precFact,

	tokenLeftParen:       precCall,
	tokenLeftBracket:     precCall,
	tokenDot:             precCall,
	tokenMinusRightAngle: precCall,
}

/* == token reader ========================================================== */

type tokenReader struct {
	scanner
	next     token
	current  token
	previous token
	hadError bool
	panic    bool
}

func newTokenReader(source []byte) *tokenReader {
	p := &tokenReader{scanner: newScanner(source)}
	p.advance()
	p.advance()
	return p
}

func (r *tokenReader) errorAt(token *token, message string) {
	if r.panic {
		return
	}
	r.panic = true
	if !r.hadError {
		fmt.Fprint(os.Stderr, "compile error: ")
	} else {
		fmt.Fprint(os.Stderr, "  also ")
	}
	fmt.Fprintf(os.Stderr, "ln %d: %s", token.line, message)

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
	r.current = r.next
	for {
		r.next = r.scanner.scan()
		if r.next.tokenType != tokenError {
			break
		}
		r.errorAt(&r.next, r.next.literal)
	}
}

func (r *tokenReader) consume(t tokenType) {
	if r.current.tokenType == t {
		r.advance()
		return
	}
	r.errorAtCurrent(fmt.Sprintf("'%s' expected", t))
}

func (r *tokenReader) consumeSemicolon() {
	if modeAutoSemicolons {
		if !r.match(tokenNewLine) && !r.match(tokenEof) {
			r.consume(tokenSemicolon)
		}
	} else {
		r.consume(tokenSemicolon)
	}
}

func (r *tokenReader) matchSemicolon() bool {
	if modeAutoSemicolons {
		return r.match(tokenNewLine) || r.match(tokenSemicolon)
	} else {
		return r.match(tokenSemicolon)
	}
}

func (r *tokenReader) consumeEnd() {
	if modeAutoSemicolons {
		_ = r.match(tokenNewLine) || r.match(tokenSemicolon)
	} else {
		r.consumeSemicolon()
	}
}

func (r *tokenReader) ignoreNewLine() { r.match(tokenNewLine) }

func (r *tokenReader) check(t tokenType) bool {
	return r.current.tokenType == t
}

func (r *tokenReader) checkNext(t tokenType) bool {
	return r.next.tokenType == t
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
		if r.previous.tokenType == tokenSemicolon ||
			r.previous.tokenType == tokenNewLine {
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

type compUpval struct {
	IsLocal bool  `json:"is_local"`
	Index   uint8 `json:"index"`
}

type parseFn func(canAssign bool)

type localVar struct {
	name          string
	depth         int
	isInitialized bool
	isCaptured    bool
}

type loopType int

const (
	loopLoop loopType = iota
	loopBlock
	loopSwitch
)

type loop struct {
	label string
	loopType
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
	fnTypeDefault
)
