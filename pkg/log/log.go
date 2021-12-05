package log

import (
	"log"
	"os"
)

type Logger struct {
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
}

func NewLogger() *Logger {
	l := Logger{}

	l.Info = log.New(os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	l.Warning = log.New(os.Stdout,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	l.Error = log.New(os.Stderr,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	return &l
}
