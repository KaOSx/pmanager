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
