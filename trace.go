package debug

import (
	"runtime"
	"strings"
)

const (
	traceStepBack int = 2
)

// Trace
type trace struct {
	Package string
	Function string
	File     string
	Line     int
	Stack    string
}

func NewTrace() *trace {
	return new(trace)
}

func (self *trace) Trace(level int) *trace {
	var ok bool
	var pc uintptr
	var buf []byte
	var tmp []string
	var i int

	if level == 0 {
		level = traceStepBack
	}
	buf = make([]byte, 1<<16)
	pc, self.File, self.Line, ok = runtime.Caller(level)
	if ok == true {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			self.Function = fn.Name()
		}
		i = runtime.Stack(buf, true)
		self.Stack = string(buf[:i])
	}
	tmp = strings.SplitN(self.Function, `.`, 2)
	if len(tmp) == 2 {
		self.Package, self.Function = tmp[0], tmp[1]
	}
	return self
}
