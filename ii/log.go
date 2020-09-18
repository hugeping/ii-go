// Simple log system.
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

// Default mode. All messages are shown.
func InitLog() {
	Trace = log.New(os.Stdout, "=== ", log.Ldate|log.Ltime)
	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	Error = log.New(os.Stderr, "ERR: ", log.Ldate|log.Ltime)
}

// Custom mode. Use io.Writers to select what verbose level is needed.
// For example: ii.OpenLog(ioutil.Discard, os.Stdout, os.Stderr)
func OpenLog(trace io.Writer, info io.Writer, error io.Writer) {
	Trace = log.New(trace, "=== ", log.Ldate|log.Ltime)
	Info = log.New(info, "INFO: ", log.Ldate|log.Ltime)
	Error = log.New(error, "ERR: ", log.Ldate|log.Ltime)
}
