package calc

import (
	"errors"
	"math"
	"testing"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"positives", 2, 3, 5},
		{"negatives", -2, -3, -5},
		{"zero identity", 0, 7, 7},
		{"max int overflow wraps", math.MaxInt, 1, math.MinInt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Add(tt.a, tt.b); got != tt.want {
				t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDivide(t *testing.T) {
	tests := []struct {
		name    string
		a, b    int
		want    int
		wantErr error
	}{
		{"exact", 10, 2, 5, nil},
		{"truncates toward zero", -7, 2, -3, nil},
		{"negative divisor", 7, -2, -3, nil},
		{"by zero", 1, 0, 0, ErrDivideByZero},
		{"min int by -1 wraps", math.MinInt, -1, math.MinInt, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Divide(tt.a, tt.b)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Divide(%d, %d) error = %v, want %v", tt.a, tt.b, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Divide(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestEval(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		want    int
		wantErr error
	}{
		{"plus", "1+2", 3, nil},
		{"spaces", "10 / 2", 5, nil},
		{"negative left", "-3 * 4", -12, nil},
		{"minus", "9-4", 5, nil},
		{"divide by zero", "1/0", 0, ErrDivideByZero},
		{"empty", "", 0, nil}, // wantErr checked by type below
		{"missing op", "12", 0, nil},
		{"bad right", "1+two", 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Eval(tt.expr)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Eval(%q) error = %v, want %v", tt.expr, err, tt.wantErr)
				}
				return
			}
			if tt.name == "empty" || tt.name == "missing op" || tt.name == "bad right" {
				var se *ErrSyntax
				if !errors.As(err, &se) {
					t.Fatalf("Eval(%q) error = %v, want *ErrSyntax", tt.expr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tt.expr, err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %d, want %d", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEvalSyntaxPosition(t *testing.T) {
	_, err := Eval("1+two")
	var se *ErrSyntax
	if !errors.As(err, &se) {
		t.Fatalf("want *ErrSyntax, got %T", err)
	}
	if se.Pos != 2 {
		t.Errorf("ErrSyntax.Pos = %d, want 2 (right operand starts after '1+')", se.Pos)
	}
}
