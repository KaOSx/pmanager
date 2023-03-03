package flag

import (
	"fmt"
	"net/http"
	"pmanager/conf"
	"pmanager/database"
	"pmanager/log"
	"pmanager/util/conv"
	"pmanager/util/resource"
	"strconv"
	"strings"
)

const help = `Available commands:
  help                           display this help
  list                           list the flags
  delete (all|<range id>)        delete selected flags (needs to launch list before)
  quit                           exit the prompt

Range id formating:
  1,2,8:      select the flags with an id 1, 2 or 8
  2-5:        select the flags with an id between 2 and 5
  5-2:        same as 2-5
  1,4-5,18,2: mixing range and discrete values
`

func listOffline() (flags []database.Flag) {
	database.Search(
		&flags,
		database.NewOrderRequest(database.NewSort("created_at", true)),
	)

	return
}

func listOnline(port string) (flags []database.Flag) {
	url := fmt.Sprintf("http://localhost:%s/flag/list?field=date&asc=0", port)
	data, err := http.Get(url)

	if err != nil {
		log.Fatalln(err)
	}

	if data.Body != nil {
		defer data.Body.Close()
		m := make(conv.Map)
		conv.ReadJson(data.Body, &m)
		if d, ok := m["data"]; ok {
			if err := conv.ToData(d, &flags); err != nil {
				log.Fatalln(err)
			}
		}
	}

	return
}

func listFlags() []database.Flag {
	port := conf.String("api.port")

	if resource.IsPortOpen("localhost", port) {
		return listOnline(port)
	}

	return listOffline()
}

func deleteOffline(ids []uint) int {
	return database.DeleteFlags(ids)
}

func deleteOnline(ids []uint, port string) (c int) {
	sids := make([]string, len(ids))
	for i, id := range ids {
		sids[i] = fmt.Sprint(id)
	}

	url := fmt.Sprintf("http://localhost:%s/flag/delete?ids=%s", port, strings.Join(sids, ","))
	data, err := http.Get(url)

	if err != nil {
		log.Fatalln(err)
	}

	if data.Body != nil {
		defer data.Body.Close()
		m := make(conv.Map)
		conv.ReadJson(data.Body, &m)
		c = int(m.GetInt("flags_deleted"))
	}

	return
}

func deleteFlags(ids []uint) int {
	port := conf.String("api.port")

	if resource.IsPortOpen("localhost", port) {
		return deleteOnline(ids, port)
	}

	return deleteOffline(ids)
}

func getRange(arg string, c int) (rg []int) {
	if arg == "all" {
		rg = make([]int, c)
		for i := range rg {
			rg[i] = i
		}
		return
	}

	srg := strings.Split(arg, ",")
	for _, e := range srg {
		r := strings.SplitN(e, "-", 2)
		if len(r) == 1 {
			if i, err := strconv.Atoi(r[0]); err == nil && i > 0 && i <= c {
				rg = append(rg, i-1)
			}
		} else {
			i1, e1 := strconv.Atoi(r[0])
			i2, e2 := strconv.Atoi(r[1])
			if e1 == nil && e2 == nil {
				if i1 > i2 {
					i1, i2 = i2, i1
				}
				for i := i1; i <= i2; i++ {
					if i > 0 && i <= c {
						rg = append(rg, i-1)
					}
				}
			}
		}
	}

	return
}

func getIds(args []string, flags []database.Flag) (ids []uint) {
	c := len(flags)
	done := make(map[int]bool)

	for _, e := range args {
		rg := getRange(e, c)
		for _, i := range rg {
			if !done[i] {
				done[i] = true
				ids = append(ids, flags[i].ID)
			}
		}
	}

	return
}
