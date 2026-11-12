package debug

import (
	"bufio"
	"bytes"
	"go/token"
	"io"
	"os"
	"reflect"
	"sync/atomic"
	"time"
	"unsafe"
)

// Шаблоны оформления отладочного вывода.
const (
	tplDelimiterBeg string = "[ %02d.%02d.%04d %02d:%02d:%02d:%010d ] ------------------------------------------------------------"
	tplDelimiterEnd string = "-----------------------------------------------------------------------------------------------"
	tplLineEnd      string = "\n"
	tplLineEndCRLF  string = "\r\n"
)

// Константы трассировки стека вызовов.
const (
	traceStepBack    int    = 2
	packageSeparator string = "/"
	methodSeparator  string = "."
)

// Шаблоны форматирования значений принтером.
const (
	tplStructOpen     string = "%s {"
	tplStructClose    string = "}"
	tplBadStringer    string = "%%!STRING(неподдерживаемая сигнатура метода)"
	tplObjectRef      string = "%s (obj @ %d)"
	tplObjectRefShort string = "(obj @ %d)"
	tplMapOpen        string = "%s (len = %d) {"
	tplSliceOpen      string = "%s (len = %d, cap = %d) {"
	tplNil            string = "nil"
	tplPointer        string = "*"
	tplHexQuoted      string = "%#q"
	tplQuoted         string = "%q"
	tplPos            string = "%s"
	tplValue          string = "%v"
	tplColon          string = ": "
)

// Переменные состояния пакета.
var (
	// defaultCRLF — признак окончания строки CRLF по умолчанию: для Windows или при значении
	// переменной окружения DEBUG_CRLF=true. Переопределяется значением DEBUG_CRLF.
	defaultCRLF bool
	// seeCalls — вывод места вызова для поиска забытых отладочных вызовов в большом проекте.
	// Включается значением переменной окружения DEBUG_CALLS=true.
	seeCalls atomic.Bool
	// seeTrace — вывод дампа стека вызовов. Включается значением переменной окружения DEBUG_TRACESTACK=true.
	seeTrace atomic.Bool
)

// Переменные, имитируемые тестами.
var (
	// getwd и mkdirAll вынесены в переменные, чтобы тесты могли имитировать их сбои.
	getwd    = os.Getwd
	mkdirAll = os.MkdirAll
)

// Переменные принтера.
var (
	// indent — отступ одного уровня вложенности при выводе значений.
	indent = []byte("\t")
	// errorInterfaceType — тип интерфейса error для проверки реализации значением.
	errorInterfaceType = reflect.TypeFor[error]()
)

// debug — объект формирования отладочного дампа.
type debug struct {
	Result     *bytes.Buffer
	Buffer     *bytes.Buffer
	ReadWriter *bufio.ReadWriter
	UseCRLF    bool
	Now        time.Time
	Trace      *trace
}

// Trace — информация о месте вызова и стеке вызовов.
type trace struct {
	Package  string
	Function string
	File     string
	Line     int
	Stack    string
}

type (
	// printerFieldFilter — функция фильтрации полей структуры при выводе.
	printerFieldFilter func(name string, value reflect.Value) bool
	// printer — внутренний принтер для рекурсивного вывода значений.
	printer struct {
		output            io.Writer
		fset              *token.FileSet
		filter            printerFieldFilter
		lineNumber        map[visit]int
		indentLevel       int
		lastByteWrite     byte
		currentLineNumber int
	}
	// localError — тип паники, через который ошибки пробрасываются к recover.
	localError struct {
		err error
	}
	// visit — идентификатор уже выводившегося объекта: тип и указатель на данные.
	// Используется для обнаружения циклических ссылок между объектами.
	visit struct {
		typ reflect.Type
		ptr unsafe.Pointer
	}
)
