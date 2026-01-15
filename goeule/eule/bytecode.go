package eule

import "fmt"

const (
	opPop uint8 = iota
	opDup
	opSwap

	opNihil
	opFalse
	opTrue

	opSmallInteger // slow
	opConstant

	opStoreTemp
	opLoadTemp

	opDefineGlobal
	opStoreGlobal
	opLoadGlobal

	opStoreLocal
	opLoadLocal

	opDefineKey
	opStoreKey
	opLoadKey
	opLoadKeyNoPop

	opTable

	opAdd
	opSub
	opMul
	opDiv
	opEq
	opLt
	opLe
	opNot
	opNeg
	opPos
	opTypeOf

	opJump
	opJumpIfFalse
	opJumpBack

	opCall
	opReturn
)

func printBytecode(f *Function) {
	printFunctionCode(f)
	for _, c := range f.Constants {
		if f, ok := c.(*Function); ok {
			printFunctionCode(f)
		}
	}
}

func printFunctionCode(f *Function) {
	fmt.Println(coverString("function", 24, '='))

	for offset := 0; offset < len(f.Code); {
		offset = printInstruction(f, offset)
		fmt.Println()
	}
}

func printInstruction(f *Function, offset int) int {
	fmt.Printf("%04d", offset)
	if offset > 0 && f.Lines[offset] == f.Lines[offset-1] {
		fmt.Printf("   | ")
	} else {
		fmt.Printf("%4d ", f.Lines[offset])
	}

	switch op := f.Code[offset]; op {
	case opPop, opDup, opSwap, opNihil, opFalse, opTrue, opTable, opAdd, opSub,
		opMul, opDiv, opEq, opLt, opLe, opNot, opNeg, opPos, opTypeOf, opReturn,
		opStoreTemp, opLoadTemp, opDefineKey, opStoreKey, opLoadKey,
		opLoadKeyNoPop:
		return simpleInstruction(f, offset)
	case opConstant, opDefineGlobal, opStoreGlobal,
		opLoadGlobal:
		return constantInstruction(f, offset)
	case opSmallInteger, opCall, opStoreLocal, opLoadLocal:
		return byteInstruction(f, offset)
	case opJump, opJumpIfFalse, opJumpBack:
		sign := 1
		if op == opJumpBack {
			sign = -1
		}
		return jumpInstruction(f, offset, sign)
	default:
		panic(unreachable)
	}
}

func constantInstruction(f *Function, offset int) int {
	name := opNames[f.Code[offset]]
	index := f.Code[offset+1]
	fmt.Printf(
		"%-16s |> %04d %-8v ",
		name,
		index,
		shortString(f.Constants[index].String(), 8),
	)
	return offset + 2
}

func simpleInstruction(f *Function, offset int) int {
	name := opNames[f.Code[offset]]
	fmt.Printf("%-16s |%16c", name, ' ')
	return offset + 1
}

func byteInstruction(f *Function, offset int) int {
	name := opNames[f.Code[offset]]
	slot := f.Code[offset+1]
	fmt.Printf("%-16s |> %04d%10c", name, slot, ' ')
	return offset + 2
}

func jumpInstruction(f *Function, offset int, sign int) int {
	name := opNames[f.Code[offset]]
	jump := uint16(f.Code[offset+1]) << 8
	jump |= uint16(f.Code[offset+2])
	fmt.Printf("%-16s |> %04d >>> %04d ", name, offset, offset+3+sign*int(jump))
	return offset + 3
}

var opNames = [...]string{
	opPop:  "pop",
	opDup:  "dup",
	opSwap: "swap",

	opNihil: "nihil",
	opFalse: "false",
	opTrue:  "true",

	opSmallInteger: "small_integer",
	opConstant:     "constant",

	opStoreTemp: "store_temp",
	opLoadTemp:  "load_temp",

	opDefineGlobal: "define_global",
	opStoreGlobal:  "store_global",
	opLoadGlobal:   "load_global",

	opStoreLocal: "store_local",
	opLoadLocal:  "load_local",

	opDefineKey: "define_key",
	opStoreKey:  "store_key",
	opLoadKey:   "load_key",

	opTable: "table",

	opAdd:    "add",
	opSub:    "sub",
	opMul:    "mul",
	opDiv:    "div",
	opEq:     "eq",
	opLt:     "lt",
	opLe:     "le",
	opNot:    "not",
	opNeg:    "neg",
	opPos:    "pos",
	opTypeOf: "type_of",

	opJump:        "jump",
	opJumpIfFalse: "jump_if_false",
	opJumpBack:    "jump_back",

	opCall:   "call",
	opReturn: "return",
}
