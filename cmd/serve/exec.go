package serve

import (
	"net/http"
	"pmanager/log"
)

func Exec() {
	for rn, rf := range routes {
		http.HandleFunc(rn, rf)
	}
	url := ":" + port
	if err := http.ListenAndServe(url, nil); err != nil {
		log.Fatalln(err)
	}
	log.Debugf("Server started: %s\n", url)
}
