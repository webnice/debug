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

const (
	delimiterBeg string = `[ %02d.%02d.%04d %02d:%02d:%02d:%010d ] ------------------------------------------------------------`
	delimiterEnd string = `-----------------------------------------------------------------------------------------------`
	lineEnd      string = "\n"
	lineEndCRLF  string = "\r\n"
)

var (
	defaultCRLF bool // For operating systems with the end of line CRLF or set environment value DEBUG_CRLF=true or DEBUG_CRLF=false
	seeCalls    bool // Search on a large project forgotten debug calls. Set to true by environment value DEBUG_CALLS=true
	seeTrace    bool // Printing with the bump data call stack. Set to true by set environment value DEBUG_TRACESTACK=true
)

type debug struct {
	Result     *bytes.Buffer
	Buffer     *bytes.Buffer
	ReadWriter *bufio.ReadWriter
	UseCRLF    bool
	Now        time.Time
	Trace      *trace
}

func init() {
	switch runtime.GOOS {
	case "windows":
		defaultCRLF = true
	}
	if os.Getenv("DEBUG_CALLS") != "" {
		if strings.EqualFold(os.Getenv("DEBUG_CALLS"), "false") != true {
			seeCalls = true
		}
	}
	if os.Getenv("DEBUG_CRLF") != "" {
		if strings.EqualFold(os.Getenv("DEBUG_CRLF"), "false") != true {
			defaultCRLF = true
		}
	}
	if os.Getenv("DEBUG_TRACESTACK") != "" {
		if strings.EqualFold(os.Getenv("DEBUG_TRACESTACK"), "false") != true {
			seeTrace = true
		}
	}
}

func newDebug() (self *debug) {
	self = new(debug)
	self.UseCRLF = defaultCRLF
	self.Result = bytes.NewBuffer([]byte{})
	self.Buffer = bytes.NewBuffer([]byte{})
	self.ReadWriter = bufio.NewReadWriter(bufio.NewReader(self.Buffer), bufio.NewWriter(self.Buffer))
	self.Now = time.Now().In(time.Local)
	self.Trace = newTrace().Trace(traceStepBack + 1)
	return
}

// Dump all variables
func (d *debug) Dump(idl ...interface{}) *debug {
	var i int
	var fset *token.FileSet

	for i = range idl {
		fset = token.NewFileSet()
		_ = printerPrint(d.ReadWriter, fset, idl[i], notNilFilter)
	}
	// call stack
	if seeTrace {
		_, _ = d.ReadWriter.WriteString(d.Trace.Stack + lineEnd)
	}
	return d
}

// Add information before dump
func (d *debug) Prefix(fn string) *debug {
	_, _ = d.ReadWriter.WriteString(fmt.Sprintf(delimiterBeg+lineEnd, d.Now.Day(), d.Now.Month(), d.Now.Year(), d.Now.Hour(), d.Now.Minute(), d.Now.Second(), d.Now.Nanosecond()))
	_, _ = d.ReadWriter.WriteString(fmt.Sprintf("[ %30s ] %s:%d [%s] [%s()]"+lineEnd, fn, d.Trace.File, d.Trace.Line, d.Trace.Package, d.Trace.Function))
	return d
}

// Add information after dump
func (d *debug) Suffix() *debug {
	_, _ = d.ReadWriter.WriteString(delimiterEnd + lineEnd)
	return d
}

// Finalisation dump
func (d *debug) Final() *bytes.Buffer {
	var line []byte
	var isPrefix bool
	var err error

	_ = d.ReadWriter.Flush()
	for {
		line, isPrefix, err = d.ReadWriter.ReadLine()
		_, _ = d.Result.Write(line)
		if isPrefix {
			continue
		}
		if len(line) > 0 {
			if d.UseCRLF {
				_, _ = d.Result.WriteString(lineEndCRLF)
			} else {
				_, _ = d.Result.WriteString(lineEnd)
			}
		}
		if err != nil {
			break
		}
	}
	return d.Result
}
