package debug

import (
	"fmt"
)

// Nop Empty function. if set environment value DEBUG_CALLS=1 all calls Nop() displayed filename and line
func Nop() {
	if seeCalls {
		var self = newDebug()
		_, _ = self.ReadWriter.WriteString(fmt.Sprintf("[ %30s ] %s:%d", "debug.Nop()", self.Trace.File, self.Trace.Line))
		fmt.Print(self.Final().String())
	}
}

// Dumper Dump all variables to STDOUT
func Dumper(idl ...interface{}) {
	fmt.Print(newDebug().Prefix("debug.Dumper()").Dump(idl...).Suffix().Final().String())
}

// DumperString Dump all variables to string
func DumperString(idl ...interface{}) string {
	return newDebug().Prefix("debug.DumperString()").Dump(idl...).Suffix().Final().String()
}

// DumperByte Dump all variables to []byte
func DumperByte(idl ...interface{}) []byte {
	return newDebug().Prefix("debug.DumperByte()").Dump(idl...).Suffix().Final().Bytes()
}

// DumperFile Dump all variables to file. Name file is the same program name and .txt at extension
func DumperFile(idl ...interface{}) {
	_ = FileSave(newDebug().Prefix("debug.DumperFile()").Dump(idl...).Suffix().Final().Bytes(), "", "")
}

// SeeCallStack Enable or disable call stack trace
func SeeCallStack(v bool) {
	seeTrace = v
	if seeCalls {
		var self = newDebug()
		_, _ = self.ReadWriter.WriteString(fmt.Sprintf("[ %30s ] %s:%d", "debug.SeeCallStack()", self.Trace.File, self.Trace.Line))
		fmt.Print(self.Final().String())
	}
}
