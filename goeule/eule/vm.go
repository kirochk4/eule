package eule

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
	"os"
)

const (
	framesMax int = 64
	stackMax      = framesMax * uint8Count

	globalInitCapacity = 8
)

var (
	ErrInterpretRuntimeError = errors.New("runtime error")
	ErrInterpretCompileError = errors.New("compile error")
)

type callFrame struct {
	fn     *Function
	cursor int
	slots  int
	upvals []*Upvalue
}

type protect struct {
	cst int
	st  int
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
	prot       []protect
}

func New() *VM {
	vm := &VM{
		callStack: [framesMax]callFrame{},
		stack:     [stackMax]Value{},
		Global:    newTable(globalInitCapacity, nil),
	}
	vm.Global.Store(String("print"), Native(nativePrint))
	vm.Global.Store(String("clock"), Native(nativeClock))
	vm.Global.Store(String("assert"), Native(nativeAssert))
	vm.Global.Store(String("proto"), Native(nativeSetPrototype))
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

func (vm *VM) run() error {
	frame := vm.currentFrame()

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
		case opPop:
			vm.pop()
		case opDup:
			vm.push(vm.peek(0))
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
			vm.push(newTable(0, nil))
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
		case opDefineKey:
			value := vm.pop()
			key := vm.pop()
			table := vm.peek(0).(*Table)
			table.Store(key, value)
		case opStoreKey:
			value := vm.pop()
			key := vm.pop()
			object := vm.pop()
			table, ok := object.(*Table)
			if !ok {
				return vm.runtimeError(
					"attempt to store key in %s",
					typeOf(object),
				)
			}
			vm.push(table.Store(key, value))
		case opLoadKey:
			key := vm.pop()
			object := vm.pop()
			table, ok := object.(*Table)
			if !ok {
				return vm.runtimeError(
					"attempt to load key from %s",
					typeOf(object),
				)
			}
			vm.push(table.Load(key))
		case opLoadKeyNoPop:
			key := vm.peek(0)
			object := vm.peek(1)
			table, ok := object.(*Table)
			if !ok {
				return vm.runtimeError(
					"attempt to load key from %s",
					typeOf(object),
				)
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
				return vm.runtimeError("variable '%s' is undefined", name)
			}
			vm.Global.Pairs[name] = vm.peek(0)
		case opLoadGlobal:
			name := frame.readString()
			if value, ok := vm.Global.Pairs[name]; !ok {
				return vm.runtimeError("variable '%s' is undefined", name)
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
				return vm.runtimeError(
					"attempt to add %s and %s", typeOf(v1), typeOf(v2),
				)
			}
		case opLt, opLe, opSub, opMul, opDiv, opMod:
			v2 := vm.pop()
			v1 := vm.pop()
			if num1, num2, ok := assertValues[Number](v1, v2); ok {
				vm.push(numOps[op](num1, num2))
			} else {
				return vm.runtimeError(
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
			vm.push(!vm.pop().toBoolean())
		case opNeg:
			v, isNumber := vm.peek(0).(Number)
			if !isNumber {
				return vm.runtimeError(
					"attempt to %s %s",
					opNames[op], typeOf(v),
				)
			}
			vm.pop()
			vm.push(-v)
		case opPos:
			v, isNumber := vm.peek(0).(Number)
			if !isNumber {
				return vm.runtimeError(
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
			if !vm.peek(0).toBoolean() {
				frame.cursor += offset
			}
		case opJumpIfNihil:
			offset := int(frame.readShort())
			if isNihil(vm.peek(0)) {
				frame.cursor += offset
			}
		case opJumpBack:
			frame.cursor -= int(frame.readShort())
		case opCall:
			argCount := int(frame.readByte())
			if err := vm.callValue(vm.peek(argCount), argCount); err != nil {
				return err
			}
			frame = vm.currentFrame()

		case opReturn:
			result := vm.pop()
			vm.closeUpvalues(frame.slots)
			vm.st = frame.slots - 1
			vm.push(result)
			vm.cst--
			if vm.cst == 0 {
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

func (vm *VM) callValue(value Value, argCount int) error {
	switch callee := value.(type) {
	case *Closure:
		return vm.callFunction(callee.fn, argCount, callee.upvals)
	case *Function:
		return vm.callFunction(callee, argCount, nil)
	case Native:
		return vm.callNative(callee, argCount)
	default:
		return vm.runtimeError(
			"%s is not callable", typeOf(value),
		)
	}
}

func (vm *VM) balanceArguments(argCount, paramCount int, vararg bool) {
	var tbl *Table
	if vararg {
		paramCount--
	}

	if argCount <= paramCount {
		for range paramCount - argCount {
			vm.push(Nihil{})
		}
		if vararg {
			tbl = newTable(0, nil)
			tbl.Store(String("length"), Number(0))
		}
	} else {
		shift := argCount - paramCount
		if vararg {
			tbl = newTable(shift, nil)
			for i := range shift {
				tbl.Store(Number(i), vm.peek(shift-i-1))
			}
			tbl.Store(String("length"), Number(shift))
		}
		vm.st -= shift
	}

	if vararg {
		vm.push(tbl)
	}
}

func (vm *VM) callFunction(
	fn *Function,
	argCount int,
	upvals []*Upvalue,
) error {
	if vm.cst == framesMax {
		return vm.runtimeError("stack overflow")
	}
	vm.callStack[vm.cst] = callFrame{fn, 0, vm.st - argCount, upvals}
	vm.cst++
	vm.balanceArguments(argCount, fn.ParamCount, fn.Vararg)
	return nil
}

func (vm *VM) callNative(fn Native, argCount int) error {
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

func (vm *VM) unwind(errValue Value) bool {
	if len(vm.prot) != 0 {
		prot := slicePop(&vm.prot)
		vm.st, vm.cst = prot.st, prot.cst
		vm.push(errValue)
		return true
	}
	return false
}

func (vm *VM) runtimeError(format string, a ...any) error {
	if vm.unwind(String(fmt.Sprintf(format, a...))) {
		return nil
	}

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
