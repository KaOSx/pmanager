package conv

import (
	"fmt"
)

func ToSize(size int64) string {
	u := uint64(0)
	m := map[uint64]string{
		10: "K",
		20: "M",
		30: "G",
	}

	for u < 30 {
		if size < (1 << (u + 10)) {
			break
		}
		u += 10
	}

	if u == 0 {
		return fmt.Sprint(size)
	}

	e := size >> u
	f := (size & ((1 << u) - 1)) * 10 >> u

	return fmt.Sprintf("%d.%d%s", e, f, m[u])
}
