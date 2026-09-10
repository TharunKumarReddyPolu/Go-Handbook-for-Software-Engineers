package main

import (
	"reflect"
	"testing"
)

func TestCollect_OrderPreserved(t *testing.T) {
	got := Collect(Produce(5))
	want := []int{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect(Produce(5)) = %v, want %v", got, want)
	}
}

func TestCollect_ZeroItems(t *testing.T) {
	if got := Collect(Produce(0)); len(got) != 0 {
		t.Errorf("Collect(Produce(0)) = %v, want empty", got)
	}
}
