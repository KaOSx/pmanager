package util

import (
	"fmt"
	"net/http"
	"pmanager/conf"
)

func Refresh(name string) {
	port := conf.Read("api.port")
	url := fmt.Sprintf("http://localhost:%s/refresh?type=%s", port, name)
	_, err := http.Head(url)
	if err != nil {
		Debugf("\033[1;31mRefresh of %s unsuccessful: %s\033[m\n", name, err)
		return
	}
}
