package db

import (
	"fmt"
	"os"
	"path"
	"pmanager/conf"
	"pmanager/util"
	"reflect"
	"sort"
	"strings"
	"sync"
)

var (
	db       database
	tables   []string
	repos    []string
	nonrepos []string
	mtype    map[string]interface{}
)

func init() {
	repos = GetRepoNames()
	nonrepos = []string{"flag", "git", "mirror"}
	tables = append(repos, nonrepos...)
	db = make(database)
	mtype = map[string]interface{}{
		"flag":   Flag{},
		"git":    Git{},
		"mirror": Country{},
	}
	for _, n := range tables {
		db[n] = &Datatable{name: n}
	}
	for _, n := range repos {
		mtype[n] = Package{}
	}
}

func completeName(pkgname, pkgver string) string { return fmt.Sprintf("%s-%s", pkgname, pkgver) }
func repoName(pkgname, pkgver, repo string) string {
	return fmt.Sprintf("%s/%s", repo, completeName(pkgname, pkgver))
}
func fileName(pkgname, pkgver, arch string) string {
	//@TODO modify for tar.zst in 2022
	return fmt.Sprintf("%s-%s.pkg.tar.xz", completeName(pkgname, pkgver), arch)
}

func dbpath() string   { return path.Join(conf.Basedir(), conf.Read("database.subdir")) }
func repopath() string { return path.Join(dbpath(), "repo") }
func ext() string      { return "." + conf.Read("database.extension") }
func mkdir(fp string) error {
	dir := path.Dir(fp)
	return os.MkdirAll(dir, 0755)
}

func GetRepoNames() (names []string) {
	rp := repopath()
	files, err := os.ReadDir(rp)
	if err != nil {
		util.Debugf("\033[1;31mNo repo database found in %s\033[m\n", rp)
		return
	}
	e := ext()
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), e) {
			names = append(names, strings.TrimSuffix(f.Name(), e))
		}
	}
	return
}

type Data interface{}

type CmpFunc func(Data, Data) int

type MatchFunc func(Data) bool

type ReplaceFunc func(Data) Data

type Request struct {
	sort    CmpFunc
	filter  MatchFunc
	replace ReplaceFunc
}

func (req Request) SetFilter(args ...MatchFunc) Request {
	if len(args) == 1 && args[0] == nil {
		req.filter = nil
	} else {
		req.filter = func(e Data) bool {
			for _, cb := range args {
				if !cb(e) {
					return false
				}
			}
			return true
		}
	}
	return req
}

func (req Request) SetSort(args ...CmpFunc) Request {
	if len(args) == 1 && args[0] == nil {
		req.sort = nil
	} else {
		req.sort = func(e1, e2 Data) int {
			for _, cb := range args {
				if c := cb(e1, e2); c != 0 {
					return c
				}
			}
			return 0
		}
	}
	return req
}

func (req Request) ReverseSort() Request {
	if req.sort != nil {
		cb := req.sort
		req.sort = func(e1, e2 Data) int { return -cb(e1, e2) }
	}
	return req
}

func (req Request) SetReplace(args ...ReplaceFunc) Request {
	if len(args) == 1 && args[0] == nil {
		req.replace = nil
	} else {
		req.replace = func(e Data) Data {
			for _, cb := range args {
				e = cb(e)
			}
			return e
		}
	}
	return req
}

type Pagination map[string]int64

func NewPagination(limit, page int64) Pagination {
	return Pagination{
		"limit": limit,
		"page":  page,
	}
}

func (p Pagination) Set(name string, value int64) Pagination {
	p[name] = value
	return p
}

func (p Pagination) Paginate(dl Datalist) Datalist {
	total := int64(len(dl))
	limit := p["limit"]
	if limit == 0 {
		limit = total
	}
	offset := (p["page"] - 1) * limit
	last := int64(1)
	if limit > 0 {
		last = (total-1)/limit + 1
	}
	p.
		Set("total", total).
		Set("limit", limit).
		Set("offset", offset).
		Set("last", last)
	return dl.Slice(limit, offset)
}

type Datalist []Data

func clone(src Data, dest interface{}) {
	vdest := reflect.ValueOf(dest)
	vsrc := reflect.ValueOf(src)
	reflect.Indirect(vdest).Set(vsrc)
}

func DatalistOf(src interface{}) Datalist {
	v := reflect.ValueOf(src)
	if v.Kind() != reflect.Slice {
		return Datalist{src}
	}
	l, c := v.Len(), v.Cap()
	out := make(Datalist, l, c)
	for i := range out {
		out[i] = v.Index(i).Interface()
	}
	return out
}

func (dl *Datalist) Add(data ...Data) {
	*dl = append(*dl, data...)
}

func (dl *Datalist) AddUniq(data ...Data) {
	done := make(map[Data]bool)
	for _, e := range *dl {
		done[e] = true
	}
	for _, e := range data {
		if !done[e] {
			dl.Add(e)
			done[e] = true
		}
	}
}

func (dl *Datalist) Remove(data ...Data) {
	m := make(map[Data]bool)
	for _, e := range data {
		m[e] = true
	}
	dlNew := make(Datalist, 0, len(*dl))
	for _, e := range *dl {
		if !m[e] {
			dlNew.Add(e)
		}
	}
	*dl = dlNew
}

func (dl Datalist) Replace(req Request) {
	if req.replace == nil {
		return
	}
	if req.filter == nil {
		req.filter = func(Data) bool { return true }
	}
	for i, e := range dl {
		if req.filter(e) {
			dl[i] = req.replace(e)
		}
	}
}

func (dl Datalist) Filter(cb MatchFunc) Datalist {
	if cb == nil {
		return dl
	}
	var out Datalist
	for _, e := range dl {
		if cb(e) {
			out.Add(e)
		}
	}
	return out
}

func (dl Datalist) Sort(cb CmpFunc) Datalist {
	less := func(i, j int) bool { return cb(dl[i], dl[j]) <= 0 }
	sort.Slice(dl, less)
	return dl
}

func (dl Datalist) Slice(limit, offset int64) Datalist {
	l := int64(len(dl))
	if limit <= 0 || offset >= l {
		return nil
	}
	if offset >= 0 {
		dl = dl[offset:]
	}
	if limit < l {
		dl = dl[:limit]
	}
	return dl
}

func (dl Datalist) Search(req Request) Datalist {
	return dl.
		Filter(req.filter).
		Sort(req.sort)
}

func (dl Datalist) First(cb MatchFunc) (d Data, ok bool) {
	if ok = len(dl) > 0; !ok {
		return
	}
	if cb == nil {
		return dl[0], true
	}
	ok = false
	for _, e := range dl {
		if ok = cb(e); ok {
			d = e
			return
		}
	}
	return
}

func (dl Datalist) Contains(cb MatchFunc) bool {
	_, ok := dl.First(cb)
	return ok
}

func (dl Datalist) clone(dest interface{}) {
	t := reflect.TypeOf(dest)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Slice {
		panic("dest must be a pointer of slice")
	}
	v := reflect.MakeSlice(t.Elem(), len(dl), cap(dl))
	for i, e := range dl {
		vi := v.Index(i).Addr()
		clone(e, vi.Interface())
	}
	vdest := reflect.ValueOf(dest)
	reflect.Indirect(vdest).Set(v)
}

type Datatable struct {
	sync.Mutex
	Datalist
	name   string
	loaded bool
}

func (dt *Datatable) GetAll() Datalist {
	dt.Lock()
	defer dt.Unlock()
	out := make(Datalist, len(dt.Datalist))
	copy(out, dt.Datalist)
	return out
}

func (dt *Datatable) IsRepo() bool {
	for _, n := range repos {
		if n == dt.name {
			return true
		}
	}
	return false
}

func (dt *Datatable) IsLoaded() bool {
	return dt.loaded
}

func (dt *Datatable) newDatalist() reflect.Value {
	t := reflect.TypeOf(mtype[dt.name])
	ts := reflect.SliceOf(t)
	return reflect.New(ts)
}

func (dt *Datatable) file() string {
	var basepath string
	if dt.IsRepo() {
		basepath = repopath()
	} else {
		basepath = dbpath()
	}
	return path.Join(basepath, dt.name+ext())
}

func (dt *Datatable) load() error {
	data := dt.newDatalist()
	fpath := dt.file()
	f, err := os.Open(fpath)
	if err != nil {
		return nil
	}
	defer f.Close()
	err = util.ReadJSON(f, data.Interface())
	if err != nil {
		err = fmt.Errorf("Failed to load %s: %s", fpath, err)
		return err
	}
	dt.Datalist = DatalistOf(reflect.Indirect(data).Interface())
	return nil
}

func (dt *Datatable) save() error {
	if !dt.IsLoaded() && len(dt.Datalist) == 0 {
		util.Debugln("Nothing to save")
		return nil
	}
	fpath := dt.file()
	util.Debugf("Saving %s in %s\n", dt.name, fpath)
	if err := mkdir(fpath); err != nil {
		return err
	}
	f, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer f.Close()
	return util.WriteJSON(f, dt.Datalist)
}

func (dt *Datatable) Load(force ...bool) {
	f := len(force) == 1 && force[0]
	if dt.IsLoaded() && !f {
		return
	}
	dt.Lock()
	defer dt.Unlock()
	err := dt.load()
	if dt.loaded = err == nil; !dt.loaded {
		util.Printf("\033[1;31m%s\033[m\n", err)
	} else {
		util.Debugf("%s database loaded\n", dt.name)
	}
}

func (dt *Datatable) Save() error {
	dt.Lock()
	defer dt.Unlock()
	return dt.save()
}

func (dt *Datatable) Set(data interface{}, save ...bool) error {
	s := len(save) == 1 && save[0]
	dt.Lock()
	defer dt.Unlock()
	dt.Datalist = DatalistOf(data)
	if s {
		return dt.save()
	}
	return nil
}

type database map[string]*Datatable

func getTables(name string) []string {
	var tn []string
	switch name {
	case "all":
		tn = tables
	case "package":
		tn = repos
	default:
		tn = []string{name}
	}
	return tn
}

func (self database) get(name string) (out Datalist) {
	if name != "package" {
		return Table(name).GetAll()
	}
	for _, n := range repos {
		out.Add(self[n].GetAll()...)
	}
	return
}

func HasTable(name string) bool {
	_, ok := db[name]
	return ok
}

func Table(name string) *Datatable {
	if t, ok := db[name]; ok {
		return t
	}
	t := &Datatable{name: name}
	tables = append(tables, name)
	repos = append(repos, name)
	mtype[name] = Package{}
	db[name] = t
	return t
}

func All(name string, dest interface{}) int64 {
	dl := db.get(name)
	dl.clone(dest)
	return int64(len(dl))
}

func FindAll(name string, dest interface{}, req Request) int64 {
	dl := db.get(name).Search(req)
	dl.clone(dest)
	return int64(len(dl))
}

func Paginate(name string, dest interface{}, req Request, pagination Pagination) {
	dl := db.get(name).Search(req)
	dl = pagination.Paginate(dl)
	dl.clone(dest)
}

func Find(name string, dest interface{}, cb MatchFunc) bool {
	data, ok := db.get(name).First(cb)
	if ok {
		clone(data, dest)
	}
	return ok
}

func Contains(name string, cb MatchFunc) bool {
	return db.get(name).Contains(cb)
}

func Load(name string, force ...bool) {
	for _, n := range getTables(name) {
		Table(n).Load(force...)
	}
}

func Save(name string) {
	for _, n := range getTables(name) {
		if err := Table(n).Save(); err != nil {
			util.Printf("\033[1;31mError at saving %s: %s\033[m\n", n, err)
		}
	}
}

func Set(name string, data interface{}) {
	if err := Table(name).Set(data, true); err != nil {
		util.Printf("\033[1;31mError at saving %s: %s\033[m\n", name, err)
	}
}

func Remove(name string, data interface{}) {
	dl := DatalistOf(data)
	dt := Table(name)
	dt.Lock()
	defer dt.Unlock()
	dt.load()
	dt.Remove(dl...)
	if err := dt.save(); err != nil {
		util.Printf("\033[1;31mError at saving %s: %s\033[m\n", name, err)
	}
}

func Add(name string, data interface{}, reload ...bool) {
	dl := DatalistOf(data)
	dt := Table(name)
	dt.Lock()
	defer dt.Unlock()
	if len(reload) > 0 && reload[0] {
		dt.load()
	}
	dt.AddUniq(dl...)
	if err := dt.save(); err != nil {
		util.Printf("\033[1;31mError at saving %s: %s\033[m\n", name, err)
	}
}

func Replace(name string, req Request, reload ...bool) {
	dt := Table(name)
	dt.Lock()
	defer dt.Unlock()
	if len(reload) > 0 && reload[0] {
		dt.load()
	}
	dt.Replace(req)
	if err := dt.save(); err != nil {
		util.Printf("\033[1;31mError at saving %s: %s\033[m\n", name, err)
	}
}
