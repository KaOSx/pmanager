package conv

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Slice2String(s []string) string { return strings.Join(s, ",") }
func Any2String(v any) string        { return fmt.Sprint(v) }

func String2Int(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)

	return i
}

func Bool2Int(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func Date2Int(d time.Time) int64 { return d.Unix() }

func String2Slice(s string) []string {
	out := strings.Split(s, ",")
	for i, e := range out {
		out[i] = strings.TrimSpace(e)
	}

	return out
}

func String2Bool(s string) bool {
	b, _ := strconv.ParseBool(s)

	return b
}

func Int2Bool(i int64) bool { return i != 0 }

func reg(s string) *regexp.Regexp { return regexp.MustCompile(s) }

func match(r, s string) bool { return reg(r).MatchString(s) }

func Int2Date(i int64) time.Time { return time.Unix(i, 0) }

func String2Date(s string) time.Time {
	switch {
	case match(`^\d{4}.\d{2}.\d{2}$`, s):
		y, m, d := String2Int(s[:4]), String2Int(s[5:7]), String2Int(s[8:10])
		return time.Date(int(y), time.Month(m), int(d), 0, 0, 0, 0, time.Local)
	case match(`^\d{4}.\d{2}.\d{2}T\d{2}.\d{2}$`, s):
		y, m, d := String2Int(s[:4]), String2Int(s[5:7]), String2Int(s[8:10])
		h, n := String2Int(s[11:13]), String2Int(s[14:16])
		return time.Date(int(y), time.Month(m), int(d), int(h), int(n), 0, 0, time.Local)
	case match(`^\d{4}.\d{2}.\d{2}T\d{2}.\d{2}.\d{2}$`, s):
		y, m, d := String2Int(s[:4]), String2Int(s[5:7]), String2Int(s[8:10])
		h, n, ss := String2Int(s[11:13]), String2Int(s[14:16]), String2Int(s[17:19])
		return time.Date(int(y), time.Month(m), int(d), int(h), int(n), int(ss), 0, time.Local)
	case match(`^\d+$`, s):
		return Int2Date(String2Int(s))
	}

	d, _ := time.Parse(time.RFC3339, s)
	return d
}

type Map map[string]any

func (m Map) Exists(key string) bool {
	_, ok := m[key]

	return ok
}

func (m Map) GetString(key string) (e string) {
	if v, ok := m[key]; ok {
		switch v.(type) {
		case string:
			e = v.(string)
		case []string:
			e = Slice2String(v.([]string))
		default:
			e = Any2String(v)
		}
	}

	return
}

func (m Map) GetInt(key string) (e int64) {
	if v, ok := m[key]; ok {
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			e = rv.Int()
		case reflect.Float32, reflect.Float64:
			e = int64(rv.Float())
		case reflect.String:
			e = String2Int(rv.String())
		case reflect.Bool:
			e = Bool2Int(rv.Bool())
		}
	}

	return
}

func (m Map) GetBool(key string) (e bool) {
	if v, ok := m[key]; ok {
		switch v.(type) {
		case bool:
			e = v.(bool)
		case int:
			e = Int2Bool(int64(v.(int)))
		case int64:
			e = Int2Bool(v.(int64))
		case string:
			e = String2Bool(v.(string))
		}
	}

	return
}

func (m Map) GetSlice(key string) (e []string) {
	if v, ok := m[key]; ok {
		switch v.(type) {
		case []string:
			e = v.([]string)
		case string:
			e = String2Slice(v.(string))
		}
	}

	return
}

func (m Map) GetDate(key string) (e time.Time) {
	if v, ok := m[key]; ok {
		switch v.(type) {
		case time.Time:
			e = v.(time.Time)
		case int:
			e = Int2Date(int64(v.(int)))
		case int64:
			e = Int2Date(v.(int64))
		case string:
			e = String2Date(v.(string))
		}
	}

	return
}

func (m Map) Delete(keys ...string) Map {
	for _, k := range keys {
		delete(m, k)
	}

	return m
}
