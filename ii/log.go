package ii

import (
	"io"
	"log"
	"os"
)

var (
	Trace *log.Logger
	Info  *log.Logger
	Error *log.Logger
)
func InitLog() {
	Trace = log.New(os.Stdout, "=== ", log.Ldate|log.Ltime)
	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	Error = log.New(os.Stderr, "ERR: ", log.Ldate|log.Ltime)
}
func OpenLog(trace io.Writer, info io.Writer, error io.Writer) {
	Trace = log.New(trace, "=== ", log.Ldate|log.Ltime)
	Info = log.New(info, "INFO: ", log.Ldate|log.Ltime)
	Error = log.New(error, "ERR: ", log.Ldate|log.Ltime)
}
