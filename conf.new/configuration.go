package conf

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type configuration struct {
	raw  []string
	data map[string]string
	line map[string]int
}

func newConfiguration(r io.Reader) *configuration {
	c := &configuration{
		data: make(map[string]string),
		line: make(map[string]int),
	}
	sc := bufio.NewScanner(r)
	var section string

	for sc.Scan() {
		line := sc.Text()
		c.raw = append(c.raw, line)
		line = strings.TrimSpace(line)
		l := len(line)
		// Line is comment or blank line
		if l == 0 || line[0] == '#' || line[0] == ';' {
			continue
		}
		// line is section header
		if line[0] == '[' && line[l-1] == ']' {
			section = line[1 : l-1]
			continue
		}
		if i := strings.Index(line, "="); i > 0 {
			key, value := strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
			key = section + "." + key
			c.line[key] = len(c.raw) - 1
			c.data[key] = value
		}
	}

	return c
}

func (c *configuration) string(key string) string {
	if value, exists := c.data[key]; exists {
		return value
	}
	return ""
}

func (c *configuration) bool(key string) bool {
	if value, err := strconv.ParseBool(c.string(key)); err == nil {
		return value
	}
	return false
}

func (c *configuration) int(key string) int {
	if value, err := strconv.Atoi(c.string(key)); err == nil {
		return value
	}
	return 0
}

func (c *configuration) slice(key string) []string {
	v := c.string(key)
	if len(v) == 0 {
		return []string{}
	}
	value := strings.Split(v, ",")
	for i, e := range value {
		value[i] = strings.TrimSpace(e)
	}
	return value
}

func (c *configuration) fusion(c2 *configuration) (modified bool) {
	for k, v := range c.data {
		if v2, exists := c2.data[k]; !exists {
			modified = true
		} else if v != v2 {
			modified = true
			c.data[k] = v2
			n := c.line[k]
			line := c.raw[n]
			i := strings.Index(line, "=")
			c.raw[n] = line[:i+1] + " " + v2
		}
	}
	if !modified {
		for k := range c2.data {
			if _, exists := c.data[k]; !exists {
				modified = true
				break
			}
		}
	}
	return
}

func (c *configuration) writeTo(w io.Writer) (err error) {
	buf := bufio.NewWriter(w)
	for _, line := range c.raw {
		if _, err = buf.WriteString(line); err != nil {
			break
		}
		if err = buf.WriteByte('\n'); err != nil {
			break
		}
	}
	return
}

func String(key string) string {
	return cnf.string(key)
}

func Bool(key string) bool {
	return cnf.bool(key)
}

func Int(key string) int {
	return cnf.int(key)
}

func Slice(key string) []string {
	return cnf.slice(key)
}
