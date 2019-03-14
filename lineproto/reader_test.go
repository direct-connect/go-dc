package lineproto

import (
	"bytes"
	"reflect"
	"testing"
)

func TestReader(t *testing.T) {
	var byts []byte
	byts = append(byts, []byte("$ZOn|")...)
	byts = append(byts, []byte{120, 156, 82, 241, 47, 201, 72, 45, 114,
		206, 207, 205, 77, 204, 75, 81, 40, 73, 45,
		46, 169, 1, 4, 0, 0, 255, 255, 69, 30, 7, 66}...)
	byts = append(byts, []byte("$Uncompressed|")...)

	r := NewReader(bytes.NewReader(byts), '|')

	l1Expected := []byte("$ZOn|")
	l1, err := r.ReadLine()
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(l1, l1Expected) == false {
		t.Fatalf("l1 error: %v vs %v", l1, l1Expected)
	}

	r.ActivateCompression()

	l2Expected := []byte("$OtherCommand test|")
	l2, err := r.ReadLine()
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(l2, l2Expected) == false {
		t.Fatalf("l2 error: %v vs %v", l2, l2Expected)
	}

	l3Expected := []byte("$Uncompressed|")
	l3, err := r.ReadLine()
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(l3, l3Expected) == false {
		t.Fatalf("l3 error: %v vs %v", l3, l3Expected)
	}
}
