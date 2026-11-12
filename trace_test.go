package debug

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// Общие шаблоны сообщений тестов трассировки.
const errBadFuncName string = "неверное имя функции: %q"
const errEmptyStack string = "стек вызовов пуст"

// newTrace возвращает непустой объект трассировки.
func TestNewTrace(t *testing.T) {
	const errNilTrace string = "newTrace() вернул nil"
	var a *trace

	if a = newTrace(); a == nil {
		t.Error(errNilTrace)
	}
}

// Trace с уровнем 1 указывает на вызывающую функцию.
func TestTraceLevel(t *testing.T) {
	const (
		tplFunc      = "TestTraceLevel"
		tplFile      = "trace_test.go"
		tplSuffix    = "webnice/debug"
		errBadFileNm = "неверное имя файла: %q"
		errBadLine   = "неверный номер строки: %d"
		errBadPkg    = "неверное имя пакета: %q"
	)
	var a *trace

	a = newTrace().Trace(1)
	// Одно присваивание обслуживает несколько условий ниже: объединение в одну строку невозможно.
	if a.Function != tplFunc {
		t.Errorf(errBadFuncName, a.Function)
	}
	if !strings.HasSuffix(a.File, tplFile) {
		t.Errorf(errBadFileNm, a.File)
	}
	if a.Line <= 0 {
		t.Errorf(errBadLine, a.Line)
	}
	if !strings.HasSuffix(a.Package, tplSuffix) {
		t.Errorf(errBadPkg, a.Package)
	}
}

// Trace с уровнем 0 использует уровень по умолчанию и указывает на вызывающий код.
func TestTraceDefaultLevel(t *testing.T) {
	const (
		errNoFuncName = "имя функции не определено"
		errBadPlace   = "неверные место вызова: %s:%d"
	)
	var a *trace

	a = newTrace().Trace(0)
	// Уровень по умолчанию смотрит выше тестовой функции (на раннер тестов).
	// Одно присваивание обслуживает несколько условий ниже: объединение в одну строку невозможно.
	if a.Function == "" {
		t.Error(errNoFuncName)
	}
	if a.File == "" || a.Line <= 0 {
		t.Errorf(errBadPlace, a.File, a.Line)
	}
	if a.Stack == "" {
		t.Error(errEmptyStack)
	}
}

// Trace увеличивает буфер, пока трассировка всех горутин не поместится целиком.
func TestTraceStackGrowth(t *testing.T) {
	const (
		goroutines    int = 1000
		bufSize       int = 1 << 16
		errStackSmall     = "трассировка %d горутин (%d байт) должна превышать буфер 64 КБ и вызывать его рост"
	)
	var (
		release chan struct{}
		a       *trace
	)

	release = make(chan struct{})
	for range goroutines {
		go func() { <-release }()
	}
	// Дать горутинам запуститься и встать в ожидание.
	time.Sleep(100 * time.Millisecond)
	a = newTrace().Trace(1)
	close(release)
	if len(a.Stack) <= bufSize {
		t.Errorf(errStackSmall, goroutines, len(a.Stack))
	}
}

// Trace с заведомо большим уровнем: информация недоступна, поля остаются пустыми.
func TestTraceOutOfRange(t *testing.T) {
	const (
		level           int = 1 << 20
		errFieldsFilled     = "при выходе за пределы стека поля должны оставаться пустыми: %+v"
	)
	var a *trace

	if a = newTrace().Trace(level); a.File != "" || a.Line != 0 || a.Function != "" || a.Stack != "" {
		t.Errorf(errFieldsFilled, *a)
	}
}

// Стек вызовов не пуст и содержит текущую функцию.
func TestTraceStack(t *testing.T) {
	const (
		tplFunc          = "TestTraceStack"
		errNoSelfInStack = "стек вызовов не содержит имя текущей функции"
	)
	var a *trace

	a = newTrace().Trace(1)
	// Одно присваивание обслуживает несколько условий ниже: объединение в одну строку невозможно.
	if a.Stack == "" {
		t.Error(errEmptyStack)
	}
	if !strings.Contains(a.Stack, tplFunc) {
		t.Error(errNoSelfInStack)
	}
}

// funcNameByPC возвращает имя функции или пустую строку для несуществующего адреса.
func TestFuncNameByPC(t *testing.T) {
	const (
		tplSuffix      = "TestFuncNameByPC"
		errNoEmptyName = "для нулевого счётчика ожидалась пустая строка, получено %q"
	)
	var name string

	if name = funcNameByPC(0); name != "" {
		t.Errorf(errNoEmptyName, name)
	}
	if name = funcNameByPC(reflect.ValueOf(TestFuncNameByPC).Pointer()); !strings.HasSuffix(name, tplSuffix) {
		t.Errorf(errBadFuncName, name)
	}
}
