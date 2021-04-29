package log

import (
	"fmt"
	"io"
	"log"
	"os"
)

// LPrintf calls l.Output to print to the logger.
// calldepth assign to 3 to get the real calling function
func LPrintf(l *log.Logger, format string, v ...interface{}) {
	l.Output(3, fmt.Sprintf(format, v...))
}

// log modules
const (
	ModuleGoVtep string = "[GoVtep]"
	ModuleTAI    string = "[TAI]"
	ModuleDriver string = "[Driver]"
)

// log variables
var (
	logFile, _ = os.OpenFile("/var/log/controller_vtep.log", os.
			O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	InfoLogger *log.Logger = log.
			New(io.MultiWriter(logFile), "[INFO]", log.Ltime|log.Lshortfile)
	WarningLogger *log.Logger = log.
			New(io.MultiWriter(logFile, os.Stderr), "[WARN]", log.Ltime|log.Lshortfile)
	ErrorLogger *log.Logger = log.
			New(io.MultiWriter(logFile, os.Stderr), "[ERROR]", log.Ltime|log.Lshortfile)
)

// Info func
func Info(format string, v ...interface{}) {
	LPrintf(InfoLogger, format, v...)
}

// Warning func
func Warning(format string, v ...interface{}) {
	LPrintf(WarningLogger, format, v...)
}

// Error func
func Error(format string, v ...interface{}) {
	LPrintf(ErrorLogger, format, v...)
}
