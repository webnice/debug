package debug

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"io"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

// Общие шаблоны сообщений тестов принтера.
const errNoWriteErr string = "ожидалась ошибка записи"

// withString — значение с методом String() верной сигнатуры.
type withString struct{ V int }

// String возвращает строковое представление значения.
func (w withString) String() string {
	const tplStringer string = "withString=%d"

	return fmt.Sprintf(tplStringer, w.V)
}

// withBadStringInt — значение с методом String() неверной сигнатуры.
type withBadStringInt struct{ V int }

// String имеет неверную сигнатуру и не должна использоваться принтером.
//
//goland:noinspection ALL
func (w withBadStringInt) String(int) string {
	const tplBad string = "неверная сигнатура"

	return tplBad
}

// withBadStringMulti — значение с методом String(), возвращающим два значения.
type withBadStringMulti struct{ V int }

// String имеет неверную сигнатуру и не должна использоваться принтером.
func (w withBadStringMulti) String() (string, error) { return "", nil }

// stringSlice — именованный срез с методом String().
type stringSlice []int

// String возвращает строковое представление среза.
func (s stringSlice) String() string {
	const tplName string = "stringSlice"

	return tplName
}

// stringArray — именованный массив с методом String().
type stringArray [2]int

// String возвращает строковое представление массива.
func (a stringArray) String() string {
	const tplName string = "stringArray"

	return tplName
}

// badStringSlice — именованный срез с методом String() неверной сигнатуры.
type badStringSlice []int

// String имеет неверную сигнатуру и не должна использоваться принтером.
//
//goland:noinspection ALL
func (s badStringSlice) String(int) string {
	const tplBad string = "неверная сигнатура"

	return tplBad
}

// badStringArray — именованный массив с методом String() неверной сигнатуры.
type badStringArray [2]int

// String имеет неверную сигнатуру и не должна использоваться принтером.
func (a badStringArray) String() (string, error) { return "", nil }

// valueError — ошибка с методом Error() на приёмнике-значении.
type valueError struct{ Msg string }

// Error возвращает текст ошибки.
func (e valueError) Error() string { return e.Msg }

// pointerError — ошибка с методом Error() на приёмнике-указателе.
type pointerError struct{ Msg string }

// Error возвращает текст ошибки.
func (e *pointerError) Error() string { return e.Msg }

// stringerWithPanic — значение, метод String() которого всегда паникует.
type stringerWithPanic struct{}

// String всегда паникует.
func (stringerWithPanic) String() string {
	const errPanic string = "сбой в String()"

	panic(errPanic)
}

// writerWithFailures падает с ошибкой записи после заданного числа успешных записей.
type writerWithFailures struct {
	left int // сколько записей ещё разрешено
}

// Write уменьшает счётчик разрешённых записей и возвращает ошибку после его исчерпания.
func (w *writerWithFailures) Write(p []byte) (n int, err error) {
	const errWriteFail string = "сбой записи"

	if w.left <= 0 {
		err = errors.New(errWriteFail)
		return
	}
	w.left--
	n = len(p)

	return
}

// Вывод значений всех основных видов (kind).
func TestPrintKinds(t *testing.T) {
	type nested struct {
		Int    int
		Uint   uint
		Float  float64
		Bool   bool
		Str    string
		Bytes  []byte
		Ch     chan int
		Fn     func()
		If     any
		Arr    [2]int
		Slice  []int
		EmptyS []int
		EmptyM map[string]int
		Uptr   uintptr
		Unsafe unsafe.Pointer
		Cplx   complex128
		Skip   *int // nil — отфильтрован
	}
	const (
		tplBytes     string = "ab"
		tplHex       string = "0x"
		tplSkip      string = "Skip"
		errNotInOut  string = "в выводе нет %q:\n%s"
		errNoAddr    string = "канал, функция и указатель не выведены:\n%s"
		errNotFilter string = "nil-поле должно быть отфильтровано:\n%s"
	)
	var (
		marker int
		v      nested
		out    string
		wants  []string
		want   string
	)

	marker = 0
	v = nested{
		Int: 1, Uint: 2, Float: 3.5, Bool: true, Str: "текст",
		Bytes: []byte(tplBytes), Ch: make(chan int), Fn: func() {},
		If: "значение", Arr: [2]int{7, 8}, Slice: []int{9},
		EmptyS: []int{}, EmptyM: map[string]int{}, Uptr: 10,
		Unsafe: unsafe.Pointer(&marker), Cplx: 1 + 2i,
	}
	out = DumperString(v)
	// Проверка наличия ключевых фрагментов вывода.
	wants = []string{
		"3.5", "true", "\"текст\"", "`ab`", "7", "8", "9",
		"(1+2i)", "len = 0",
	}
	for _, want = range wants {
		if !strings.Contains(out, want) {
			t.Errorf(errNotInOut, want, out)
		}
	}
	// Канал, функция и указатель выводятся своими адресами.
	if !strings.Contains(out, tplHex) {
		t.Errorf(errNoAddr, out)
	}
	// Nil-поле отфильтровано и не выводится.
	if strings.Contains(out, tplSkip) {
		t.Errorf(errNotFilter, out)
	}
}

// Ключи map выводятся в отсортированном порядке.
func TestPrintMapSorted(t *testing.T) {
	const (
		tplOne      string = "один"
		tplTwo      string = "два"
		tplThree    string = "три"
		tplQOne     string = `"один"`
		tplQTwo     string = `"два"`
		tplQThree   string = `"три"`
		errBadOrder string = "ключи выведены не по порядку:\n%s"
	)
	var (
		m   map[int]string
		out string
		i1  int
		i2  int
		i3  int
	)

	m = map[int]string{3: tplThree, 1: tplOne, 2: tplTwo}
	out = DumperString(m)
	i1 = strings.Index(out, tplQOne)
	i2 = strings.Index(out, tplQTwo)
	i3 = strings.Index(out, tplQThree)
	// Условие использует три переменные, присвоенные выше: объединение в одну строку невозможно.
	if !(i1 >= 0 && i1 < i2 && i2 < i3) {
		t.Errorf(errBadOrder, out)
	}
}

// Структура с методом String() выводится через него.
func TestPrintStructWithString(t *testing.T) {
	const (
		tplWant       string = `"withString=5"`
		errNoStringer string = "метод String() не использован:\n%s"
	)
	var out string

	if out = DumperString(withString{V: 5}); !strings.Contains(out, tplWant) {
		t.Errorf(errNoStringer, out)
	}
}

// Метод String() с неверной сигнатурой не приводит к панике.
func TestPrintStructBadString(t *testing.T) {
	const (
		tplBadSig      string = "%!STRING"
		errBadSig      string = "ожидалась пометка о неверной сигнатуре:\n%s"
		errBadSigSlice string = "ожидалась пометка о неверной сигнатуре среза:\n%s"
		errBadSigArray string = "ожидалась пометка о неверной сигнатуре массива:\n%s"
	)
	var out string

	if out = DumperString(withBadStringInt{V: 1}); !strings.Contains(out, tplBadSig) {
		t.Errorf(errBadSig, out)
	}
	if out = DumperString(withBadStringMulti{V: 1}); !strings.Contains(out, tplBadSig) {
		t.Errorf(errBadSig, out)
	}
	if out = DumperString(badStringSlice{1}); !strings.Contains(out, tplBadSig) {
		t.Errorf(errBadSigSlice, out)
	}
	if out = DumperString(badStringArray{1, 2}); !strings.Contains(out, tplBadSig) {
		t.Errorf(errBadSigArray, out)
	}
}

// Именованные срез и массив с методом String() выводятся через него.
func TestPrintSliceArrayWithString(t *testing.T) {
	const (
		tplWantSlice       string = `"stringSlice"`
		tplWantArray       string = `"stringArray"`
		errNoSliceStringer string = "метод String() среза не использован:\n%s"
		errNoArrayStringer string = "метод String() массива не использован:\n%s"
	)
	var out string

	if out = DumperString(stringSlice{1, 2}); !strings.Contains(out, tplWantSlice) {
		t.Errorf(errNoSliceStringer, out)
	}
	if out = DumperString(stringArray{1, 2}); !strings.Contains(out, tplWantArray) {
		t.Errorf(errNoArrayStringer, out)
	}
}

// Поле структуры, имеющее метод String(), выводится через него.
func TestPrintStructFieldWithString(t *testing.T) {
	type holder struct{ W withString }
	const (
		tplWant            string = `"withString=9"`
		errNoFieldStringer string = "метод String() поля не использован:\n%s"
	)
	var out string

	if out = DumperString(holder{W: withString{V: 9}}); !strings.Contains(out, tplWant) {
		t.Errorf(errNoFieldStringer, out)
	}
}

// Структура только с неэкспортированными полями выводится без полей.
func TestPrintUnexportedOnly(t *testing.T) {
	type hidden struct{ secret int }
	const (
		tplWant      string = "hidden {}"
		errHasFields string = "ожидался вывод без полей:\n%s"
	)
	var out string

	if out = DumperString(hidden{1}); !strings.Contains(out, tplWant) {
		t.Errorf(errHasFields, out)
	}
}

// Ошибки с разными приёмниками выводятся своим строковым представлением.
func TestPrintErrors(t *testing.T) {
	const (
		tplMsgValue   string = "значение"
		tplMsgPointer string = "указатель"
		tplWantValue  string = `error "значение"`
		tplWantPtr    string = `*error "указатель"`
		errValueError string = "ошибка со значением-приёмником выведена неверно:\n%s"
		errPointerErr string = "ошибка с указателем-приёмником выведена неверно:\n%s"
	)
	var out string

	if out = DumperString(valueError{Msg: tplMsgValue}); !strings.Contains(out, tplWantValue) {
		t.Errorf(errValueError, out)
	}
	if out = DumperString(&pointerError{Msg: tplMsgPointer}); !strings.Contains(out, tplWantPtr) {
		t.Errorf(errPointerErr, out)
	}
}

// token.Pos выводится через FileSet или как число без него.
func TestPrintTokenPos(t *testing.T) {
	const (
		tplFile      string = "test.go"
		tplNumber    string = "6"
		errNoPosFile string = "ожидалась позиция с файлом, получено: %s"
		errNoPosNum  string = "ожидалась числовая позиция, получено: %s"
	)
	var (
		buf  bytes.Buffer
		fset *token.FileSet
		f    *token.File
		pos  token.Pos
		err  error
	)

	fset = token.NewFileSet()
	f = fset.AddFile(tplFile, 1, 100)
	pos = f.Pos(5)
	if err = printerPrint(&buf, fset, pos, nil); err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if !strings.Contains(buf.String(), tplFile) {
		t.Errorf(errNoPosFile, buf.String())
	}
	// Без FileSet позиция выводится числом.
	buf.Reset()
	if err = printerPrint(&buf, nil, pos, nil); err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if !strings.Contains(buf.String(), tplNumber) {
		t.Errorf(errNoPosNum, buf.String())
	}
}

// Дамп nil выводится строкой nil.
func TestDumpNil(t *testing.T) {
	const (
		tplNilOut string = "nil"
		errNoNil  string = "ожидался вывод nil:\n%s"
	)
	var out string

	if out = DumperString(nil); !strings.Contains(out, tplNilOut) {
		t.Errorf(errNoNil, out)
	}
}

// printerPrint с фильтром nil выводит все поля структуры.
func TestPrintStructNilFilter(t *testing.T) {
	type sample struct {
		Shown  int
		Hidden *int
	}
	const (
		tplShown        string = "Shown"
		tplHidden       string = "Hidden"
		errNotAllFields string = "ожидались все поля структуры: %s"
	)
	var (
		buf bytes.Buffer
		err error
	)

	if err = printerPrint(&buf, nil, sample{Shown: 1}, nil); err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if !strings.Contains(buf.String(), tplShown) || !strings.Contains(buf.String(), tplHidden) {
		t.Errorf(errNotAllFields, buf.String())
	}
}

// Ошибка записи возвращается, а не приводит к панике.
func TestPrinterPrintWriteError(t *testing.T) {
	var err error

	if err = printerPrint(&writerWithFailures{}, token.NewFileSet(), 42, notNilFilter); err == nil {
		t.Error(errNoWriteErr)
	}
}

// Неизвестная паника внутри вывода пробрасывается дальше, а не маскируется.
func TestPrinterPrintRepanic(t *testing.T) {
	const (
		errNoPanic      string = "ожидалась проброшенная паника"
		errLocalPanicRe string = "паника localError не должна пробрасываться"
	)
	defer func() {
		var (
			r  any
			ok bool
		)

		r = recover()
		if r == nil {
			t.Error(errNoPanic)
		}
		if _, ok = r.(localError); ok {
			t.Error(errLocalPanicRe)
		}
	}()
	_ = printerPrint(io.Discard, token.NewFileSet(), stringerWithPanic{}, notNilFilter)
}

// printStringer напрямую: метод String отсутствует или имеет неверную сигнатуру.
func TestPrintStringerDirect(t *testing.T) {
	const (
		errNoStringerFalse string = "для значения без String() ожидалось false"
		errBadSigFalse     string = "для String() с неверной сигнатурой ожидалось false"
		errNoStringerTrue  string = "для String() с верной сигнатурой ожидалось true"
	)
	var (
		buf bytes.Buffer
		p   printer
	)

	p = printer{output: &buf}
	if p.printStringer(reflect.ValueOf(42)) {
		t.Error(errNoStringerFalse)
	}
	if p.printStringer(reflect.ValueOf(withBadStringInt{})) {
		t.Error(errBadSigFalse)
	}
	if !p.printStringer(reflect.ValueOf(withString{V: 1})) {
		t.Error(errNoStringerTrue)
	}
}

// printerAsError напрямую: все варианты реализации интерфейса error.
func TestPrinterAsError(t *testing.T) {
	type (
		wrapValue   struct{ e valueError }
		wrapPointer struct{ e pointerError }
	)
	const (
		tplVV          string = "вв"
		tplPE          string = "пе"
		tplNA          string = "на"
		tplHidden      string = "скрыто"
		errNoValue     string = "значение с Error() на приёмнике-значении не распознано"
		errNoAddrOk    string = "адресуемое значение с Error() на указателе не распознано"
		errNoAddrBad   string = "не адресуемое значение с Error() на указателе не должно распознаваться"
		errIntIsError  string = "int не должен распознаваться как error"
		errNoIface     string = "значение без доступа к интерфейсу не должно распознаваться"
		errNoAddrIface string = "адрес без доступа к интерфейсу не должен распознаваться"
	)
	var (
		ok bool
		wv reflect.Value
		wp reflect.Value
	)

	// Значение реализует error.
	if _, ok = printerAsError(reflect.ValueOf(valueError{Msg: tplVV})); !ok {
		t.Error(errNoValue)
	}
	// Указатель реализует error, значение адресуемо.
	if _, ok = printerAsError(reflect.ValueOf(&pointerError{Msg: tplPE}).Elem()); !ok {
		t.Error(errNoAddrOk)
	}
	// Указатель реализует error, значение не адресуемо.
	if _, ok = printerAsError(reflect.ValueOf(pointerError{Msg: tplNA})); ok {
		t.Error(errNoAddrBad)
	}
	// Не ошибка.
	if _, ok = printerAsError(reflect.ValueOf(1)); ok {
		t.Error(errIntIsError)
	}
	// Неэкспортированное поле: Interface() недоступен.
	wv = reflect.ValueOf(&wrapValue{e: valueError{Msg: tplHidden}}).Elem().Field(0)
	if _, ok = printerAsError(wv); ok {
		t.Error(errNoIface)
	}
	// Неэкспортированное поле, Error() на указателе: Addr().Interface() недоступен.
	wp = reflect.ValueOf(&wrapPointer{e: pointerError{Msg: tplHidden}}).Elem().Field(0)
	if _, ok = printerAsError(wp); ok {
		t.Error(errNoAddrIface)
	}
}

// Write: ошибка при записи фрагмента с переводом строки.
func TestWriteNewlineError(t *testing.T) {
	const (
		tplData  string = "ab\n"
		errWrite string = "ожидалась ошибка записи, n=%d err=%v"
	)
	var (
		n   int
		err error
		p   printer
	)

	p = printer{output: &writerWithFailures{}, lastByteWrite: 'x'}
	if n, err = p.Write([]byte(tplData)); err == nil || n != 0 {
		t.Errorf(errWrite, n, err)
	}
}

// Write: ошибка при записи префикса с номером строки.
func TestWritePrefixError(t *testing.T) {
	const (
		tplData          string = "abc"
		errNoPrefixWrite string = "ожидалась ошибка записи префикса"
	)
	var (
		err error
		p   printer
	)

	p = printer{output: &writerWithFailures{}, lastByteWrite: '\n'}
	if _, err = p.Write([]byte(tplData)); err == nil {
		t.Error(errNoPrefixWrite)
	}
}

// Write: ошибка при записи отступа.
func TestWriteIndentError(t *testing.T) {
	const (
		tplData          string = "ab\ncd"
		errNoIndentWrite string = "ожидалась ошибка записи отступа"
	)
	var (
		err error
		p   printer
	)

	p = printer{output: &writerWithFailures{left: 1}, lastByteWrite: '\n', indentLevel: 1}
	if _, err = p.Write([]byte(tplData)); err == nil {
		t.Error(errNoIndentWrite)
	}
}

// Write: ошибка при записи хвоста без перевода строки.
func TestWriteTailError(t *testing.T) {
	const (
		tplData        string = "abc"
		errNoTailWrite string = "ожидалась ошибка записи хвоста"
	)
	var (
		err error
		p   printer
	)

	p = printer{output: &writerWithFailures{left: 1}, lastByteWrite: '\n'}
	if _, err = p.Write([]byte(tplData)); err == nil {
		t.Error(errNoTailWrite)
	}
}

// Write: нумерация строк и отступы в успешном сценарии.
func TestWriteNumberingAndIndent(t *testing.T) {
	const (
		tplData string = "ab\ncd\nef"
		tplWant string = "000000: \tab\n000001: \tcd\n000002: \tef"
	)
	var (
		buf bytes.Buffer
		err error
		p   printer
	)

	p = printer{output: &buf, lastByteWrite: '\n', indentLevel: 1}
	if _, err = p.Write([]byte(tplData)); err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if buf.String() != tplWant {
		t.Errorf(errGotWant, buf.String(), tplWant)
	}
}
