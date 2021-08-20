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

func logf(prefix, tmpl string, args ...interface{}) {
	if prefix == cError {
		tmpl = "\033[1;31m" + tmpl + "\033[m"
	}
	logsingleton.open(prefix)
	defer logsingleton.close()
	logsingleton.Printf(tmpl, args...)
}

func logln(prefix string, args ...interface{}) {
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

func Printf(tmpl string, args ...interface{}) {
	logf(cInfo, tmpl, args...)
}

func Println(args ...interface{}) {
	logln(cInfo, args...)
}

func Warnf(tmpl string, args ...interface{}) {
	logf(cWarning, tmpl, args...)
}

func Warnln(args ...interface{}) {
	logln(cWarning, args...)
}

func Errorf(tmpl string, args ...interface{}) {
	logf(cError, tmpl, args...)
}

func Errorln(args ...interface{}) {
	logln(cError, args...)
}

func Fatalf(tmpl string, args ...interface{}) {
	logsingleton.open(cFatal)
	defer logsingleton.close()
	logsingleton.Fatalf(tmpl, args...)
}

func Fatalln(args ...interface{}) {
	logsingleton.open(cFatal)
	defer logsingleton.close()
	logsingleton.Fatalln(args...)
}

func Debugln(v ...interface{}) {
	if Debug {
		Println(v...)
	}
}

func Debugf(f string, v ...interface{}) {
	if Debug {
		Printf(f, v...)
	}
}

func Copy(r io.Reader) {
	var buf strings.Builder
	io.Copy(&buf, r)
	Println(buf.String())
}
