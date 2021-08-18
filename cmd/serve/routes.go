package serve

import (
	"html"
	"net/http"
	"net/mail"
	"pmanager/database"
	"pmanager/log"

	"pmanager/conf.new"
	"pmanager/util.new/conv"
)

var routes = map[string]func(http.ResponseWriter, *http.Request){
	"/flag/list": func(w http.ResponseWriter, r *http.Request) {
		q := initPaginationQuery(r)
		mf := getFilter(r, "search", "repo", "email", "from|d", "to|d")
		ms := getSort(r, "name", "repo", "date")

		if mf.Exists("search") {
			q.AddFilter("name", "LIKE", like(mf.GetString("search")))
		}
		if mf.Exists("repo") {
			q.AddFilter("repository", "=", mf.GetString("repo"))
		}
		if mf.Exists("email") {
			q.AddFilter("email", "LIKE", like(mf.GetString("email")))
		}
		if mf.Exists("from") {
			q.AddFilter("created_at", ">=", mf.GetDate("from"))
		}
		if mf.Exists("to") {
			q.AddFilter("created_at", "<=", mf.GetDate("to"))
		}

		if ms != nil {
			field := ms.GetString("field")
			desc := !ms.GetBool("asc")
			switch field {
			case "name":
				q.AddSort("name", desc).AddSort("version", desc)
			case "repo":
				q.AddSort("repository", desc)
			case "date":
				q.AddSort("created_at", desc)
			}
		}

		var flags []database.Flag
		pagination, _ := database.Paginate(&flags, q)

		writeResponse(r, w, conv.Map{
			"data":     flags,
			"filter":   mf,
			"sort":     ms,
			"paginate": pagination,
		})
	},
	"/flag/add": func(w http.ResponseWriter, r *http.Request) {
		email, err := mail.ParseAddress(getString(r, "email"))
		if err != nil {
			writeResponse(r, w, conv.Map{
				"data": database.Flag{},
			})
			return
		}
		f := database.Flag{
			Name:       getString(r, "name"),
			Version:    getString(r, "version"),
			Repository: getString(r, "repo"),
			Email:      email.Address,
			Comment:    html.EscapeString(getString(r, "comment")),
		}
		var p database.Package
		q := (new(database.Request)).
			AddFilter("name", "=", f.Name).
			AddFilter("version", "=", f.Version).
			AddFilter("Repository", "=", f.Repository)
		if database.First(&p, q) {
			p.Flag = f
			if err := database.CreateFlag(&p); err == nil {
				sendMail(p)
			} else {
				log.Debugf("Failed to create flag: %s\n", err)
			}
		} else {
			f = database.Flag{}
		}
		writeResponse(r, w, conv.Map{
			"data": f,
		})
	},
	"/mirror": func(w http.ResponseWriter, r *http.Request) {
		var countries []database.Country
		database.SearchAll(&countries)
		writeResponse(r, w, countries)
	},
	"/update/mirror": func(w http.ResponseWriter, r *http.Request) {
		data := database.UpdateMirrors(
			conf.String("mirror.pacmanconf"),
			conf.String("mirror.pacmanmirror"),
			conf.String("mirror.main_mirror"),
		)
		writeResponse(r, w, data)
	},
	"/update/repo": func(w http.ResponseWriter, r *http.Request) {
		data := database.UpdatePackages(
			conf.String("repository.base"),
			conf.String("repository.extension"),
			conf.Slice("repository.exclude"),
		)
		writeResponse(r, w, data)
	},
	"/update/all": func(w http.ResponseWriter, r *http.Request) {
		data := database.UpdateAll(
			conf.String("mirror.pacmanconf"),
			conf.String("mirror.pacmanmirror"),
			conf.String("mirror.main_mirror"),
			conf.String("repository.base"),
			conf.String("repository.extension"),
			conf.Slice("repository.exclude"),
		)
		writeResponse(r, w, data)
	},
}
