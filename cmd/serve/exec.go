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
	log.Debugf("Server started: %s\n", url)

	if err := http.ListenAndServe(url, nil); err != nil {
		log.Fatalln(err)
	}
}
