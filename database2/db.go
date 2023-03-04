package database

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// database is a representation of a database in Go.
// It is safe-thread for reading and writing.
type database struct {
	sync.Mutex
	*gorm.DB
}

func newDb(connector gorm.Dialector, tables ...any) (dbl *database, err error) {
	var db *gorm.DB

	db, err = gorm.Open(connector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return
	}

	dbl, err = &database{DB: db}, dbl.AutoMigrate(tables...)

	return
}

// SqlSlice is a custom type of field to
// store slice informations in the database.
type SqlSlice []string

func (sl *SqlSlice) Scan(v any) error {
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

// Filter is a structure
// to represent the clause WHERE
// of an SQL request.
type Filter struct {
	field     string
	operation string
	value     any
}

// NewFilter creates a new filter clause.
// - field : name of the field to apply the filter
// - operation : comparison operator between the field and the value
// - value : value to compare with the field
func NewFilter(field, operation string, value any) Filter {
	return Filter{
		field:     field,
		operation: operation,
		value:     value,
	}
}

func (f Filter) String() string {
	return fmt.Sprintf("%s %s ?", f.field, f.operation)
}

// Sort is a structure
// to represent the clause ORDER BY
// of an SQL request.
type Sort struct {
	field string
	desc  bool
}

// NewSort creates a new sort clause.
// - field : name of the field to sort
// - desc : if true, sort is descending, and ascending otherwise
func NewSort(field string, desc bool) Sort {
	return Sort{
		field: field,
		desc:  desc,
	}
}

func (s Sort) String() string {
	direction := "ASC"
	if s.desc {
		direction = "DESC"
	}

	return fmt.Sprintf("%s %s", s.field, direction)
}

// Pagination is a structure
// to store the information the paginated result.
type Pagination struct {
	Total   int64 // Total of elements who match the filter request
	Limit   int64 // Number of elements per page
	Offset  int64 // First element position to display in the result
	Current int64 // Current page of the result
	Last    int64 // Number of pages of the result
}

// Request is a structure to prepare an SQL request.
type Request struct {
	filters []Filter
	sorts   []Sort
	lim     int64
	off     int64
}

// NewRequest returns a new prepared request.
//   - filters : list of filters to apply
//   - sorts : list of order criterias to apply
//   - pageLimit (optionnal) : if set, first number gives the page of the result (default 1)
//     and second number gives the limit of the results (default 50)
func NewRequest(filters []Filter, sorts []Sort, pageLimit ...int64) *Request {
	r := Request{
		filters: filters,
		sorts:   sorts,
		lim:     -1,
		off:     -1,
	}

	if len(pageLimit) > 0 {
		limit := int64(50)
		if len(pageLimit) > 1 {
			limit = pageLimit[1]
		}
		page := pageLimit[0]
		if limit > 0 {
			r.SetLimit(limit).SetPage(page)
		}
	}

	return &r
}

func NewFilterRequest(filters ...Filter) *Request {
	return NewRequest(filters, nil)
}

func NewOrderRequest(sorts ...Sort) *Request {
	return NewRequest(nil, sorts)
}

func (r *Request) AddFilter(field, operation string, value any) *Request {
	r.filters = append(r.filters, NewFilter(field, operation, value))

	return r
}

func (r *Request) AddSort(field string, desc bool) *Request {
	r.sorts = append(r.sorts, NewSort(field, desc))

	return r
}

func (r *Request) SetLimit(limit int64) *Request {
	r.lim = limit

	return r
}

func (r *Request) SetPage(page int64) *Request {
	if r.lim > 0 {
		if page > 0 {
			r.off = (page - 1) * r.lim
		}
	}

	return r
}

func (r *Request) where() func(*gorm.DB) *gorm.DB {
	return func(sc *gorm.DB) *gorm.DB {
		l := len(r.filters)
		if l == 0 {
			return sc
		}

		filters, values := make([]string, l), make([]any, l)
		for i, f := range r.filters {
			filters[i] = f.String()
			values[i] = f.value
		}
		filter := strings.Join(filters, " AND ")

		return sc.Where(filter, values...)
	}
}

func (r *Request) order() func(*gorm.DB) *gorm.DB {
	return func(sc *gorm.DB) *gorm.DB {
		for _, s := range r.sorts {
			sc = sc.Order(s.String())
		}

		return sc
	}
}

func (r *Request) limit() func(*gorm.DB) *gorm.DB {
	return func(sc *gorm.DB) *gorm.DB {
		if r.lim > 0 {
			return sc.Limit(int(r.lim))
		}
		return sc
	}
}

func (r *Request) offset() func(*gorm.DB) *gorm.DB {
	return func(sc *gorm.DB) *gorm.DB {
		if r.off > 0 {
			return sc.Offset(int(r.off))
		}
		return sc
	}
}

func (r *Request) paginate(total int64) (p Pagination) {
	p = Pagination{
		Total:   total,
		Limit:   r.lim,
		Offset:  r.off,
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
