package eule

import (
	"fmt"
	"strconv"
	"time"
)

type Value interface {
	fmt.Stringer
	toString() String
	toBoolean() Boolean
}

type Nihil empty

type Boolean bool

type Number float64

type String string

type Table struct {
	Pairs map[String]Value
	Meta  *Table
}

func (t *Table) Store(keyValue Value, value Value) Value {
	t.Pairs[keyValue.toString()] = value
	return value
}

func (t *Table) Load(keyValue Value) Value {
	return t.Pairs[keyValue.toString()]
}

func (t *Table) Has(keyValue Value) Boolean {
	_, ok := t.Pairs[keyValue.toString()]
	return Boolean(ok)
}

func (t *Table) Delete(keyValue Value) {
	delete(t.Pairs, keyValue.toString())
}

func newTable(cap int, meta *Table) *Table {
	return &Table{
		Pairs: make(map[String]Value, cap),
		Meta:  meta,
	}
}

type Function struct {
	Code      []uint8
	Constants []Value
	Lines     []int
	Arity     int
}

func (f *Function) addConstant(constant Value) int {
	for i, c := range f.Constants {
		if c == constant {
			return i
		}
	}
	f.Constants = append(f.Constants, constant)
	return len(f.Constants) - 1
}

func (f *Function) writeCode(code uint8, line int) {
	f.Code = append(f.Code, code)
	f.Lines = append(f.Lines, line)
}

func NewFunction() *Function {
	return &Function{}
}

type Native func(vm *VM, values []Value) Value

func nativePrint(vm *VM, values []Value) Value {
	for _, value := range values {
		fmt.Printf("%v ", value)
	}
	fmt.Println()
	return Nihil{}
}

func nativeClock(vm *VM, values []Value) Value {
	return Number(float64(time.Now().UnixNano()) / float64(time.Second))
}

func (v Nihil) String() string     { return "void" }
func (v Boolean) String() string   { return strconv.FormatBool(bool(v)) }
func (v Number) String() string    { return formatNumber(v) }
func (v String) String() string    { return string(v) }
func (v *Table) String() string    { return fmt.Sprintf("<table %p>", v) }
func (v *Function) String() string { return fmt.Sprintf("<fn %p>", v) }
func (v Native) String() string    { return fmt.Sprintf("<fn %p>", v) }

func (v Nihil) toString() String     { return String(v.String()) }
func (v Boolean) toString() String   { return String(v.String()) }
func (v Number) toString() String    { return String(v.String()) }
func (v String) toString() String    { return String(v.String()) }
func (v *Table) toString() String    { return String(v.String()) }
func (v *Function) toString() String { return String(v.String()) }
func (v Native) toString() String    { return String(v.String()) }

func (v Nihil) toBoolean() Boolean     { return false }
func (v Boolean) toBoolean() Boolean   { return v }
func (v Number) toBoolean() Boolean    { return true }
func (v String) toBoolean() Boolean    { return true }
func (v *Table) toBoolean() Boolean    { return true }
func (v *Function) toBoolean() Boolean { return true }
func (v Native) toBoolean() Boolean    { return true }

func isNihil(v Value) bool {
	_, ok := v.(Nihil)
	return ok
}

func typeOf(v Value) String {
	switch v.(type) {
	case Nihil:
		return "void"
	case Boolean:
		return "bool"
	case Number:
		return "num"
	case String:
		return "str"
	case *Table:
		return "tbl"
	case *Function:
		return "fn"
	case Native:
		return "fn"
	default:
		panic(unreachable)
	}
}

func assertValues[T Value](v1 Value, v2 Value) (T, T, bool) {
	va1, isType1 := v1.(T)
	va2, isType2 := v2.(T)
	if !isType1 || !isType2 {
		return zero[T](), zero[T](), false
	}
	return va1, va2, true
}
