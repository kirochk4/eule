package eule

import (
	"errors"
	"fmt"
	"math"
	"os"
)

const (
	framesMax int = 64
	stackMax      = framesMax * uint8Count

	globalTableInitCapacity = 8
)

var (
	ErrInterpretRuntimeError = errors.New("runtime error")
	ErrInterpretCompileError = errors.New("compile error")
)

type callFrame struct {
	fn     *Function
	cursor int
	slots  int
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
	callStack [framesMax]callFrame
	cst       int
	stack     [stackMax]Value
	st        int
	Global    *Table
}

func New() *VM {
	vm := &VM{
		callStack: [framesMax]callFrame{},
		stack:     [stackMax]Value{},
		Global:    newTable(globalTableInitCapacity, nil),
	}
	vm.Global.Store(String("print"), Native(nativePrint))
	vm.Global.Store(String("clock"), Native(nativeClock))
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
	vm.callFunction(fn, 0)

	return vm.run()
}

func (vm *VM) run() error {
	frame := &vm.callStack[vm.cst-1]

	for {
		if debugTraceExecution {
			printInstruction(frame.fn, frame.cursor)
			fmt.Print("|: ")
			for i := 0; i < vm.st; i++ {
				fmt.Printf("[%v] ", vm.stack[i])
			}
			fmt.Println()
		}

		switch instr := frame.readByte(); instr {
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
				return vm.runtimeError("Only arrays and maps have indexes.")
			}
			vm.push(table.Store(key, value))
		case opLoadKey:
			key := vm.pop()
			object := vm.pop()
			table, ok := object.(*Table)
			if !ok {
				return vm.runtimeError("Only arrays and maps have indexes.")
			}
			vm.push(table.Load(key))
		case opLoadKeyNoPop:
			key := vm.peek(0)
			object := vm.peek(1)
			table, ok := object.(*Table)
			if !ok {
				return vm.runtimeError("Only arrays and maps have indexes.")
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
				return vm.runtimeError("Undefined variable '%s'.", name)
			}
			vm.Global.Pairs[name] = vm.peek(0)
		case opLoadGlobal:
			name := frame.readString()
			if value, ok := vm.Global.Pairs[name]; !ok {
				return vm.runtimeError("Undefined variable '%s'.", name)
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
					"Operands must be two numbers or two strings.",
				)
			}
		case opLt, opLe, opSub, opMul, opDiv:
			v2 := vm.pop()
			v1 := vm.pop()
			if num1, num2, ok := assertValues[Number](v1, v2); ok {
				vm.push(numOps[instr](num1, num2))
			} else {
				return vm.runtimeError("Operands must be numbers.")
			}
		case opNot:
			vm.push(!vm.pop().toBoolean())
		case opNeg:
			v, isNumber := vm.peek(0).(Number)
			if !isNumber {
				return vm.runtimeError("Operand must be a number.")
			}
			vm.pop()
			vm.push(-v)
		case opPos:
			v, isNumber := vm.peek(0).(Number)
			if !isNumber {
				return vm.runtimeError("Operand must be a number.")
			}
			vm.pop()
			vm.push(Number(math.Abs(float64(v))))
		case opTypeOf:
			vm.push(typeOf(vm.pop()))
		case opJump:
			frame.cursor += int(frame.readShort())
		case opJumpIfFalse:
			offset := int(frame.readShort())
			condition := vm.peek(0).toBoolean()
			if !condition {
				frame.cursor += offset
			}
		case opJumpBack:
			frame.cursor -= int(frame.readShort())
		case opCall:
			argCount := int(frame.readByte())
			if err := vm.callValue(vm.peek(argCount), argCount); err != nil {
				return err
			}
			frame = &vm.callStack[vm.cst-1]

		case opReturn:
			result := vm.pop()
			vm.st = frame.slots - 1
			vm.push(result)
			vm.cst--
			if vm.cst == 0 {
				return nil
			}
			frame = &vm.callStack[vm.cst-1]
		}
	}
}

func (vm *VM) callValue(value Value, argCount int) error {
	switch callee := value.(type) {
	case *Function:
		return vm.callFunction(callee, argCount)
	case Native:
		return vm.callNative(callee, argCount)
	default:
		return vm.runtimeError("Can only call functions.")
	}
}

func (vm *VM) callFunction(fn *Function, argCount int) error {
	if argCount < fn.Arity {
		for range fn.Arity - argCount {
			vm.push(Nihil{})
		}
	} else if argCount > fn.Arity {
		vm.st -= argCount - fn.Arity
	}
	if vm.cst == framesMax {
		return vm.runtimeError("Stack overflow.")
	}
	vm.callStack[vm.cst] = callFrame{fn, 0, vm.st - argCount}
	vm.cst++
	return nil
}

func (vm *VM) callNative(fn Native, argCount int) error {
	args := vm.stack[vm.st-argCount : vm.st]
	vm.st = vm.st - argCount - 1
	vm.push(fn(vm, args))
	return nil
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

func (vm *VM) runtimeError(format string, a ...any) error {
	fmt.Fprintf(os.Stderr, format+"\n", a...)

	for i := vm.cst - 1; i >= 0; i-- {
		frame := &vm.callStack[i]
		fn := frame.fn
		line := fn.Lines[frame.cursor]
		fmt.Fprintf(os.Stderr, "  [line %d] in function\n", line)
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
}
