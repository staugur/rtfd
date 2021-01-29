package vars

import "testing"

func TestVar(t *testing.T) {

	v := BRK("p")
	if v != "p:builder" {
		t.Fatal("BRK error")
	}
}
