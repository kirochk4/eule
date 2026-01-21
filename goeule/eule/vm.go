package eule

import (
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/bits"
	"os"
)

//go:embed include/include.eul
var include []byte

const (
	framesMax int = 64
	stackMax      = framesMax * uint8Count
)

var (
	ErrInterpretRuntimeError = errors.New("runtime error")
	ErrInterpretCompileError = errors.New("compile error")
)

type throwError error

type callFrame struct {
	fn     *Function
	cursor int
	slots  int
	upvals []*Upvalue
}

type tryHandler struct {
	cst  int
	st   int
	rcvr int
}

func (f *callFrame) readByte() uint8 {
	f.cursor++
	return f.fn.Code[f.cursor-1]
}

func (f *callFrame) readShort() uint16 {
	big := f.readByte()
	small := f.readByte()
	return uint16(big)<<8 | uint16(small)
}

func (f *callFrame) readConstant() Value {
	return f.fn.Constants[f.readByte()]
}

func (f *callFrame) readString() String {
	return f.readConstant().(String)
}

type VM struct {
	callStack  [framesMax]callFrame
	cst        int
	stack      [stackMax]Value
	st         int
	Global     *Table
	openUpvals *Upvalue
	try        []tryHandler
	arrayProto *Table
}

func New() *VM {
	vm := &VM{
		callStack: [framesMax]callFrame{},
		stack:     [stackMax]Value{},
		Global:    newTable(tableCapacity, nil),
	}

	vm.Global.Store(String("print"), Native(nativePrint))
	vm.Global.Store(String("clock"), Native(nativeClock))
	vm.Global.Store(String("assert"), Native(nativeAssert))
	vm.Global.Store(String("setPrototype"), Native(nativeSetPrototype))
	vm.Global.Store(String("getPrototype"), Native(nativeGetPrototype))
	vm.Global.Store(String("error"), Native(nativeError))

	vm.Interpret(include)
	vm.arrayProto = vm.Global.Load(magicArray).(*Table)
	return vm
}

func (vm *VM) Interpret(source []byte) error {
	fn := newCompiler(source).compile()
	if fn == nil {
		return ErrInterpretCompileError
	}

	if debugPrintBytecode {
		printBytecode(fn)
	}

	vm.push(fn)
	vm.callFunction(fn, 0, nil)

	return vm.run()
}

func (vm *VM) currentFrame() *callFrame {
	return &vm.callStack[vm.cst-1]
}

func (vm *VM) run() (err error) {
	frame := vm.currentFrame()
	throwString := func(format string, a ...any) {
		if err := vm.throw(&frame, format, a...); err != nil {
			panic(throwError(err))
		}
	}
	throwValue := func(v Value) {
		throwString("%v", v)
	}
	defer catch(func(e throwError) { err = e })

	for {
		if debugTraceExecution {
			printInstruction(frame.fn, frame.cursor)
			fmt.Print("|: ")
			for i := 0; i < vm.st; i++ {
				fmt.Printf("[%v] ", vm.stack[i])
			}
			fmt.Println()
		}

		switch op := frame.readByte(); op {
		case opOpenTry:
			offset := int(frame.readShort())
			vm.try = append(vm.try, tryHandler{
				vm.cst, vm.st,
				frame.cursor + offset,
			})
		case opCloseTry:
			slicePop(&vm.try)
			val := vm.pop()
			r := newTable(2, nil)
			r.Store(magicValue, val)
			r.Store(magicError, Boolean(false))
			vm.push(r)
		case opPop:
			vm.pop()
		case opDup:
			vm.push(vm.peek(0))
		case opDupTwo:
			vm.push(vm.peek(1))
			vm.push(vm.peek(1))
		case opSwap:
			vm.stack[vm.st-1], vm.stack[vm.st-2] =
				vm.stack[vm.st-2], vm.stack[vm.st-1]
		case opNihil:
			vm.push(Nihil{})
		case opFalse:
			vm.push(Boolean(false))
		case opTrue:
			vm.push(Boolean(true))
		case opSmallInteger:
			vm.push(Number(frame.readByte()))
		case opConstant:
			vm.push(frame.readConstant())
		case opTable:
			vm.push(newTable(tableCapacity, nil))
		case opArray:
			array := newTable(tableCapacity, vm.arrayProto)
			array.Store(magicLength, Number(0))
			vm.push(array)
		case opAddArrayElement:
			array := vm.peek(1).(*Table)
			oldLength := array.Load(magicLength).(Number)
			newLength := oldLength + Number(1)
			array.Store(oldLength, vm.pop())
			array.Store(magicLength, newLength)
		case opAddArraySpread:
			array := vm.peek(1).(*Table)
			oldLength := array.Load(magicLength).(Number)
			newLength := oldLength

			spr := vm.pop()
			switch spr := spr.(type) {
			case Nihil:
			case *Table:
				if length, ok := spr.Load(magicLength).(Number); ok {
					var i Number
					for i = 0; i < length; i++ {
						array.Store(oldLength+i, spr.Load(i))
					}
					newLength = oldLength + length
				}
			default:
				throwString("attempt to spread %s", typeOf(spr))
			}

			array.Store(magicLength, newLength)
		case opClosure:
			fn := vm.pop().(*Function)
			cls := &Closure{fn, nil}
			vm.push(cls)
			for _, upval := range fn.Upvals {
				if upval.IsLocal {
					cls.upvals = append(
						cls.upvals,
						vm.captureUpvalue(frame.slots+int(upval.Index)),
					)
				} else {
					cls.upvals = append(cls.upvals, frame.upvals[upval.Index])
				}
			}
		case opCloseUpvalue:
			vm.closeUpvalues(vm.st - 1)
			vm.pop()
		case opStoreUpvalue:
			index := int(frame.readByte())
			upval := frame.upvals[index]
			upval.Store(vm.peek(0))
		case opLoadUpvalue:
			index := int(frame.readByte())
			upval := frame.upvals[index]
			vm.push(upval.Load())
		case opStoreTemp:
			vm.stack[frame.slots-1] = vm.peek(0)
		case opLoadTemp:
			vm.stack[vm.st-1] = vm.stack[frame.slots-1]
		case opAddTableKey:
			value := vm.pop()
			key := vm.pop()
			table := vm.peek(0).(*Table)
			table.Store(key, value)
		case opAddTableSpread:
			spr := vm.pop()
			table := vm.peek(0).(*Table)
			switch spr := spr.(type) {
			case Nihil:
			case *Table:
				maps.Copy(table.Pairs, spr.Pairs)
			default:
				throwString("attempt to spread %s", typeOf(spr))
			}
		case opStoreKey:
			value := vm.pop()
			key := vm.pop()
			object := vm.pop()
			table, ok := object.(*Table)
			if !ok {
				throwString("attempt to store key in %s", typeOf(object))
			}
			vm.push(table.Store(key, value))
		case opLoadKey:
			key := vm.pop()
			object := vm.pop()
			table, ok := object.(*Table)
			if !ok {
				throwString("attempt to load key from %s", typeOf(object))
			}
			vm.push(table.Load(key))
		case opStoreLocal:
			slot := int(frame.readByte())
			vm.stack[frame.slots+slot] = vm.peek(0)
		case opLoadLocal:
			slot := int(frame.readByte())
			vm.push(vm.stack[frame.slots+slot])
		case opDefineGlobal:
			name := frame.readString()
			vm.Global.Pairs[name] = vm.pop()
		case opStoreGlobal:
			name := frame.readString()
			if _, ok := vm.Global.Pairs[name]; !ok {
				throwString("variable '%s' is undefined", name)
			}
			vm.Global.Pairs[name] = vm.peek(0)
		case opLoadGlobal:
			name := frame.readString()
			if value, ok := vm.Global.Pairs[name]; !ok {
				throwString("variable '%s' is undefined", name)
			} else {
				vm.push(value)
			}
		case opEq:
			v2 := vm.pop()
			v1 := vm.pop()
			vm.push(Boolean(v1 == v2))
		case opAdd:
			v2 := vm.pop()
			v1 := vm.pop()
			if str1, str2, ok := assertValues[String](v1, v2); ok {
				vm.push(str1 + str2)
			} else if num1, num2, ok := assertValues[Number](v1, v2); ok {
				vm.push(num1 + num2)
			} else {
				throwString(
					"attempt to add %s and %s",
					typeOf(v1), typeOf(v2),
				)
			}
		case opLt, opLe, opSub, opMul, opDiv, opMod:
			v2 := vm.pop()
			v1 := vm.pop()
			if num1, num2, ok := assertValues[Number](v1, v2); ok {
				vm.push(numOps[op](num1, num2))
			} else {
				throwString(
					"attempt to %s %s and %s",
					opNames[op], typeOf(v1), typeOf(v2),
				)
			}
		case opOr:
			v2 := vm.pop()
			v1 := vm.pop()
			if b1, b2, ok := assertValues[Boolean](v1, v2); ok {
				vm.push(b1 || b2)
			} else {
				vm.push(Number(toBitwise(v1) | toBitwise(v2)))
			}
			panic(unreachable)
		case opXor:
			v2 := vm.pop()
			v1 := vm.pop()
			if b1, b2, ok := assertValues[Boolean](v1, v2); ok {
				vm.push(b1 || b2)
			} else {
				vm.push(Number(toBitwise(v1) ^ toBitwise(v2)))
			}
			panic(unreachable)
		case opAnd:
			v2 := vm.pop()
			v1 := vm.pop()
			if b1, b2, ok := assertValues[Boolean](v1, v2); ok {
				vm.push(b1 && b2)
			} else {
				vm.push(Number(toBitwise(v1) & toBitwise(v2)))
			}
			panic(unreachable)
		case opRev:
			v := vm.pop()
			if b, ok := assertValue[Boolean](v); ok {
				vm.push(!b)
			} else {
				vm.push(Number(bits.Reverse64(toBitwise(v))))
			}
			panic(unreachable)
		case opNot:
			vm.push(!toBoolean(vm.pop()))
		case opNeg:
			v, isNumber := vm.peek(0).(Number)
			if !isNumber {
				throwString(
					"attempt to %s %s",
					opNames[op], typeOf(v),
				)
			}
			vm.pop()
			vm.push(-v)
		case opPos:
			v, isNumber := vm.peek(0).(Number)
			if !isNumber {
				throwString(
					"attempt to %s %s",
					opNames[op], typeOf(v),
				)
			}
			vm.pop()
			vm.push(Number(math.Abs(float64(v))))
		case opTypeOf:
			vm.push(typeOf(vm.pop()))
		case opJump:
			frame.cursor += int(frame.readShort())
		case opJumpIfFalse:
			offset := int(frame.readShort())
			if !toBoolean(vm.peek(0)) {
				frame.cursor += offset
			}
		case opJumpIfDone:
			offset := int(frame.readShort())
			obj := vm.pop()
			if tbl, ok := obj.(*Table); ok {
				if !toBoolean(tbl.Load(magicDone)) {
					vm.push(tbl.Load(magicValue))
					break
				}
			}
			frame.cursor += offset
		case opJumpBack:
			frame.cursor -= int(frame.readShort())
		case opCall:
			argCount := int(frame.readByte())
			if err := vm.callValue(vm.peek(argCount), argCount); err != nil {
				throwValue(err)
			}
			frame = vm.currentFrame()
		case opCallSpread:
			argCount := int(frame.readByte())

			spr := vm.pop()
			switch spr := spr.(type) {
			case Nihil:
			case *Table:
				if length, ok := spr.Load(magicLength).(Number); ok {
					var i Number
					for i = 0; i < length; i++ {
						vm.push(spr.Load(i))
						argCount++
					}
				}
			default:
				throwString("attempt to spread %s", typeOf(spr))
			}

			if err := vm.callValue(vm.peek(argCount), argCount); err != nil {
				throwValue(err)
			}
			frame = vm.currentFrame()

		case opReturn:
			result := vm.pop()
			vm.closeUpvalues(frame.slots)
			vm.st = frame.slots - 1
			vm.push(result)
			vm.cst--
			if vm.cst == 0 {
				vm.pop()
				return nil
			}
			frame = vm.currentFrame()
		default:
			panic(unreachable)
		}

	}
}

func (vm *VM) captureUpvalue(loc int) *Upvalue {
	var prev *Upvalue = nil
	upval := vm.openUpvals

	for upval != nil && upval.loc > loc {
		prev = upval
		upval = upval.next
	}

	if upval != nil && upval.loc == loc {
		return upval
	}

	new := &Upvalue{loc, &vm.stack, nil, upval}
	if prev != nil {
		prev.next = new
	} else {
		vm.openUpvals = new
	}
	return new
}

func (vm *VM) closeUpvalues(locOfLast int) {
	for vm.openUpvals != nil && vm.openUpvals.loc >= locOfLast {
		upval := vm.openUpvals
		upval.clsd = upval.Load()
		upval.loc = -1
		vm.openUpvals = upval.next
	}
}

func (vm *VM) callValue(value Value, argCount int) Value {
	switch callee := value.(type) {
	case *Closure:
		return vm.callFunction(callee.fn, argCount, callee.upvals)
	case *Function:
		return vm.callFunction(callee, argCount, nil)
	case Native:
		return vm.callNative(callee, argCount)
	default:
		return sprintString(
			"%s is not callable", typeOf(value),
		)
	}
}

func (vm *VM) balanceArguments(argCount, paramCount int, hasVararg bool) {
	var vararg Value

	if argCount <= paramCount {
		for range paramCount - argCount {
			vm.push(Nihil{})
		}
		if hasVararg {
			vararg = Nihil{}
		}
	} else {
		shift := argCount - paramCount
		if hasVararg {
			tbl := newTable(shift+1, nil)
			for i := range shift {
				tbl.Store(Number(i), vm.peek(shift-i-1))
			}
			tbl.Store(magicLength, Number(shift))
			vararg = tbl
		}
		vm.st -= shift
	}

	if hasVararg {
		vm.push(vararg)
	}
}

func (vm *VM) callFunction(
	fn *Function,
	argCount int,
	upvals []*Upvalue,
) Value {
	if vm.cst == framesMax {
		return String("stack overflow")
	}
	vm.callStack[vm.cst] = callFrame{fn, 0, vm.st - argCount, upvals}
	vm.cst++
	vm.balanceArguments(argCount, fn.ParamCount, fn.Vararg)
	return nil
}

func (vm *VM) callNative(fn Native, argCount int) Value {
	args := vm.stack[vm.st-argCount : vm.st]
	vm.st = vm.st - argCount - 1
	if val, err := fn(vm, args); err != nil {
		return err
	} else {
		vm.push(val)
		return nil
	}
}

func (vm *VM) push(value Value) {
	vm.stack[vm.st] = value
	vm.st++
}

func (vm *VM) pop() Value {
	vm.st--
	return vm.stack[vm.st]
}

func (vm *VM) peek(distance int) Value {
	return vm.stack[vm.st-1-distance]
}

func (vm *VM) throw(frame **callFrame, format string, a ...any) error {
	if vm.unwind(frame) {
		r := newTable(2, nil)
		r.Store(magicValue, String(fmt.Sprintf(format, a...)))
		r.Store(magicError, Boolean(true))
		vm.push(r)
		return nil
	}

	return vm.runtimeError(format, a...)
}

func (vm *VM) unwind(frame **callFrame) bool {
	if len(vm.try) != 0 {
		try := slicePop(&vm.try)
		vm.st, vm.cst = try.st, try.cst
		vm.closeUpvalues(try.st)
		*frame = vm.currentFrame()
		(*frame).cursor = try.rcvr
		return true
	}
	return false
}

func (vm *VM) runtimeError(format string, a ...any) error {
	fmt.Fprint(os.Stderr, "runtime error: ")
	fmt.Fprintf(os.Stderr, format+"\n", a...)

	for i := vm.cst - 1; i >= 0; i-- {
		frame := &vm.callStack[i]
		fn := frame.fn
		line := fn.Lines[frame.cursor]
		fmt.Fprintf(os.Stderr, "  ln %d: fn %s\n", line, fn.Name)
	}

	return ErrInterpretRuntimeError
}

var numOps = map[uint8]func(a, b Number) Value{
	opLt:  func(a, b Number) Value { return Boolean(a < b) },
	opLe:  func(a, b Number) Value { return Boolean(a <= b) },
	opAdd: func(a, b Number) Value { return a + b },
	opSub: func(a, b Number) Value { return a - b },
	opMul: func(a, b Number) Value { return a * b },
	opDiv: func(a, b Number) Value { return a / b },
	opMod: func(a, b Number) Value { return mod(a, b) },
}
