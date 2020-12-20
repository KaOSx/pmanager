package util

import (
	"fmt"
	"net/http"
	"pmanager/conf"
)

func Refresh(name string) {
	port := conf.Read("api.port")
	url := fmt.Sprintf("http://localhost:%s/refresh?type=%s", port, name)
	resp, err := http.Get(url)
	if err != nil {
		Debugf("Refresh unsuccessful: %s\n", err)
		return
	}
	resp.Body.Close()
}
