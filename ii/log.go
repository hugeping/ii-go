package ii

import (
	"log"
	"io"
)

var (
	Trace   *log.Logger
	Info    *log.Logger
	Error   *log.Logger
)

func OpenLog(trace io.Writer, info io.Writer, error io.Writer) {
	Trace = log.New(trace, "=== ", log.Ldate|log.Ltime)
	Info = log.New(info, "INFO: ", log.Ldate|log.Ltime)
	Error = log.New(error, "ERR: ", log.Ldate|log.Ltime)
}
