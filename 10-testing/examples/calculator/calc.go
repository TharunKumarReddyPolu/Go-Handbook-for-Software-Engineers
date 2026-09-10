// Package calc is the handbook's testing subject: a tiny expression
// evaluator small enough to read, real enough to exercise table tests,
// doubles, fuzzing, and benchmarks. Companion to all 10-testing chapters.
package calc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrDivideByZero is returned by Divide on a zero divisor.
var ErrDivideByZero = errors.New("divide by zero")

// ErrSyntax reports malformed expressions; the wrapped position makes
// failures precise.
type ErrSyntax struct {
	Pos  int
	What string
}

func (e *ErrSyntax) Error() string {
	return fmt.Sprintf("syntax error at %d: %s", e.Pos, e.What)
}

// Add returns the sum of a and b (documented to wrap on overflow).
func Add(a, b int) int { return a + b }

// Divide returns a/b, or ErrDivideByZero.
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, ErrDivideByZero
	}
	return a / b, nil
}

// Eval evaluates a two-operand expression: "a op b", where op is one of
// + - * /. Spaces are optional. Examples: "1+2", "10 / 2", "-3 * 4".
func Eval(expr string) (int, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, &ErrSyntax{Pos: 0, What: "empty expression"}
	}

	opIdx := strings.IndexAny(expr[1:], "+-*/") // skip first char: it may be a sign
	if opIdx < 0 {
		return 0, &ErrSyntax{Pos: 0, What: "missing operator"}
	}
	opIdx++ // adjust for the slice offset

	left, err := strconv.Atoi(strings.TrimSpace(expr[:opIdx]))
	if err != nil {
		return 0, &ErrSyntax{Pos: 0, What: "bad left operand"}
	}
	op := expr[opIdx]
	rightStr := strings.TrimSpace(expr[opIdx+1:])
	right, err := strconv.Atoi(rightStr)
	if err != nil {
		return 0, &ErrSyntax{Pos: opIdx + 1, What: "bad right operand"}
	}

	switch op {
	case '+':
		return Add(left, right), nil
	case '-':
		return left - right, nil
	case '*':
		return left * right, nil
	case '/':
		return Divide(left, right)
	default:
		return 0, &ErrSyntax{Pos: opIdx, What: "unknown operator"}
	}
}
