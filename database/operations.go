package database

import (
	"fmt"
	"pmanager/log"
	"reflect"
	"sync"

	"gorm.io/gorm"
)

func UpdateMirrors(pacmanConf, pacmanMirrors, mainMirrorName string) map[string]int {
	countries, err := getMirrors(pacmanConf, pacmanMirrors, mainMirrorName)
	if err != nil {
		log.Fatalf("Failed to get mirrors: %s\n", err)
	}
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	if err = dbsingleton.Transaction(updateMirrors(countries)); err != nil {
		log.Fatalf("Failed to update mirrors database: %s\n", err)
	}
	c, m := len(countries), 0
	for _, e := range countries {
		m += len(e.Mirrors)
	}
	return map[string]int{
		"countries": c,
		"mirrors":   m,
	}
}

func UpdatePackages(base, extension string, excludes []string) map[string]int {
	packages, err := getPackages(base, extension, excludes)
	if err != nil {
		log.Fatalf("Failed to get packages: %s\n", err)
	}
	oldPackages := findAllPackages()
	add, update, remove, removeFlags := unzipPackages(oldPackages, packages)
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	if err = dbsingleton.Transaction(updatePackages(add, update, remove, removeFlags)); err != nil {
		log.Fatalf("Failed to update packages database: %s\n", err)
	}
	return map[string]int{
		"packages_added":   len(add),
		"packages_updated": len(update),
		"packages_removed": len(remove),
		"flags_removed":    len(removeFlags),
	}
}

func UpdateAll(
	pacmanConf,
	pacmanMirrors,
	mainMirrorName,
	base,
	extension string,
	excludes []string,
) map[string]int {
	var wg sync.WaitGroup
	var err error
	var countries []Country
	var add, update, remove []Package
	var removeFlags []Flag

	wg.Add(2)
	go func() {
		defer wg.Done()
		if countries, err = getMirrors(pacmanConf, pacmanMirrors, mainMirrorName); err != nil {
			log.Errorf("Failed to get mirrors: %s\n", err)
		}
	}()
	go func() {
		defer wg.Done()
		var packages, oldPackages []Package
		if packages, err = getPackages(base, extension, excludes); err != nil {
			log.Errorf("Failed to get packages: %s\n", err)
			return
		}
		oldPackages = findAllPackages()
		add, update, remove, removeFlags = unzipPackages(oldPackages, packages)
	}()
	wg.Wait()

	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	err = dbsingleton.Transaction(func(tx *gorm.DB) (err error) {
		if err = updateMirrors(countries)(tx); err != nil {
			return
		}
		return updatePackages(add, update, remove, removeFlags)(tx)
	})
	if err != nil {
		log.Fatalf("Failed to update database: %s\n", err)
	}
	c, m := len(countries), 0
	for _, e := range countries {
		m += len(e.Mirrors)
	}
	return map[string]int{
		"countries":        c,
		"mirrors":          m,
		"packages_added":   len(add),
		"packages_updated": len(update),
		"packages_removed": len(remove),
		"flags_removed":    len(removeFlags),
	}
}

func First(e interface{}, r *Request) bool {
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	return dbsingleton.Scopes(r.where(), r.order()).First(e).Error == nil
}

func Search(e interface{}, r *Request) bool {
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	return dbsingleton.Scopes(r.where(), r.order(), r.limit(), r.offset()).Find(e).Error == nil
}

func SearchAll(e interface{}) bool {
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	return dbsingleton.Find(e).Error == nil
}

func Paginate(e interface{}, r *Request) (p Pagination, ok bool) {
	t := reflect.TypeOf(e)
	if t.Kind() != reflect.Ptr {
		log.Errorln("Not a pointer")
		return
	}
	t = t.Elem()
	if t.Kind() != reflect.Slice {
		log.Errorln("Not a pointer of slice")
		return
	}
	v := reflect.New(t.Elem()).Interface()
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	w, o := r.where(), r.order()
	var c int64
	if err := dbsingleton.Model(v).Scopes(w, o).Count(&c).Error; err != nil {
		return
	}
	p = r.paginate(c)
	if c > 0 {
		ok = dbsingleton.Scopes(w, o, r.limit(), r.offset()).Find(e).Error != nil
	} else {
		ok = true
	}
	return
}

func GetPackage(p *Package, r *Request, base string) (ok bool) {
	if ok = Search(p, r); !ok {
		return
	}
	if p.FlagID == 0 && p.Repository != "build" {
		pb := new(Package)
		if Search(pb, NewFilterRequest(
			NewFilter("repository", "=", "build"),
			NewFilter("name", "=", p.Name),
		)) {
			p.BuildVersion = pb
		}
	}
	if p.GitID == 0 {
		if searchGit(base, p) {
			dbsingleton.Lock()
			defer dbsingleton.Unlock()
			dbsingleton.Transaction(updateGit(&p.Git, p.Name))
			p.GitID = p.Git.ID
		}
	}
	return
}

func CreateFlag(p *Package) error {
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	return dbsingleton.Transaction(createFlag(p))
}

func SumSizes(r *Request, field string) (c int64) {
	w, o := r.where(), r.order()
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	dbsingleton.
		Model(&Package{}).
		Select(fmt.Sprintf("sum(%s)", field)).
		Scopes(w, o).
		Scan(&c)
	return
}