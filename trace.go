package debug

import (
	"runtime"
	"strings"
)

const (
	traceStepBack    int    = 2
	packageSeparator string = `/`
)

// Trace
type trace struct {
	Package  string
	Function string
	File     string
	Line     int
	Stack    string
}

func newTrace() *trace {
	return new(trace)
}

func (t *trace) Trace(level int) *trace {
	var ok bool
	var pc uintptr
	var fn *runtime.Func
	var buf []byte
	var tmp []string
	var i int

	if level == 0 {
		level = traceStepBack
	}
	buf = make([]byte, 1<<16)
	pc, t.File, t.Line, ok = runtime.Caller(level)
	if ok == true {
		fn = runtime.FuncForPC(pc)
		if fn != nil {
			t.Function = fn.Name()
		}
		i = runtime.Stack(buf, true)
		t.Stack = string(buf[:i])

		tmp = strings.Split(t.Function, packageSeparator)
		if len(tmp) > 1 {
			t.Package += strings.Join(tmp[:len(tmp)-1], packageSeparator)
			t.Function = tmp[len(tmp)-1]
		}
		tmp = strings.SplitN(t.Function, `.`, 2)
		if len(tmp) == 2 {
			if t.Package != "" {
				t.Package += packageSeparator
			}
			t.Package += tmp[0]
			t.Function = tmp[1]
		}

	}
	return t
}
