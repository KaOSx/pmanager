package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"
)

const (
	cInfo    = "Info"
	cWarning = "Warning"
	cError   = "Error"
	cFatal   = "Fatal"
)

var (
	logsingleton *logger
	Debug        bool
)

type wc struct {
	io.Writer
}

func (w wc) Close() error {
	return nil
}

type logger struct {
	logpath string
	w       io.WriteCloser
	*log.Logger
	sync.Mutex
}

func (l *logger) openWriter() {
	switch l.logpath {
	case "stderr", "":
		l.w = wc{os.Stderr}
	case "stdout":
		l.w = wc{os.Stdout}
	default:
		var err error
		if l.w, err = os.OpenFile(l.logpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
			l.w = wc{os.Stderr}
		}
	}
}

func (l *logger) open(prefix string) {
	l.Lock()
	l.openWriter()
	l.Logger = log.New(l.w, fmt.Sprintf("[%s]: ", prefix), log.LstdFlags|log.Lmsgprefix)
}

func (l *logger) close() {
	l.w.Close()
	l.Logger = nil
	l.Unlock()
}

func Init(logpath string) {
	logsingleton = &logger{
		logpath: logpath,
	}

	logdir := path.Dir(logpath)
	if err := os.MkdirAll(logdir, 0755); err != nil {
		logsingleton.logpath = "stdout"
	}
}

func logf(prefix, tmpl string, args ...any) {
	if prefix == cError {
		tmpl = "\033[1;31m" + tmpl + "\033[m"
	}

	logsingleton.open(prefix)
	defer logsingleton.close()

	logsingleton.Printf(tmpl, args...)
}

func logln(prefix string, args ...any) {
	if prefix == cError {
		strargs := make([]string, len(args))
		for i, e := range args {
			strargs[i] = fmt.Sprint(e)
		}
		logf(prefix, "%s", strings.Join(strargs, " "))
		return
	}

	logsingleton.open(prefix)
	defer logsingleton.close()

	logsingleton.Println(args...)
}

func Printf(tmpl string, args ...any) {
	logf(cInfo, tmpl, args...)
}

func Println(args ...any) {
	logln(cInfo, args...)
}

func Warnf(tmpl string, args ...any) {
	logf(cWarning, tmpl, args...)
}

func Warnln(args ...any) {
	logln(cWarning, args...)
}

func Errorf(tmpl string, args ...any) {
	logf(cError, tmpl, args...)
}

func Errorln(args ...any) {
	logln(cError, args...)
}

func Fatalf(tmpl string, args ...any) {
	logsingleton.open(cFatal)
	defer logsingleton.close()

	logsingleton.Fatalf(tmpl, args...)
}

func Fatalln(args ...any) {
	logsingleton.open(cFatal)
	defer logsingleton.close()

	logsingleton.Fatalln(args...)
}

func Debugln(v ...any) {
	if Debug {
		Println(v...)
	}
}

func Debugf(f string, v ...any) {
	if Debug {
		Printf(f, v...)
	}
}

func Copy(r io.Reader) {
	var buf strings.Builder

	io.Copy(&buf, r)
	Println(buf.String())
}
