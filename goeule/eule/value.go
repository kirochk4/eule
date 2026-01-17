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
	Proto *Table
}

func (t *Table) Store(keyValue Value, value Value) Value {
	t.Pairs[keyValue.toString()] = value
	return value
}

func (t *Table) Load(keyValue Value) Value {
	if value, ok := t.Pairs[keyValue.toString()]; ok {
		return value
	} else if t.Proto != nil {
		return t.Proto.Load(keyValue)
	}
	return Nihil{}
}

func (t *Table) Has(keyValue Value) Boolean {
	_, ok := t.Pairs[keyValue.toString()]
	return Boolean(ok)
}

func (t *Table) Delete(keyValue Value) {
	delete(t.Pairs, keyValue.toString())
}

func newTable(cap int, proto *Table) *Table {
	return &Table{
		Pairs: make(map[String]Value, cap),
		Proto: proto,
	}
}

type Function struct {
	Name       string      `json:"name"`
	Code       []uint8     `json:"code"`
	Constants  []Value     `json:"constants"`
	Lines      []int       `json:"lines"`
	Upvals     []compUpval `json:"upvalues"`
	ParamCount int         `json:"parameters"`
	Vararg     bool        `json:"vararg"`
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

func NewFunction(name string) *Function {
	if name == "" {
		name = "_"
	}
	return &Function{Name: name}
}

type Native func(vm *VM, values []Value) (Value, error)

type Closure struct {
	fn     *Function
	upvals []*Upvalue
}

type Upvalue struct {
	loc  int
	ref  *[stackMax]Value
	clsd Value
	next *Upvalue
}

func (u *Upvalue) Store(v Value) {
	if u.loc != -1 {
		(*u.ref)[u.loc] = v
	} else {
		u.clsd = v
	}
}

func (u *Upvalue) Load() Value {
	if u.loc != -1 {
		return (*u.ref)[u.loc]
	} else {
		return u.clsd
	}
}

func nativePrint(vm *VM, values []Value) (Value, error) {
	for i, value := range values {
		fmt.Printf("%v", value)
		if i != len(values)-1 {
			fmt.Print(" ")
		}
	}
	fmt.Println()
	return Nihil{}, nil
}

func nativeClock(vm *VM, values []Value) (Value, error) {
	return Number(float64(time.Now().UnixNano()) / float64(time.Second)), nil
}

func nativeAssert(vm *VM, values []Value) (Value, error) {
	if len(values) < 1 {
		vm.runtimeError("assert condition required")
	}
	if values[0].toBoolean() {
		return Nihil{}, nil
	} else {
		return Nihil{}, vm.runtimeError("assertion failed")
	}
}

func nativeSetPrototype(vm *VM, values []Value) (Value, error) {
	if len(values) < 2 {
		return Nihil{}, vm.runtimeError("not enough arguments")
	}
	if tbl, ok := values[0].(*Table); ok {
		if proto, ok := values[1].(*Table); ok {
			tbl.Proto = proto
			return tbl, nil
		} else if isNihil(values[1]) {
			tbl.Proto = nil
			return tbl, nil
		}
	}
	return Nihil{}, vm.runtimeError("wrong types")
}

func nativeGetPrototype(vm *VM, values []Value) (Value, error) {
	if len(values) < 1 {
		return Nihil{}, vm.runtimeError("not enough arguments")
	}
	if tbl, ok := values[0].(*Table); ok {
		if tbl.Proto != nil {
			return tbl.Proto, nil
		}
	}
	return Nihil{}, nil
}

func (v Nihil) String() string     { return nihilLiteral }
func (v Boolean) String() string   { return strconv.FormatBool(bool(v)) }
func (v Number) String() string    { return formatNumber(v) }
func (v String) String() string    { return string(v) }
func (v *Table) String() string    { return "<table>" /* formatTable(v) */ }
func (v *Function) String() string { return fmt.Sprintf("<fn %s>", v.Name) }
func (v *Closure) String() string  { return v.fn.String() }
func (v Native) String() string    { return "<native fn>" }

func (v Nihil) toString() String     { return String(v.String()) }
func (v Boolean) toString() String   { return String(v.String()) }
func (v Number) toString() String    { return String(v.String()) }
func (v String) toString() String    { return String(v.String()) }
func (v *Table) toString() String    { return "<table>" }
func (v *Function) toString() String { return String(v.String()) }
func (v *Closure) toString() String  { return String(v.String()) }
func (v Native) toString() String    { return String(v.String()) }

func (v Nihil) toBoolean() Boolean     { return false }
func (v Boolean) toBoolean() Boolean   { return v }
func (v Number) toBoolean() Boolean    { return true }
func (v String) toBoolean() Boolean    { return true }
func (v *Table) toBoolean() Boolean    { return true }
func (v *Function) toBoolean() Boolean { return true }
func (v *Closure) toBoolean() Boolean  { return true }
func (v Native) toBoolean() Boolean    { return true }

func isNihil(v Value) bool {
	_, ok := v.(Nihil)
	return ok
}

func typeOf(v Value) String {
	switch v.(type) {
	case Nihil:
		return nihilLiteral
	case Boolean:
		return "boolean"
	case Number:
		return "number"
	case String:
		return "string"
	case *Table:
		return "table"
	case *Function:
		return "function"
	case *Closure:
		return "function"
	case Native:
		return "function"
	default:
		panic(unreachable)
	}
}

func assertValue[T Value](v Value) (T, bool) {
	va, isType1 := v.(T)
	if !isType1 {
		return zero[T](), false
	}
	return va, true
}

func assertValues[T Value](v1 Value, v2 Value) (T, T, bool) {
	va1, isType1 := v1.(T)
	va2, isType2 := v2.(T)
	if !isType1 || !isType2 {
		return zero[T](), zero[T](), false
	}
	return va1, va2, true
}

func toBitwise(val Value) uint64 {
	switch val := val.(type) {
	case nil:
		return 0
	case Boolean:
		return uint64(boolToInt(bool(val)))
	case Number:
		return uint64(val)
	default:
		return 1
	}
}
