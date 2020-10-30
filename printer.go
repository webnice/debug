package debug

import (
	"fmt"
	"go/token"
	"io"
	"reflect"
	"unicode"
	"unicode/utf8"
)

type (
	printerFieldFilter func(name string, value reflect.Value) bool
	printer            struct {
		output            io.Writer
		fset              *token.FileSet
		filter            printerFieldFilter
		lineNumber        map[interface{}]int
		indentLevel       int
		lastByteWrite     byte
		currentLineNumber int
	}
	localError struct {
		err error
	}
)

var (
	indent = []byte("\t")
)

func isExported(name string) bool {
	var ch, _ = utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func notNilFilter(name string, value reflect.Value) (ret bool) {
	ret = true
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		ret = !value.IsNil()
	}

	return
}

func printerPrint(w io.Writer, fset *token.FileSet, x interface{}, f printerFieldFilter) (err error) {
	var p = printer{
		output:        w,
		fset:          fset,
		filter:        f,
		lineNumber:    make(map[interface{}]int),
		lastByteWrite: '\n',
	}
	defer func() {
		if e := recover(); e != nil {
			err = e.(localError).err
		}
	}()
	if x == nil {
		p.printf("nil\n")
		return
	}
	p.print(reflect.ValueOf(x))
	p.printf("\n")

	return
}

func (p *printer) printf(format string, args ...interface{}) {
	if _, err := fmt.Fprintf(p, format, args...); err != nil {
		panic("err")
	}
}

func (p *printer) Write(data []byte) (n int, err error) {
	var m int

	for i, b := range data {
		if b == '\n' {
			m, err = p.output.Write(data[n : i+1])
			n += m
			if err != nil {
				return
			}
			p.currentLineNumber++
		} else if p.lastByteWrite == '\n' {
			_, err = fmt.Fprintf(p.output, "%06d: ", p.currentLineNumber)
			if err != nil {
				return
			}
			for j := p.indentLevel; j > 0; j-- {
				_, err = p.output.Write(indent)
				if err != nil {
					return
				}
			}
		}
		p.lastByteWrite = b
	}
	if len(data) > n {
		m, err = p.output.Write(data[n:])
		n += m
	}

	return
}

func (p *printer) LenPlus(x reflect.Value) {
	if x.Len() > 0 {
		p.indentLevel++
		p.printf("\n")
		for i, n := 0, x.Len(); i < n; i++ {
			p.printf("%d: ", i)
			p.print(x.Index(i))
			p.printf("\n")
		}
		p.indentLevel--
	}
}

func (p *printer) printMethodString(x reflect.Value) {
	var (
		a reflect.Value
		b func() string
	)

	p.printf("\n")
	a = x.MethodByName("String")
	b = a.Interface().(func() string)
	p.printf("%q\n", b())
}

func (p *printer) printStruct(x reflect.Value) {
	var (
		t         reflect.Type
		value     reflect.Value
		ok, first bool
		i, n      int
	)

	t = x.Type()
	if t.Name() == "errorString" {
		p.printf("error %q\n", x.Interface())
		return
	}

	p.printf("%s {", t)
	p.indentLevel++
	defer func() {
		p.indentLevel--
		p.printf("}")
	}()
	if _, ok = t.MethodByName("String"); ok {
		p.printMethodString(x)
		return
	}
	first = true
	for i, n = 0, t.NumField(); i < n; i++ {
		if name := t.Field(i).Name; isExported(name) {
			value = x.Field(i)
			if p.filter == nil || p.filter(name, value) {
				if first {
					p.printf("\n")
					first = false
				}
				p.printf("%s: ", name)
				if value.MethodByName("String").IsValid() {
					p.printStruct(value)
				} else {
					p.print(value)
				}
				p.printf("\n")
			}
		}
	}
}

func (p *printer) print(x reflect.Value) {
	if !notNilFilter("", x) {
		p.printf("nil")
		return
	}
	switch x.Kind() {
	case reflect.Interface:
		p.print(x.Elem())
	case reflect.Map:
		p.printf("%s (len = %d) {", x.Type(), x.Len())
		if x.Len() > 0 {
			p.indentLevel++
			p.printf("\n")
			for _, key := range x.MapKeys() {
				p.print(key)
				p.printf(": ")
				p.print(x.MapIndex(key))
				p.printf("\n")
			}
			p.indentLevel--
		}
		p.printf("}")
	case reflect.Ptr:
		p.printf("*")
		ptr := x.Interface()
		if line, exists := p.lineNumber[ptr]; exists {
			p.printf("(obj @ %d)", line)
		} else {
			p.lineNumber[ptr] = p.currentLineNumber
			p.print(x.Elem())
		}
	case reflect.Array:
		p.printf("%s {", x.Type())
		_, ok := x.Type().MethodByName("String")
		if ok {
			p.indentLevel++
			p.printf("\n")
			a := x.MethodByName("String")
			b := a.Interface().(func() string)
			p.printf("%q\n", b())
			p.indentLevel--
		} else {
			p.LenPlus(x)
		}
		p.printf("}")
	case reflect.Slice:
		if s, ok := x.Interface().([]byte); ok {
			p.printf("%#q", s)
			return
		}
		p.printf("%s (len = %d) {", x.Type(), x.Len())
		if _, ok := x.Type().MethodByName("String"); ok {
			p.indentLevel++
			p.printf("\n")
			a := x.MethodByName("String")
			b := a.Interface().(func() string)
			p.printf("%q\n", b())
			p.indentLevel--
		} else {
			p.LenPlus(x)
		}
		p.printf("}")
	case reflect.Struct:
		p.printStruct(x)
	default:
		v := x.Interface()
		switch v := v.(type) {
		case string:
			p.printf("%q", v)
			return
		case token.Pos:
			if p.fset != nil {
				p.printf("%s", p.fset.Position(v))
				return
			}
		}
		p.printf("%v", v)
	}
}
