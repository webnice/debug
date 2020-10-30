package debug

import "testing"

func TestNewTrace(t *testing.T) {
	var a = newTrace()

	if a == nil {
		t.Errorf("error in newTrace()")
	}
}
