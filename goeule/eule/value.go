package eule

import (
	"fmt"
	"strconv"
	"time"
)

type Value interface {
	fmt.Stringer
	valueMark()
}

type Nihil empty

func (n Nihil) Load(keyValue Value) Value {
	return Nihil{}
}

type Boolean bool

type Number float64

type String string

type Proto interface {
	Load(keyValue Value) Value
	Value
}

type Table struct {
	Pairs map[String]Value
	Proto
}

func (t *Table) Store(keyValue Value, value Value) Value {
	t.Pairs[toString(keyValue)] = value
	return value
}

func (t *Table) Load(keyValue Value) Value {
	if value, ok := t.Pairs[toString(keyValue)]; ok {
		return value
	} else {
		return t.Proto.Load(keyValue)
	}
}

func (t *Table) Has(keyValue Value) Boolean {
	_, ok := t.Pairs[toString(keyValue)]
	return Boolean(ok)
}

func (t *Table) Delete(keyValue Value) {
	delete(t.Pairs, toString(keyValue))
}

func newTable(cap int, proto Proto) *Table {
	if proto == nil {
		proto = Nihil{}
	}
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

type Native func(vm *VM, values []Value) (Value, Value)

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

func nativePrint(vm *VM, values []Value) (Value, Value) {
	for i, value := range values {
		fmt.Printf("%s", value)
		if i != len(values)-1 {
			fmt.Print(" ")
		}
	}
	fmt.Println()
	return Nihil{}, nil
}

func nativeClock(vm *VM, values []Value) (Value, Value) {
	return Number(float64(time.Now().UnixNano()) / float64(time.Second)), nil
}

func nativeAssert(vm *VM, values []Value) (Value, Value) {
	if len(values) < 1 {
		return nil, String("assert condition required")
	}
	if toBoolean(values[0]) {
		return Nihil{}, nil
	} else {
		return nil, String("assertion failed")
	}
}

func nativeSetPrototype(vm *VM, values []Value) (Value, Value) {
	if len(values) < 2 {
		return nil, String("not enough arguments")
	}
	if tbl, ok := values[0].(*Table); ok {
		if proto, ok := values[1].(Proto); ok {
			tbl.Proto = proto
			return tbl, nil
		}
	}
	return nil, String("wrong types")
}

func nativeGetPrototype(vm *VM, values []Value) (Value, Value) {
	if len(values) < 1 {
		return nil, String("not enough arguments")
	}
	if tbl, ok := values[0].(*Table); ok {
		return tbl.Proto, nil
	}
	return Nihil{}, nil
}

func nativeError(vm *VM, values []Value) (Value, Value) {
	if len(values) < 1 {
		return nil, Nihil{}
	}
	return nil, values[0]
}

func (v Nihil) String() string     { return nihilLiteral }
func (v Boolean) String() string   { return strconv.FormatBool(bool(v)) }
func (v Number) String() string    { return formatNumber(v) }
func (v String) String() string    { return string(v) }
func (v *Table) String() string    { return formatTable(v) }
func (v *Function) String() string { return fmt.Sprintf("<fn %s>", v.Name) }
func (v *Closure) String() string  { return v.fn.String() }
func (v Native) String() string    { return "<native fn>" }

func (v Nihil) valueMark()     {}
func (v Boolean) valueMark()   {}
func (v Number) valueMark()    {}
func (v String) valueMark()    {}
func (v *Table) valueMark()    {}
func (v *Function) valueMark() {}
func (v *Closure) valueMark()  {}
func (v Native) valueMark()    {}

func toString(v Value) String {
	switch v := v.(type) {
	case *Table:
		return "<table>"
	case *Function, *Closure, Native:
		return "<function>"
	default:
		return String(v.String())
	}
}

func sprintString(format string, a ...any) String {
	return String(fmt.Sprintf(format, a...))
}

func toBoolean(v Value) Boolean {
	switch v := v.(type) {
	case Nihil:
		return false
	case Boolean:
		return v
	default:
		return true
	}
}

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
