package eule

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const Version = "0.0.0"

const (
	debugPrintTokens    = false
	debugPrintBytecode  = false
	debugTraceExecution = false
)

const (
	modeAutoSemicolons = true
)

const (
	nul byte = 0

	uint8Max  uint8  = math.MaxUint8
	uint16Max uint16 = math.MaxUint16

	uint8Count int = math.MaxUint8 + 1

	unreachable = "unreachable"

	nihilLiteral    = "void"
	variableLiteral = "var"
	functionLiteral = "func"

	fnMaxParams = 16 // must be < 256

	useSmallInteger = false

	magicLength String = "length"
	magicArray  String = "__array"
	magicValue  String = "value"
	magicError  String = "error"
	magicDone   String = "done"

	tableCapacity = 32
)

type empty struct{}

func shortString(str string, length int) string {
	runes := []rune(str)
	if len(runes) <= length {
		return str
	}
	return string(runes[:length])
}

func coverString(str string, width int, char rune) string {
	toCover := width - (len(str) + 2)
	if toCover < 0 {
		return str
	}
	left := toCover / 2
	right := left + toCover%2
	return fmt.Sprintf(
		"%s %s %s",
		multiplyRune(char, left),
		str,
		multiplyRune(char, right),
	)
}

func multiplyRune(char rune, c int) string {
	runes := make([]rune, c)
	for i := range runes {
		runes[i] = char
	}
	return string(runes)
}

func boolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func intToBool(i int) bool {
	return i != 0
}

func formatNumber(num Number) string {
	f := float64(num)
	if math.IsNaN(f) {
		return "nan"
	} else if math.IsInf(f, 0) {
		if math.IsInf(f, 1) {
			return "inf"
		} else {
			return "-inf"
		}
	} else {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
}

func formatTable(tbl *Table) string {
	if len(tbl.Pairs) == 0 {
		return "{}"
	}
	var str strings.Builder
	str.WriteString("{ ")
	i := 0
	for k, v := range tbl.Pairs {
		str.WriteString(fmt.Sprintf("\"%s\": %s", k, toString(v)))
		if i != len(tbl.Pairs)-1 {
			str.WriteString(", ")
		}
		i++
	}
	str.WriteString(" }")
	return str.String()
}

func mod(a, b Number) Number {
	an, bn := float64(a), float64(b)
	mod := math.Mod(an, bn)
	return Number(mod)
}

func zero[T any]() (t T) { return }

func mapHas[T comparable, U any](m map[T]U, k T) bool {
	_, ok := m[k]
	return ok
}

func slicePop[T any](s *[]T) (t T) {
	if len(*s) == 0 {
		return
	}
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v
}

func slicePush[T any](s *[]T, a ...T) {
	*s = append(*s, a...)
}

func catch[E any](onCatch func(E)) {
	if p := recover(); p != nil {
		if pe, ok := p.(E); ok {
			onCatch(pe)
		} else {
			panic(p)
		}
	}
}
