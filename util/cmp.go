package util

import (
	"time"
)

func CompareString(e1, e2 string) int {
	switch {
	case e1 < e2:
		return -1
	case e1 > e2:
		return 1
	}
	return 0
}

func CompareInt(e1, e2 int64) int {
	switch {
	case e1 < e2:
		return -1
	case e1 > e2:
		return 1
	}
	return 0
}

func CompareDate(e1, e2 time.Time) int {
	switch {
	case e1.Before(e2):
		return -1
	case e1.After(e2):
		return 1
	}
	return 0
}

func CompareBool(e1, e2 bool) int {
	if e1 == e2 {
		return 0
	}
	if e1 {
		return 1
	}
	return -1
}
