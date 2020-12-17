package flag

import (
	"fmt"
	"pmanager/db"
	"pmanager/util"
	"pmanager/util/shell"
	"sort"
	"strconv"
	"strings"
	"time"
)

type fs struct {
	format func(string) (interface{}, bool)
	filter map[string]func(interface{}) func(*db.Flag) bool
	cmp    func(*db.Flag, *db.Flag) int
}

const help = `Available commands:
  help: display this help
  list [<filter/order options>] : list the flags
  unflag (all|<range id>):        unflag selected flags (needs to launch list before)
  flag (all|<range id>):          flag selected flags (needs to launch list before)
  delete (all|<range id>):        delete selected flags (needs to launch list before)
  quit:                           exit the prompt

Range id formating:
  1,2,8:      select the flags with an id 1, 2 or 8
  2-5:        select the flags with an id between 2 and 5
  5-2:        same as 2-5
  1,4-5,18,2: mixing range and discrete values

Available filters:
  date > <YYYYMMDD[HHMM]>:  date after value
  date < <YYYYMMDD[HHMM]>:  date before value
  date = <YYYYMMDD[HHMM]>:  date equal value
  date <> <YYYYMMDD[HHMM]>: date different from value
  email = <pattern>:        email equal value
  email ~ <pattern>:        email contains substring
  repo = <name>:            repository equal value
  package = <pattern>:      package equal value
  package ~ <pattern>:      package contains substring
  flagged = (0|1):          state of the flag

Available sorts: (apply with sortby <field> (asc|desc))
  date
  email
  repo
  package
`

var mfs = map[string]fs{
	"date": fs{
		format: func(s string) (e interface{}, ok bool) {
			if len(s) != 8 && len(s) != 12 {
				return
			}
			var y, m, d, h, n int
			var err error
			if y, err = strconv.Atoi(s[:4]); err != nil {
				return
			}
			if m, err = strconv.Atoi(s[4:6]); err != nil {
				return
			}
			if d, err = strconv.Atoi(s[6:8]); err != nil {
				return
			}
			if len(s) == 12 {
				if h, err = strconv.Atoi(s[8:10]); err != nil {
					return
				}
				if n, err = strconv.Atoi(s[10:]); err != nil {
					return
				}
			}
			return time.Date(y, time.Month(m), d, h, n, 0, 0, time.Local), true
		},
		filter: map[string]func(interface{}) func(*db.Flag) bool{
			">": func(e interface{}) func(f *db.Flag) bool {
				return func(f *db.Flag) bool { return f.Date.After(e.(time.Time)) }
			},
			"<": func(e interface{}) func(f *db.Flag) bool {
				return func(f *db.Flag) bool { return f.Date.Before(e.(time.Time)) }
			},
			"=": func(e interface{}) func(f *db.Flag) bool {
				return func(f *db.Flag) bool { return f.Date.Equal(e.(time.Time)) }
			},
			"<>": func(e interface{}) func(f *db.Flag) bool {
				return func(f *db.Flag) bool { return !f.Date.Equal(e.(time.Time)) }
			},
		},
		cmp: func(f1, f2 *db.Flag) int {
			if f1.Date.Equal(f2.Date) {
				return 0
			}
			if f1.Date.Before(f2.Date) {
				return -1
			}
			return 1
		},
	},
	"email": fs{
		filter: map[string]func(interface{}) func(*db.Flag) bool{
			"=": func(e interface{}) func(*db.Flag) bool { return func(f *db.Flag) bool { return f.Email == e.(string) } },
			"~": func(e interface{}) func(*db.Flag) bool {
				return func(f *db.Flag) bool { return strings.Contains(f.Email, e.(string)) }
			},
		},
		cmp: func(f1, f2 *db.Flag) int { return util.CompareString(f1.Email, f2.Email) },
	},
	"repo": fs{
		filter: map[string]func(interface{}) func(*db.Flag) bool{
			"=": func(e interface{}) func(*db.Flag) bool {
				return func(f *db.Flag) bool { return f.Repository == e.(string) }
			},
		},
		cmp: func(f1, f2 *db.Flag) int { return util.CompareString(f1.Repository, f2.Repository) },
	},
	"package": fs{
		filter: map[string]func(interface{}) func(*db.Flag) bool{
			"=": func(e interface{}) func(*db.Flag) bool { return func(f *db.Flag) bool { return f.Name == e.(string) } },
			"~": func(e interface{}) func(*db.Flag) bool {
				return func(f *db.Flag) bool { return strings.Contains(f.Name, e.(string)) }
			},
		},
		cmp: func(f1, f2 *db.Flag) int { return util.CompareString(f1.CompleteName(), f2.CompleteName()) },
	},
	"flagged": fs{
		format: func(s string) (e interface{}, ok bool) {
			v, err := strconv.ParseBool(s)
			if err == nil {
				return v, true
			}
			return
		},
		filter: map[string]func(interface{}) func(*db.Flag) bool{
			"=": func(e interface{}) func(*db.Flag) bool { return func(f *db.Flag) bool { return f.Flagged == e.(bool) } },
		},
	},
}

var flags *db.Flaglist

func getRange(arg string) (rg []int, all bool) {
	if all = arg == "all"; all {
		for i := range *flags {
			rg = append(rg, i+1)
		}
		return
	}
	srg := strings.Split(arg, ",")
	for _, e := range srg {
		r := strings.SplitN(e, "-", 2)
		if len(r) == 1 {
			if i, err := strconv.Atoi(r[0]); err == nil {
				rg = append(rg, i)
			}
		} else {
			i1, e1 := strconv.Atoi(r[0])
			i2, e2 := strconv.Atoi(r[1])
			if e1 == nil && e2 == nil {
				if i1 > i2 {
					i1, i2 = i2, i1
				}
				for i := i1; i <= i2; i++ {
					rg = append(rg, i)
				}
			}
		}
	}
	return
}

func getIds(args []string) (rg []int) {
	var all bool
	var ids []int
	for _, e := range args {
		rg, all = getRange(e)
		if all {
			return
		}
		ids = append(ids, rg...)
	}
	sort.Ints(ids)
	rg = make([]int, 0, len(ids))
	l := flags.Len()
	for _, i := range ids {
		if i > 0 && i <= l && (len(rg) == 0 || rg[len(rg)-1] != i) {
			rg = append(rg, i)
		}
	}
	return
}

func parseNextArg(args []string) (f func(*db.Flag) bool, c func(f1, f2 *db.Flag) int, next []string, err error) {
	if args[0] == "sortby" {
		if len(args) < 2 {
			err = fmt.Errorf("Missing field after sortby")
			return
		}
		e, ok := mfs[args[1]]
		if !ok {
			err = fmt.Errorf("%s is not a field", args[1])
			return
		}
		if e.cmp == nil {
			err = fmt.Errorf("%s is not sortable", args[1])
			return
		}
		n, asc := 2, true
		if len(args) > 2 {
			if args[2] == "asc" {
				n++
			} else if args[2] == "desc" {
				n++
				asc = false
			}
		}
		next = args[n:]
		c = e.cmp
		if !asc {
			c = func(f1, f2 *db.Flag) int { return -e.cmp(f1, f2) }
		}
		return
	}
	e, ok := mfs[args[0]]
	if !ok {
		err = fmt.Errorf("%s is not a field", args[0])
		return
	}
	if len(args) < 2 {
		err = fmt.Errorf("missing operator after filter %s", args[0])
		return
	}
	sf, ok := e.filter[args[1]]
	if !ok {
		err = fmt.Errorf("%s is not an operator or is not supported for field %s", args[1], args[0])
		return
	}
	if len(args) < 3 {
		err = fmt.Errorf("Missing condition after %s %s", args[0], args[1])
	}
	var v interface{} = args[2]
	if e.format != nil {
		if v, ok = e.format(args[2]); !ok {
			fmt.Errorf("Not a valid value for field %s", args[0])
			return
		}
	}
	f, next = sf(v), args[3:]
	return
}

func getFlag(args []string) {
	var filters []func(*db.Flag) bool
	var comparators []func(f1, f2 *db.Flag) int
	var f func(*db.Flag) bool
	var c func(f1, f2 *db.Flag) int
	var e error
	for len(args) > 0 {
		f, c, args, e = parseNextArg(args)
		if e != nil {
			fmt.Println(e)
			return
		}
		if f != nil {
			filters = append(filters, f)
		}
		if c != nil {
			comparators = append(comparators, c)
		}
	}
	flags = db.LoadFlags(true).Filter(filters...).Sort(comparators...)
	for i, f := range *flags {
		flagged := "(outdated)"
		if !f.Flagged {
			flagged = ""
		}
		date := f.Date.Format(time.RFC1123)
		fmt.Printf("\033[1;36m%d\033[m → \033[1;32m%s\033m \033[1;31m%s\033[m\n", i+1, f.RepoName(), flagged)
		fmt.Printf("\033[1mDate:   \033[m %s\n", date)
		fmt.Printf("\033[1mEmail:  \033[m %s\n", f.Email)
		fmt.Printf("\033[1mComment:\033[m %s\n", f.Comment)
	}
}

func changeFlag(flagged bool, args []string) {
	if flags == nil {
		fmt.Println("You need to launch the command list before")
		return
	}
	rg := getIds(args)
	action := "flag"
	if !flagged {
		action = "unflag"
	}
	if len(rg) == 0 {
		fmt.Println("Nothing to", action)
		return
	}
	srg := make([]string, len(rg))
	for i, r := range rg {
		srg[i] = strconv.Itoa(r)
	}
	if shell.GetBool(fmt.Sprintf("%s %s?", action, strings.Join(srg, ", ")), true) {
		filter := func(f0 *db.Flag) func(*db.Flag) bool {
			return func(f *db.Flag) bool {
				return f0.RepoName() == f.RepoName() && f0.Date.Equal(f.Date)
			}
		}
		nfl := db.LoadFlags(true)
		for _, i := range rg {
			f0 := (*flags)[i-1]
			flgs := nfl.Filter(filter(f0))
			for _, f := range *flgs {
				f.Flagged = flagged
			}
			fmt.Printf("%s %sged\n", f0.RepoName(), action)
		}
		db.StoreFlags()
		util.Refresh("flag")
		flags = nil
	} else {
		fmt.Println("cancel…")
	}
}

func deleteFlag(args []string) {
	if flags == nil {
		fmt.Println("You need to launch the command list before")
		return
	}
	rg := getIds(args)
	if len(rg) == 0 {
		fmt.Println("Nothing to delete")
		return
	}
	srg := make([]string, len(rg))
	for i, r := range rg {
		srg[i] = strconv.Itoa(r)
	}
	if shell.GetBool(fmt.Sprintf("delete %s?", strings.Join(srg, ", ")), true) {
		nfl := db.LoadFlags(true)
		for _, i := range rg {
			f0 := (*flags)[i-1]
			nfl.Remove(f0)
			fmt.Println(f0.RepoName(), "deleted")
		}
		db.StoreFlags()
		util.Refresh("flag")
		flags = nil
	} else {
		fmt.Println("cancel…")
	}
}

func Exec([]string) {
	for {
		args := shell.Prompt("> ")
		if len(args) == 0 {
			fmt.Println("Type help for usage")
		} else {
			switch args[0] {
			case "help":
				fmt.Print(help)
			case "quit":
				return
			case "list":
				getFlag(args[1:])
			case "flag":
				changeFlag(true, args[1:])
			case "unflag":
				changeFlag(false, args[1:])
			case "delete":
				deleteFlag(args[1:])
			default:
				fmt.Printf("Command “%s” unknown. Type help for usage\n", args[0])
			}
		}
	}
}
