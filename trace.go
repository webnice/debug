package debug

import (
	"runtime"
	"strings"
)

// newTrace создаёт пустой объект трассировки.
func newTrace() *trace {
	return new(trace)
}

// funcNameByPC возвращает полное имя функции по её программному счётчику
// или пустую строку, если функция с таким адресом не найдена.
func funcNameByPC(pc uintptr) (name string) {
	if fn := runtime.FuncForPC(pc); fn != nil {
		name = fn.Name()
	}

	return
}

// Trace заполняет информацию о вызывающем коде на уровне level вверх по стеку.
// При level = 0 используется значение по умолчанию traceStepBack.
func (t *trace) Trace(level int) (trc *trace) {
	var (
		ok  bool
		pc  uintptr
		buf []byte
		tmp []string
		n   int
	)

	if level == 0 {
		level = traceStepBack
	}
	pc, t.File, t.Line, ok = runtime.Caller(level)
	if ok {
		t.Function = funcNameByPC(pc)
		// Буфер увеличивается до тех пор, пока трассировка всех горутин помещается целиком.
		buf = make([]byte, 1<<16)
		for {
			n = runtime.Stack(buf, true)
			if n < len(buf) {
				break
			}
			buf = make([]byte, len(buf)*2)
		}
		t.Stack = string(buf[:n])
		// Полное имя функции разделяется на имя пакета и имя функции.
		if tmp = strings.Split(t.Function, packageSeparator); len(tmp) > 1 {
			t.Package += strings.Join(tmp[:len(tmp)-1], packageSeparator)
			t.Function = tmp[len(tmp)-1]
		}
		if tmp = strings.SplitN(t.Function, methodSeparator, 2); len(tmp) == 2 {
			if t.Package != "" {
				t.Package += packageSeparator
			}
			t.Package += tmp[0]
			t.Function = tmp[1]
		}
	}
	trc = t

	return
}
