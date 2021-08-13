package database

import (
	"pmanager/log"
	"sync"

	"gorm.io/gorm"
)

func UpdateMirrors(pacmanConf, pacmanMirrors, mainMirrorName string, debug bool) {
	countries, err := getMirrors(pacmanConf, pacmanMirrors, mainMirrorName, debug)
	if err != nil {
		log.Fatalf("Failed to get mirrors: %s\n", err)
	}
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	if err = dbsingleton.Transaction(updateMirrors(countries)); err != nil {
		log.Fatalf("Failed to update mirrors database: %s\n", err)
	}
}

func UpdatePackages(base, extension string, excludes []string, debug bool) {
	packages, err := getPackages(base, extension, excludes)
	if err != nil {
		log.Fatalf("Failed to get packages: %s\n", err)
	}
	oldPackages := findAllPackages()
	add, update, remove := unzipPackages(oldPackages, packages)
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	if err = dbsingleton.Transaction(updatePackages(add, update, remove)); err != nil {
		log.Fatalf("Failed to update packages database: %s\n", err)
	}
}

func UpdateAll(
	pacmanConf,
	pacmanMirrors,
	mainMirrorName,
	base,
	extension string,
	excludes []string,
	debug bool,
) {
	var wg sync.WaitGroup
	var err error
	var countries []Country
	var add, update, remove []Package

	wg.Add(2)
	go func() {
		defer wg.Done()
		if countries, err = getMirrors(pacmanConf, pacmanMirrors, mainMirrorName, debug); err != nil {
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
		add, update, remove = unzipPackages(oldPackages, packages)
	}()
	wg.Wait()

	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	err = dbsingleton.Transaction(func(tx *gorm.DB) (err error) {
		if err = updateMirrors(countries)(tx); err != nil {
			return
		}
		return updatePackages(add, update, remove)(tx)
	})
	if err != nil {
		log.Fatalf("Failed to update database: %s\n", err)
	}
}
