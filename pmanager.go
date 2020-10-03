package main

import (
	"fmt"
	"os"
	"pmanager/cmd/api"
	"pmanager/cmd/flag"
	"pmanager/cmd/mailtest"
	"pmanager/cmd/mirror"
	"pmanager/cmd/repositories"
	"pmanager/cmd/updateall"
	"pmanager/conf"
)

var actions = map[string]func([]string){
	"update-repos":   repositories.Update,
	"update-mirrors": mirror.Update,
	"update-all":     updateall.Update,
	"serve":          api.Serve,
	"flag":           flag.Exec,
	"test-mail":      mailtest.SendMail,
}

const help = `
Available subcommands:

  update-repos [repos...]
    update the repo databases; if no repo selected, update all repos

  update-mirrors
    update the mirrors database

  update-all
    do update-repos and update-mirrors together

  serve
    Launch the API webserver

  flag
    Launch the prompt to manage the flags

  test-mail
    Try to send mail from reading configuration

Available Routes:

  /flag/list
    search=<pkgname pattern>
    repo=<repository>
    from=<minimum date of submission>
    to=<maximum date of submission>
    flagged=(0|1)
    sortby=(repo|name|date|flagged)
    sortdir=(asc|desc) (default: asc)
    page=<page number to display>
    limit=<max number of result> (default: defined in configuration, parameter pagination of section [api])

  /flag/add
    name=<pkgname>
    version=<pkgver>
    repo=<repository>
    email=<email of submitter>
    comment=<comment of submitter>

  /package/list
    exact=(0|1) (to search package with exact name)
    search=<pkgname pattern>
    from=<minimum date of build>
    to=<maximum date of build>
    flagged=(0|1)
    sortby=(repo|name|date|flagged)
    sortdir=(asc|desc) (default: asc)
    page=<page number to display>
    limit=<max number of result> (default: defined in configuration, parameter pagination of section [api])

  /package/view
    name=<pkgname-pkgver>
    repo=<repository>

  /repo/list
    repo=<repository>
    search=<pkgname pattern>
    from=<minimum date of build>
    to=<maximum date of build>
    flagged=(0|1)
    sortby=(repo|name|date|flagged)
    sortdir=(asc|desc) (default: asc)
    page=<page number to display>
    limit=<max number of result> (default: defined in configuration, parameter pagination of section [api])

  /mirror

  /all
`

func printUsage() {
	w := os.Stderr
	fmt.Fprintf(w, "\033[1;31mUsage: %s (subcommand) [args...]\033[m\n", os.Args[0])
	fmt.Fprint(w, help)
}

func main() {
	conf.Load()
	if len(os.Args) > 1 {
		action, ok := actions[os.Args[1]]
		if ok {
			action(os.Args[2:])
			return
		}
	}
	printUsage()
	os.Exit(1)
}
