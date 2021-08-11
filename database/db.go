package database

import (
	"database/sql/driver"
	"encoding/json"
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
	bytes, err := json.Marshal(v)
	if err == nil {
		err = json.Unmarshal(bytes, sl)
	}
	return err
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
