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
func (self *debug) Dump(idl ...interface{}) *debug {
	var i int
	var fset *token.FileSet

	for i = range idl {
		fset = token.NewFileSet()
		printerPrint(self.ReadWriter, fset, idl[i], notNilFilter)
	}
	// call stack
	if seeTrace {
		self.ReadWriter.WriteString(self.Trace.Stack + lineEnd)
	}
	return self
}

// Add information before dump
func (self *debug) Prefix(fn string) *debug {
	self.ReadWriter.WriteString(fmt.Sprintf(delimiterBeg+lineEnd, self.Now.Day(), self.Now.Month(), self.Now.Year(), self.Now.Hour(), self.Now.Minute(), self.Now.Second(), self.Now.Nanosecond()))
	self.ReadWriter.WriteString(fmt.Sprintf("[ %30s ] %s:%d [%s] [%s()]"+lineEnd, fn, self.Trace.File, self.Trace.Line, self.Trace.Package, self.Trace.Function))
	return self
}

// Add information after dump
func (self *debug) Suffix() *debug {
	self.ReadWriter.WriteString(delimiterEnd + lineEnd)
	return self
}

// Finalisation dump
func (self *debug) Final() *bytes.Buffer {
	var line []byte
	var isPrefix bool
	var err error

	self.ReadWriter.Flush()
	for {
		line, isPrefix, err = self.ReadWriter.ReadLine()
		self.Result.Write(line)
		if isPrefix {
			continue
		}
		if len(line) > 0 {
			if self.UseCRLF {
				self.Result.WriteString(lineEndCRLF)
			} else {
				self.Result.WriteString(lineEnd)
			}
		}
		if err != nil {
			break
		}
	}
	return self.Result
}
