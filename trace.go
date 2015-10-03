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

func (self *trace) Trace(level int) *trace {
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
	pc, self.File, self.Line, ok = runtime.Caller(level)
	if ok == true {
		fn = runtime.FuncForPC(pc)
		if fn != nil {
			self.Function = fn.Name()
		}
		i = runtime.Stack(buf, true)
		self.Stack = string(buf[:i])

		tmp = strings.Split(self.Function, packageSeparator)
		if len(tmp) > 1 {
			self.Package += strings.Join(tmp[:len(tmp)-1], packageSeparator)
			self.Function = tmp[len(tmp)-1]
		}
		tmp = strings.SplitN(self.Function, `.`, 2)
		if len(tmp) == 2 {
			if self.Package != "" {
				self.Package += packageSeparator
			}
			self.Package += tmp[0]
			self.Function = tmp[1]
		}

	}
	return self
}
