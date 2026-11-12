package debug

import (
	"fmt"
	"go/token"
	"io"
	"reflect"
	"sort"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// isExported проверяет, что имя начинается с заглавной буквы, то есть является экспортированным.
func isExported(name string) bool {
	var ch rune

	ch, _ = utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

// notNilFilter пропускает только ненулевые ссылочные значения.
//
//goland:noinspection ALL
func notNilFilter(name string, value reflect.Value) (ret bool) {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		ret = !value.IsNil()
	default:
		ret = true
	}

	return
}

// printerPrint выводит значение x в поток w через внутренний принтер.
func printerPrint(w io.Writer, fset *token.FileSet, x any, f printerFieldFilter) (err error) {
	var p printer

	p = printer{
		output:        w,
		fset:          fset,
		filter:        f,
		lineNumber:    make(map[visit]int),
		lastByteWrite: '\n',
	}
	// Ошибка записи пробрасывается как паника localError и преобразуется обратно здесь.
	defer func() {
		if e := recover(); e != nil {
			if le, ok := e.(localError); ok {
				err = le.err
				return
			}
			// Неизвестная паника: пробрасываем дальше, маскировать её нельзя.
			panic(e)
		}
	}()
	// nil выводится отдельной строкой.
	if x == nil {
		p.printf(tplNil + tplLineEnd)
		return
	}
	p.print(reflect.ValueOf(x))
	p.printf(tplLineEnd)

	return
}

// printf выводит форматированную строку; ошибка записи преобразуется в панику localError.
func (p *printer) printf(format string, args ...any) {
	if _, err := fmt.Fprintf(p, format, args...); err != nil {
		panic(localError{err: err})
	}
}

// Write реализует io.Writer: добавляет номера строк и отступы после переводов строк.
func (p *printer) Write(data []byte) (n int, err error) {
	const tplLineNumber string = "%06d: "
	var (
		i, j, m int
		b       byte
	)

	for i, b = range data {
		if b == '\n' {
			// Перевод строки завершает текущий фрагмент.
			m, err = p.output.Write(data[n : i+1])
			n += m
			if err != nil {
				return
			}
			p.currentLineNumber++
		} else if p.lastByteWrite == '\n' {
			// Начало строки: номер строки и отступы.
			if _, err = fmt.Fprintf(p.output, tplLineNumber, p.currentLineNumber); err != nil {
				return
			}
			for j = p.indentLevel; j > 0; j-- {
				if _, err = p.output.Write(indent); err != nil {
					return
				}
			}
		}
		p.lastByteWrite = b
	}
	// Хвост без перевода строки.
	if len(data) > n {
		m, err = p.output.Write(data[n:])
		n += m
	}

	return
}

// seen регистрирует объект и сообщает, выводился ли он ранее в рамках этого дампа.
func (p *printer) seen(x reflect.Value) (line int, exists bool) {
	var key visit

	key = visit{typ: x.Type(), ptr: unsafe.Pointer(x.Pointer())}
	if line, exists = p.lineNumber[key]; !exists {
		p.lineNumber[key] = p.currentLineNumber
	}

	return
}

// LenPlus выводит элементы массива или среза с индексами при непустом значении.
func (p *printer) LenPlus(x reflect.Value) {
	const tplItem string = "%d: "
	var i, n int

	if x.Len() > 0 {
		p.indentLevel++
		p.printf(tplLineEnd)
		for i, n = 0, x.Len(); i < n; i++ {
			p.printf(tplItem, i)
			p.print(x.Index(i))
			p.printf(tplLineEnd)
		}
		p.indentLevel--
	}
}

// printStringer выводит результат метода String(), если он имеет сигнатуру func() string.
func (p *printer) printStringer(x reflect.Value) (ok bool) {
	const tplResult string = "%q\n"
	var (
		a reflect.Value
		b func() string
	)

	if a = x.MethodByName("String"); !a.IsValid() {
		return
	}
	if b, ok = a.Interface().(func() string); !ok {
		return
	}
	p.printf(tplResult, b())
	ok = true

	return
}

// printerAsError возвращает значение как error, если оно (или указатель на него) реализует интерфейс error.
func printerAsError(x reflect.Value) (err error, ok bool) {
	switch t := x.Type(); {
	case t.Implements(errorInterfaceType):
		if x.CanInterface() {
			err, ok = x.Interface().(error)
		}
	case reflect.PointerTo(t).Implements(errorInterfaceType):
		if x.CanAddr() && x.Addr().CanInterface() {
			err, ok = x.Addr().Interface().(error)
		}
	}

	return
}

// printStruct выводит значение структуры: ошибки — строковым представлением,
// структуры с методом String() — через него, остальные — по экспортированным полям.
func (p *printer) printStruct(x reflect.Value) {
	const (
		tplFieldLine  string = "%s: "
		tplErrorValue string = "error %q\n"
	)
	var (
		t         reflect.Type
		value     reflect.Value
		ok, first bool
		i, n      int
		name      string
		err       error
		isErr     bool
	)

	t = x.Type()
	// Ошибки выводятся их строковым представлением.
	if err, isErr = printerAsError(x); isErr {
		p.printf(tplErrorValue, err.Error())
		return
	}
	p.printf(tplStructOpen, t)
	p.indentLevel++
	defer func() {
		p.indentLevel--
		p.printf(tplStructClose)
	}()
	// Структура с методом String() выводится через него.
	if _, ok = t.MethodByName("String"); ok {
		p.printf(tplLineEnd)
		if !p.printStringer(x) {
			p.printf(tplBadStringer)
		}
		return
	}
	// Вывод экспортированных полей, прошедших фильтр.
	first = true
	for i, n = 0, t.NumField(); i < n; i++ {
		name = t.Field(i).Name
		if !isExported(name) {
			continue
		}
		value = x.Field(i)
		if p.filter != nil && !p.filter(name, value) {
			continue
		}
		if first {
			p.printf(tplLineEnd)
			first = false
		}
		p.printf(tplFieldLine, name)
		if value.MethodByName("String").IsValid() {
			p.printStruct(value)
		} else {
			p.print(value)
		}
		p.printf(tplLineEnd)
	}
}

// print выводит значение в зависимости от его вида (kind).
func (p *printer) print(x reflect.Value) {
	var (
		v    any
		s    []byte
		ok   bool
		line int
		keys []reflect.Value
		key  reflect.Value
	)

	if !notNilFilter("", x) {
		p.printf(tplNil)
		return
	}
	//goland:noinspection GoFixInline
	switch x.Kind() {
	case reflect.Interface:
		p.print(x.Elem())
	case reflect.Map:
		// Защита от циклических ссылок: карта, содержащая саму себя.
		if x.Len() > 0 {
			if line, ok = p.seen(x); ok {
				p.printf(tplObjectRef, x.Type(), line)
				return
			}
		}
		p.printf(tplMapOpen, x.Type(), x.Len())
		if x.Len() > 0 {
			p.indentLevel++
			p.printf(tplLineEnd)
			keys = x.MapKeys()
			// Сортировка ключей делает вывод воспроизводимым между запусками.
			sort.Slice(keys, func(i, j int) bool {
				return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
			})
			for _, key = range keys {
				p.print(key)
				p.printf(tplColon)
				p.print(x.MapIndex(key))
				p.printf(tplLineEnd)
			}
			p.indentLevel--
		}
		p.printf(tplStructClose)
	case reflect.Ptr:
		p.printf(tplPointer)
		// Защита от циклических ссылок: указатель на уже выводившийся объект.
		if line, ok = p.seen(x); ok {
			p.printf(tplObjectRefShort, line)
		} else {
			p.print(x.Elem())
		}
	case reflect.Array:
		p.printf(tplStructOpen, x.Type())
		if _, ok = x.Type().MethodByName("String"); ok {
			p.indentLevel++
			p.printf(tplLineEnd)
			if !p.printStringer(x) {
				p.printf(tplBadStringer)
			}
			p.indentLevel--
		} else {
			p.LenPlus(x)
		}
		p.printf(tplStructClose)
	case reflect.Slice:
		// Срез байт выводится шестнадцатеричным литералом.
		if s, ok = x.Interface().([]byte); ok {
			p.printf(tplHexQuoted, s)
			return
		}
		// Защита от циклических ссылок: срез, содержащий сам себя.
		if x.Len() > 0 {
			if line, ok = p.seen(x); ok {
				p.printf(tplObjectRef, x.Type(), line)
				return
			}
		}
		p.printf(tplSliceOpen, x.Type(), x.Len(), x.Cap())
		if _, ok = x.Type().MethodByName("String"); ok {
			p.indentLevel++
			p.printf(tplLineEnd)
			if !p.printStringer(x) {
				p.printf(tplBadStringer)
			}
			p.indentLevel--
		} else {
			p.LenPlus(x)
		}
		p.printf(tplStructClose)
	case reflect.Struct:
		p.printStruct(x)
	default:
		v = x.Interface()
		switch w := v.(type) {
		case string:
			p.printf(tplQuoted, w)
			return
		case token.Pos:
			if p.fset != nil {
				p.printf(tplPos, p.fset.Position(w))
				return
			}
		}
		p.printf(tplValue, v)
	}
}
