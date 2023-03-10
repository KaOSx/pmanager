package main

import (
	"fmt"
	"os"
	"pmanager/cmd/flag"
	"pmanager/cmd/mailtest"
	"pmanager/cmd/serve"
	"pmanager/cmd/update"
	"pmanager/log"

	_ "pmanager/conf"
)

var actions = map[string]func(){
	"update-repos":   update.Repositories,
	"update-mirrors": update.Mirrors,
	"update-all":     update.All,
	"serve":          serve.Exec,
	"flag":           flag.Exec,
	"test-mail":      mailtest.SendMail,
	"help":           printUsage,
}

const help = `
Available subcommands:

  update-repos
    update the list of packages in all repos

  update-mirrors
    update the mirrors

  update-all
    do update-repos and update-mirrors together

  serve
    Launch the API webserver

  flag
    Launch the prompt to manage the flags

  test-mail
    Try to send mail from reading configuration

Available arguments:

  --debug|--no-debug
    Force/cancel the debug mode.
    If not present, use the debug value in the configuration.
  --log (<filepath>|stdout|stderr)
    Force to write the log in the specified path (or standard output/error).
    If not present, use the log value in the configuration.

Available Routes:

  /flag/list
    search=<pkgname pattern>
    repo=<repository>
    email=<email of the submitter>
    from=<minimum date of submission>
    to=<maximum date of submission>
    sortby=(repo|name|date)
    sortdir=(asc|desc) (default: asc)
    page=<page number to display>
    limit=<max number of result> (default: defined in configuration, parameter pagination of section [api])

  /flag/add
    name=<pkgname>
    version=<pkgver>
    repo=<repository>
    email=<email of submitter>
    comment=<comment of submitter>

  /flag/delete (INNER USE ONLY!)
    ids=<list of flag IDs separated by comma>

  /package/view
    name=<repo/pkgname-pkgver>

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

  /update/mirror (INNER USE ONLY!)

  /update/repo (INNER USE ONLY!)

  /update/all (INNER USE ONLY!)
`

func init() {
	if len(os.Args) > 2 {
		e, args := "", os.Args[2:]
		for len(args) > 0 {
			e, args = args[0], args[1:]
			switch e {
			case "--debug":
				log.Debug = true
			case "--no-debug":
				log.Debug = false
			case "--log":
				if len(args) > 0 {
					e, args = args[0], args[1:]
					log.Init(e)
				} else {
					printUsage()
					os.Exit(1)
				}
			default:
				printUsage()
				os.Exit(1)
			}
		}
	}
}

func printUsage() {
	w := os.Stderr

	fmt.Fprintf(w, "\033[1;31mUsage: %s (subcommand) [args...]\033[m\n", os.Args[0])
	fmt.Fprint(w, help)
}

func main() {
	if len(os.Args) > 1 {
		action, ok := actions[os.Args[1]]
		for i, arg := range os.Args[2:] {
			switch arg {
			case "--debug":
				log.Debug = true
			case "--no-debug":
				log.Debug = false
			case "--log":
				if i < len(os.Args)-1 {
					log.Init(os.Args[i+1])
				}
			}
		}
		if ok {
			action()
			return
		}
	}

	printUsage()
	os.Exit(1)
}
