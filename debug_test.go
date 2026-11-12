package debug

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Общие шаблоны сообщений тестов пакета.
const (
	errUnexpected string = "неожиданная ошибка: %v"
	errGotWant    string = "получено %q, ожидалось %q"
	errNoObjRef   string = "ожидалась ссылка на уже выведенный объект, получено:\n%s"
	tplObjRef     string = "(obj @"
)

// envBoolCase — набор значений для проверки одной ветки разбора envBool.
type envBoolCase struct {
	val  string
	def  bool
	want bool
}

// envBool: все ветки разбора значений переменных окружения.
func TestEnvBool(t *testing.T) {
	const (
		envName   string = "DEBUG_TEST_BOOL"
		errResult string = "envBool(%q, %v) = %v, ожидалось %v"
	)
	var (
		cases []envBoolCase
		c     envBoolCase
		got   bool
	)

	cases = []envBoolCase{
		{"", true, true},
		{"", false, false},
		{"true", false, true},
		{"TRUE", false, true},
		{" True ", false, true},
		{"1", false, true},
		{"t", false, true},
		{"yes", false, true},
		{"y", false, true},
		{"on", false, true},
		{"false", true, false},
		{"FALSE", true, false},
		{" false ", true, false},
		{"0", true, false},
		{"f", true, false},
		{"no", true, false},
		{"n", true, false},
		{"off", true, false},
		{"неизвестное", true, true},
		{"неизвестное", false, false},
		{"2", true, true},
	}
	for _, c = range cases {
		t.Setenv(envName, c.val)
		if got = envBool(envName, c.def); got != c.want {
			t.Errorf(errResult, c.val, c.def, got, c.want)
		}
	}
}

// Final с пустым буфером возвращает пустой результат.
func TestFinalEmpty(t *testing.T) {
	const errEmptyResult string = "ожидался пустой результат, получено %q"
	var got string

	if got = newDebug().Final().String(); got != "" {
		t.Errorf(errEmptyResult, got)
	}
}

// Final сохраняет строку без завершающего перевода строки.
func TestFinalNoTrailingNewline(t *testing.T) {
	const tplWant string = "abc\n"
	var (
		d   *debug
		got string
	)

	d = newDebug()
	_, _ = d.ReadWriter.WriteString("abc")
	if got = d.Final().String(); got != tplWant {
		t.Errorf(errGotWant, got, tplWant)
	}
}

// Final сохраняет пустые строки и использует терминатор CRLF.
func TestFinalCRLFAndEmptyLines(t *testing.T) {
	const (
		tplBody string = "a\n\nb\n"
		tplWant string = "a\r\n\r\nb\r\n"
	)
	var (
		d   *debug
		got string
	)

	d = newDebug()
	d.UseCRLF = true
	_, _ = d.ReadWriter.WriteString(tplBody)
	if got = d.Final().String(); got != tplWant {
		t.Errorf(errGotWant, got, tplWant)
	}
}

// Final обрабатывает строки длиннее буфера bufio (фрагменты с isPrefix).
func TestFinalLongLine(t *testing.T) {
	const (
		tplRepeat string = "x"
		tplPrefix string = "xxx"
		errLong   string = "длинная строка обработана неверно, длина результата %d"
		// Размер больше буфера bufio.Reader (4096).
		size int = 1<<13 + 100
	)
	var (
		d   *debug
		lng string
		got string
	)

	d = newDebug()
	lng = strings.Repeat(tplRepeat, size)
	_, _ = d.ReadWriter.WriteString(lng + "\n")
	got = d.Final().String()
	// Объединённая строка условия превысила бы 120 символов, присвоение вынесено отдельно.
	if len(got) != len(lng)+1 || !strings.HasPrefix(got, tplPrefix) || !strings.HasSuffix(got, "\n") {
		t.Errorf(errLong, len(got))
	}
}

// Dump с включённым флагом трассировки добавляет стек вызовов.
func TestDumpWithTrace(t *testing.T) {
	const (
		tplGoroutine string = "goroutine"
		errNoTrace   string = "ожидался стек вызовов в дампе:\n%s"
	)
	var out string

	seeTrace.Store(true)
	defer seeTrace.Store(false)
	if out = DumperString(1); !strings.Contains(out, tplGoroutine) {
		t.Errorf(errNoTrace, out)
	}
}

// Nop с включённым флагом DEBUG_CALLS выводит место вызова.
func TestNopSeeCalls(t *testing.T) {
	seeCalls.Store(true)
	defer seeCalls.Store(false)

	Nop() // вывод уходит в stdout теста
}

// Дамп карты, содержащей саму себя, не должен зацикливаться.
func TestDumperCyclicMap(t *testing.T) {
	var (
		m   map[string]any
		out string
	)

	m = map[string]any{}
	m["self"] = m
	if out = DumperString(m); !strings.Contains(out, tplObjRef) {
		t.Errorf(errNoObjRef, out)
	}
}

// Дамп среза, содержащего сам себя, не должен зацикливаться.
func TestDumperCyclicSlice(t *testing.T) {
	var (
		s   []any
		out string
	)

	s = make([]any, 1)
	s[0] = s
	if out = DumperString(s); !strings.Contains(out, tplObjRef) {
		t.Errorf(errNoObjRef, out)
	}
}

// Дамп структуры с циклической ссылкой через указатель не должен зацикливаться.
func TestDumperCyclicPointer(t *testing.T) {
	type node struct {
		Next *node
	}
	var (
		n   *node
		out string
	)

	n = &node{}
	n.Next = n
	if out = DumperString(n); !strings.Contains(out, tplObjRef) {
		t.Errorf(errNoObjRef, out)
	}
}

// Ошибка выводится своим строковым представлением.
func TestDumperError(t *testing.T) {
	const (
		tplErrText string = "ошибка для теста"
		tplWant    string = "error \"ошибка для теста\""
		errNoOut   string = "ожидался вывод ошибки, получено:\n%s"
	)
	var out string

	if out = DumperString(errors.New(tplErrText)); !strings.Contains(out, tplWant) {
		t.Errorf(errNoOut, out)
	}
}

// Конкурентное использование вывода и переключения стека вызовов не должно вызывать гонок.
func TestConcurrentUse(t *testing.T) {
	const goroutines int = 8
	var (
		wg sync.WaitGroup
		i  int
	)

	for i = range goroutines {
		wg.Add(1)
		// Значение счётчика передаётся параметром: общая переменная цикла между горутинами создаёт гонку.
		go func(v int) {
			defer wg.Done()
			DumperString(v)
			SeeCallStack(true)
			Nop()
		}(i)
	}
	wg.Wait()
}

// Dumper выводит дамп в STDOUT без паники.
func TestDumper(t *testing.T) {
	Dumper(1, "строка")
}

// DumperByte возвращает дамп в виде среза байт.
func TestDumperByte(t *testing.T) {
	const errEmptyDump string = "ожидался непустой дамп"
	var b []byte

	if b = DumperByte(42); len(b) == 0 {
		t.Error(errEmptyDump)
	}
}

// DumperFile сохраняет дамп в файл с именем программы в текущей директории.
func TestDumperFile(t *testing.T) {
	const (
		tplExt       string = ".txt"
		tplBody      string = "данные"
		errChdirFail string = "не удалось сменить директорию: %v"
		errSkipGetwd string = "не удалось определить текущую директорию: %v"
		errNoDump    string = "файл дампа %q не создан: %v"
	)
	var (
		orig string
		dir  string
		want string
		err  error
	)

	if orig, err = os.Getwd(); err != nil {
		t.Skipf(errSkipGetwd, err)
	}
	dir = t.TempDir()
	if err = os.Chdir(dir); err != nil {
		t.Fatalf(errChdirFail, err)
	}
	defer func() { _ = os.Chdir(orig) }()
	DumperFile(1, tplBody)
	want = filepath.Join(dir, filepath.Base(os.Args[0])+tplExt)
	if _, err = os.Stat(want); err != nil {
		t.Errorf(errNoDump, want, err)
	}
}
