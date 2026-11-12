package debug

import (
	"bufio"
	"bytes"
	"fmt"
	"go/token"
	"os"
	"runtime"
	"strings"
	"time"
)

// init загружает настройки пакета из переменных окружения.
func init() {
	const (
		envCRLF  string = "DEBUG_CRLF"
		envCalls string = "DEBUG_CALLS"
		envTrace string = "DEBUG_TRACESTACK"
	)

	defaultCRLF = envBool(envCRLF, runtime.GOOS == "windows")
	seeCalls.Store(envBool(envCalls, false))
	seeTrace.Store(envBool(envTrace, false))
}

// envBool возвращает булево значение переменной окружения name.
// Если переменная не задана или содержит нераспознанное значение, возвращается значение def.
func envBool(name string, def bool) bool {
	var value string

	// Пустое значение: результат — значение по умолчанию.
	if value = strings.ToLower(strings.TrimSpace(os.Getenv(name))); value == "" {
		return def
	}
	// Распознанные литералы истины и лжи.
	switch value {
	case "1", "t", "true", "y", "yes", "on":
		return true
	case "0", "f", "false", "n", "no", "off":
		return false
	}

	return def
}

// newDebug создаёт объект отладочного дампа с текущим временем и информацией о месте вызова.
func newDebug() (obj *debug) {
	var buf *bytes.Buffer

	buf = &bytes.Buffer{}
	obj = &debug{
		Result:     &bytes.Buffer{},
		Buffer:     buf,
		ReadWriter: bufio.NewReadWriter(bufio.NewReader(buf), bufio.NewWriter(buf)),
		UseCRLF:    defaultCRLF,
		Now:        time.Now().In(time.Local),
		Trace:      newTrace().Trace(traceStepBack + 1),
	}

	return
}

// Dump выводит дамп всех переданных переменных.
func (d *debug) Dump(idl ...any) *debug {
	var (
		i  int
		fs *token.FileSet
	)

	for i = range idl {
		fs = token.NewFileSet()
		_ = printerPrint(d.ReadWriter, fs, idl[i], notNilFilter)
	}
	// Стек вызовов.
	if seeTrace.Load() {
		_, _ = d.ReadWriter.WriteString(d.Trace.Stack + tplLineEnd)
	}

	return d
}

// Prefix добавляет информацию перед дампом: разделитель с датой и место вызова.
func (d *debug) Prefix(fn string) *debug {
	const tplCallInfo = "[ %30s ] %s:%d [%s] [%s()]" + tplLineEnd

	_, _ = d.ReadWriter.WriteString(fmt.Sprintf(tplDelimiterBeg+tplLineEnd, d.Now.Day(), d.Now.Month(), d.Now.Year(), d.Now.Hour(), d.Now.Minute(), d.Now.Second(), d.Now.Nanosecond()))
	_, _ = d.ReadWriter.WriteString(fmt.Sprintf(tplCallInfo, fn, d.Trace.File, d.Trace.Line, d.Trace.Package, d.Trace.Function))

	return d
}

// Suffix добавляет информацию после дампа.
func (d *debug) Suffix() *debug {
	_, _ = d.ReadWriter.WriteString(tplDelimiterEnd + tplLineEnd)

	return d
}

// Final завершает формирование дампа - переносит накопленные строки в результат, заменяя окончания строк согласно
// настройке UseCRLF.
func (d *debug) Final() *bytes.Buffer {
	var (
		line     []byte
		isPrefix bool
		err      error
	)

	_ = d.ReadWriter.Flush()
	// Пустые строки дампа сохраняются, терминатор выбирается по настройке UseCRLF.
	// bufio.Reader.ReadLine возвращает ошибку только вместе с пустой строкой,
	// поэтому выход из цикла — только через проверку в начале итерации.
	for {
		if line, isPrefix, err = d.ReadWriter.ReadLine(); err != nil && len(line) == 0 {
			break
		}
		_, _ = d.Result.Write(line)
		if isPrefix {
			continue
		}
		if d.UseCRLF {
			_, _ = d.Result.WriteString(tplLineEndCRLF)
		} else {
			_, _ = d.Result.WriteString(tplLineEnd)
		}
	}

	return d.Result
}

// logCall выводит имя отладочной функции и место её вызова.
func logCall(fn string) {
	const tplLogCall = "[ %30s ] %s:%d"
	var obj *debug

	if !seeCalls.Load() {
		return
	}
	obj = newDebug()
	_, _ = obj.ReadWriter.WriteString(fmt.Sprintf(tplLogCall, fn, obj.Trace.File, obj.Trace.Line))
	fmt.Print(obj.Final().String())
}

// Nop — пустая функция. Если задано значение переменной окружения DEBUG_CALLS=1, все вызовы Nop() выводят имя файла
// и номер строки вызова.
func Nop() {
	const tplFuncName = "debug.Nop()"

	logCall(tplFuncName)
}

// Dumper выводит дамп всех переменных в STDOUT.
func Dumper(idl ...any) {
	const tplFuncName = "debug.Dumper()"

	fmt.Print(newDebug().Prefix(tplFuncName).Dump(idl...).Suffix().Final().String())
}

// DumperString выводит дамп всех переменных в строку.
func DumperString(idl ...any) string {
	const tplFuncName = "debug.DumperString()"

	return newDebug().Prefix(tplFuncName).Dump(idl...).Suffix().Final().String()
}

// DumperByte выводит дамп всех переменных в []byte.
func DumperByte(idl ...any) []byte {
	const tplFuncName = "debug.DumperByte()"

	return newDebug().Prefix(tplFuncName).Dump(idl...).Suffix().Final().Bytes()
}

// DumperFile выводит дамп всех переменных в файл. Имя файла совпадает с именем программы,
// расширение — .txt.
func DumperFile(idl ...any) {
	const tplFuncName = "debug.DumperFile()"

	_ = FileSave(newDebug().Prefix(tplFuncName).Dump(idl...).Suffix().Final().Bytes(), "", "")
}

// SeeCallStack включает или выключает вывод стека вызовов.
func SeeCallStack(v bool) {
	const tplFuncName = "debug.SeeCallStack()"

	seeTrace.Store(v)
	logCall(tplFuncName)
}
