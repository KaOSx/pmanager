package database

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"
)

var (
	dbsingleton *database
)

type database struct {
	sync.Mutex
	*gorm.DB
}

func newDb(connector gorm.Dialector, tables ...interface{}) (dbl *database, err error) {
	var db *gorm.DB
	if db, err = gorm.Open(connector, &gorm.Config{}); err != nil {
		return
	}
	dbl = &database{DB: db}
	err = dbl.AutoMigrate(tables...)
	return
}

type SqlSlice []string

func (sl *SqlSlice) Scan(v interface{}) error {
	var bytes []byte
	switch v.(type) {
	case []byte:
		bytes = v.([]byte)
	case string:
		bytes = []byte(v.(string))
	default:
		return fmt.Errorf("%v is not convertible to database.SqlSlice", v)
	}
	return json.Unmarshal(bytes, sl)
}

func (SqlSlice) GormDataType() string {
	return "blob"
}

func (sl SqlSlice) Value() (driver.Value, error) {
	var v strings.Builder
	enc := json.NewEncoder(&v)
	if err := enc.Encode(sl); err != nil {
		return nil, err
	}
	return v.String(), nil
}

func (sl1 SqlSlice) Equal(sl2 SqlSlice) bool {
	if len(sl1) != len(sl2) {
		return false
	}
	for i, e := range sl1 {
		if sl2[i] != e {
			return false
		}
	}
	return true
}

type Filter struct {
	f string
	o string
	v interface{}
}

func (f Filter) String() string {
	return fmt.Sprintf("%s %s ?", f.f, f.o)
}

func NewFilter(field, operation string, value interface{}) Filter {
	return Filter{
		f: field,
		o: operation,
		v: value,
	}
}

type Sort struct {
	f string
	d bool
}

func NewSort(field string, desc bool) Sort {
	return Sort{
		f: field,
		d: desc,
	}
}

func (s Sort) String() string {
	dir := "ASC"
	if s.d {
		dir = "DESC"
	}
	return fmt.Sprintf("%s %s", s.f, dir)
}

type Request struct {
	f []Filter
	s []Sort
	l int64
	o int64
}

type Pagination struct {
	Total   int64
	Limit   int64
	Offset  int64
	Current int64
	Last    int64
}

func (r *Request) where() func(*gorm.DB) *gorm.DB {
	return func(sc *gorm.DB) *gorm.DB {
		l := len(r.f)
		if l == 0 {
			return sc
		}
		filters, values := make([]string, l), make([]interface{}, l)
		for i, ff := range r.f {
			filters[i] = ff.String()
			values[i] = ff.v
		}
		filter := strings.Join(filters, " AND ")
		return sc.Where(filter, values...)
	}
}

func (r *Request) order() func(*gorm.DB) *gorm.DB {
	return func(sc *gorm.DB) *gorm.DB {
		for _, s := range r.s {
			sc = sc.Order(s.String())
		}
		return sc
	}
}

func (r *Request) limit() func(*gorm.DB) *gorm.DB {
	return func(sc *gorm.DB) *gorm.DB {
		if r.l > 0 {
			return sc.Limit(int(r.l))
		}
		return sc
	}
}

func (r *Request) offset() func(*gorm.DB) *gorm.DB {
	return func(sc *gorm.DB) *gorm.DB {
		if r.o > 0 {
			return sc.Offset(int(r.o))
		}
		return sc
	}
}

func (r *Request) paginate(total int64) (p Pagination) {
	p = Pagination{
		Total:   total,
		Limit:   r.l,
		Offset:  r.o,
		Current: 1,
		Last:    1,
	}
	if p.Limit <= 0 {
		p.Limit = total
	}
	if p.Total == 0 {
		return
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	p.Current = p.Offset/p.Limit + 1
	p.Last = (p.Total-1)/p.Limit + 1
	return
}

func NewRequest(filters []Filter, sorts []Sort, pageLimit ...int64) *Request {
	r := Request{
		f: filters,
		s: sorts,
		l: -1,
		o: -1,
	}
	if len(pageLimit) > 0 {
		l := int64(50)
		if len(pageLimit) > 1 {
			l = pageLimit[1]
		}
		p := pageLimit[0]
		if l > 0 {
			r.l = l
			if p > 0 {
				r.o = (p - 1) * l
			}
		}
	}
	return &r
}
