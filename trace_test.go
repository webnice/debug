package debug

import (
	"testing"
)

func TestNewTrace(t *testing.T) {
	a := newTrace()
	if a == nil {
		t.Errorf("Error in newTrace()")
	}
}
