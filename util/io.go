package util

import (
	"encoding/json"
	"io"
	"log"
	"log/syslog"
	"os"
	"pmanager/conf"
)

func ReadJSON(r io.Reader, v interface{}) error {
	d := json.NewDecoder(r)
	return d.Decode(v)
}

func WriteJSON(w io.Writer, v interface{}) error {
	e := json.NewEncoder(w)
	return e.Encode(v)
}

func JSON(v interface{}) []byte {
	var b []byte
	if conf.Debug() {
		b, _ = json.MarshalIndent(v, "", "  ")
	} else {
		b, _ = json.Marshal(v)
	}
	return b
}

func ToMap(v interface{}) conf.Map {
	m := make(conf.Map)
	b, _ := json.Marshal(v)
	json.Unmarshal(b, &m)
	return m
}

func Logger() *log.Logger {
	if !conf.Debug() {
		l, err := syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_WARNING|syslog.LOG_NOTICE, log.LstdFlags)
		if err == nil {
			return l
		}
	}
	return log.New(os.Stderr, "", log.LstdFlags)
}

func Println(v ...interface{}) {
	Logger().Println(v...)
}

func Printf(f string, v ...interface{}) {
	Logger().Printf(f, v...)
}

func Debugln(v ...interface{}) {
	if conf.Debug() {
		Println(v...)
	}
}

func Debugf(f string, v ...interface{}) {
	if conf.Debug() {
		Printf(f, v...)
	}
}

func Fatalln(v ...interface{}) {
	Logger().Fatalln(v...)
}

func Fatalf(f string, v ...interface{}) {
	Logger().Fatalf(f, v...)
}
