package updateall

import (
	"pmanager/cmd/mirror"
	"pmanager/cmd/repositories"
)

func Update([]string) {
	repositories.Update(nil)
	mirror.Update(nil)
}
