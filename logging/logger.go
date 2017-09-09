package logging

import (
	"io"
	"log"
)

type Logger interface {
	Debug(string, ...interface{})
	Warn(string, ...interface{})
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

func (l logger) Debug(msg string, args ...interface{}) {
	if l.debug != nil {
		l.debug.Printf(msg, args...)
	}
}

func (l logger) Warn(msg string, args ...interface{}) {
	if l.warn != nil {
		l.warn.Printf(msg, args...)
	}
}
