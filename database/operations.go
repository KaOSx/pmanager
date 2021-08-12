package database

import (
	"pmanager/log"
)

func UpdateMirrors(pacmanConf, pacmanMirrors, mainMirrorName string, debug bool) {
	countries, err := getMirrors(pacmanConf, pacmanMirrors, mainMirrorName, debug)
	if err != nil {
		log.Fatalf("Failed to get mirrors: %s\n", err)
	}
	dbsingleton.Lock()
	defer dbsingleton.Unlock()
	err = updateMirrors(countries)
	if err != nil {
		log.Fatalf("Failed to update mirrors database: %s\n", err)
	}
}
