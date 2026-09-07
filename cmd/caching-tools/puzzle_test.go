package main

import "testing"

func TestA1Z26(t *testing.T) {
	got := a1z26("Cache!")
	want := []int{3, 1, 3, 8, 5}
	if len(got.Values) != len(want) { t.Fatalf("values=%v", got.Values) }
	for i := range want { if got.Values[i] != want[i] { t.Fatalf("values=%v", got.Values) } }
	if got.Sum != 20 { t.Fatalf("sum=%d", got.Sum) }
}

func TestCaesar(t *testing.T) {
	if got := caesar("Abc Xyz!", 13); got != "Nop Klm!" { t.Fatalf("got=%q", got) }
	if got := caesar("ABC", -1); got != "ZAB" { t.Fatalf("got=%q", got) }
}

func TestDigitChecksum(t *testing.T) {
	sum, root := digitChecksum("GC 12-345")
	if sum != 15 || root != 6 { t.Fatalf("sum=%d root=%d", sum, root) }
}

func TestSubstitute(t *testing.T) {
	got, err := substitute("Abba!", "ABC", "XYZ")
	if err != nil { t.Fatal(err) }
	if got != "Xyyx!" { t.Fatalf("got=%q", got) }
	if _, err := substitute("A", "ABC", "XY"); err == nil { t.Fatal("expected length error") }
}
