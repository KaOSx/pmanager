package update

import (
	"bytes"
	"fmt"
	"net/http"
	"pmanager/conf"
	"pmanager/log"
	"pmanager/util/conv"
)

func updateApi(t string) {
	url := fmt.Sprintf("http://localhost:%s/update/%s", conf.String("api.port"), t)
	data, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	if data.Body != nil {
		defer data.Body.Close()
		log.Copy(data.Body)
	}
}

func updateServer(t string) {
	data := upd[t]()
	if data != nil {
		var buf bytes.Buffer
		if conv.WriteJson(&buf, data, log.Debug) != nil {
			log.Copy(&buf)
		}
	}
}

func update(t string) {
	if serverOpen {
		updateApi(t)
	} else {
		updateServer(t)
	}
}

func Mirrors() {
	update("mirror")
}

func Repositories() {
	update("repo")
}

func All() {
	update("all")
}
