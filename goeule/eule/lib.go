package eule

import (
	"fmt"
	"math"
	"strconv"
)

const Version = "0.0.0"

const (
	debugPrintTokens    = false
	debugPrintBytecode  = false
	debugTraceExecution = false
)

const (
	uint8Max  uint8  = math.MaxUint8
	uint16Max uint16 = math.MaxUint16

	uint8Count int = math.MaxUint8 + 1

	unreachable = "unreachable"

	nihilLiteral    = "void"
	variableLiteral = "var"
	functionLiteral = "function"
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
		return strconv.FormatFloat(float64(num), 'g', -1, 64)
	}
}

func formatTable(tbl *Table) string {
	return "<table>"
}

func zero[T any]() T {
	return map[int]T{}[0]
}

func mapHas[T comparable, U any](m map[T]U, k T) bool {
	_, ok := m[k]
	return ok
}
