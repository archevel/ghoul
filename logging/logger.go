package logging

import (
	"io"
	"log"
	"os"

	e "github.com/archevel/ghoul/expressions"
)

var (
	NoLogger       = New(nil, nil)
	StandardLogger = New(nil, os.Stderr)
	VerboseLogger  = New(os.Stderr, os.Stderr)
)

type Logger interface {
	Debug(string, ...e.Expr)
	Warn(string, ...e.Expr)
}

type logger struct {
	debug *log.Logger
	warn  *log.Logger
}

func New(debug io.Writer, warn io.Writer) Logger {

	var debugLogger *log.Logger = nil
	var warnLogger *log.Logger = nil

	if debug != nil {
		debugLogger = log.New(debug, "DEBUG ", log.Lmicroseconds|log.LUTC)
	}
	if warn != nil {
		warnLogger = log.New(warn, "WARN", log.Lmicroseconds|log.LUTC)
	}
	return &logger{debugLogger, warnLogger}

}

func (l logger) Debug(msg string, args ...e.Expr) {
	if l.debug != nil {
		writeToLog(l.debug, msg, args)
	}
}

func (l logger) Warn(msg string, args ...e.Expr) {
	if l.warn != nil {
		writeToLog(l.warn, msg, args)
	}
}

func writeToLog(l *log.Logger, msg string, args []e.Expr) {
	asReprStrings := make([]interface{}, len(args))
	for i, v := range args {
		asReprStrings[i] = v.Repr()
	}

	l.Printf(msg, asReprStrings...)
}
