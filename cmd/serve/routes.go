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
		if pagination, ok := database.Paginate(&flags, q); ok {
			writeResponse(r, w, conv.Map{
				"data":     flags,
				"filter":   mf,
				"sort":     ms,
				"paginate": pagination,
			})
		} else {
			writeResponse(r, w, conv.Map{"data": nil}, http.StatusInternalServerError)
		}
	},
	"/flag/add": func(w http.ResponseWriter, r *http.Request) {
		email, err := mail.ParseAddress(getString(r, "email"))
		if err != nil {
			writeResponse(r, w, conv.Map{"data": nil}, http.StatusInternalServerError)
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
		code := http.StatusOK
		if database.First(&p, q) {
			p.Flag = f
			if err := database.CreateFlag(&p); err == nil {
				sendMail(p)
			} else {
				log.Debugf("Failed to create flag: %s\n", err)
			}
		} else {
			f = database.Flag{}
			code = http.StatusInternalServerError
		}
		writeResponse(r, w, conv.Map{
			"data": f,
		}, code)
	},
	"/mirror": func(w http.ResponseWriter, r *http.Request) {
		var countries []database.Country
		database.SearchAll(&countries, "Mirrors.Repos")
		writeResponse(r, w, countries)
	},
	"/update/mirror": func(w http.ResponseWriter, r *http.Request) {
		data := database.UpdateMirrors(
			conf.String("mirror.pacmanconf"),
			conf.String("mirror.mirrorlist"),
			conf.String("mirror.main_mirror"),
		)
		writeResponse(r, w, data)
	},
	"/update/repo": func(w http.ResponseWriter, r *http.Request) {
		data := database.UpdatePackages(
			conf.String("repository.basedir"),
			conf.String("repository.extension"),
			conf.Slice("repository.include"),
			conf.Slice("repository.exclude"),
		)
		writeResponse(r, w, data)
	},
	"/update/all": func(w http.ResponseWriter, r *http.Request) {
		data := database.UpdateAll(
			conf.String("mirror.pacmanconf"),
			conf.String("mirror.mirrorlist"),
			conf.String("mirror.main_mirror"),
			conf.String("repository.basedir"),
			conf.String("repository.extension"),
			conf.Slice("repository.include"),
			conf.Slice("repository.exclude"),
		)
		writeResponse(r, w, data)
	},
	"/package/view": func(w http.ResponseWriter, r *http.Request) {
		name := getString(r, "name")
		if name == "" {
			writeResponse(r, w, conv.Map{"data": nil}, http.StatusNotFound)
			return
		}
		var p database.Package
		if !database.GetPackage(
			&p,
			database.NewFilterRequest(
				database.NewFilter("repository||'/'||name||-||version", "=", name),
			),
			conf.String("repository.base"),
		) {
			writeResponse(r, w, conv.Map{"data": nil}, http.StatusNotFound)
			return
		}
		data := conv.Map{
			"Repository":    p.Repository,
			"Name":          p.Name,
			"Version":       p.Version,
			"Arch":          p.Arch,
			"Description":   p.Description,
			"PackageSize":   conv.ToSize(p.PackageSize),
			"InstalledSize": conv.ToSize(p.InstalledSize),
			"Licenses":      p.Licenses,
			"Groups":        p.Groups,
			"BuildDate":     p.BuildDate,
			"Depends":       p.Depends,
			"Files":         p.Files,
			"Md5Sum":        p.Md5Sum,
			"Sha256Sum":     p.Sha256Sum,
			"Filename":      p.Filename,
			"Flagged":       p.FlagID != 0,
			"CompleteName":  p.VersionName(),
		}
		if p.BuildVersion != nil {
			data["Build"] = p.BuildVersion.FullName()
		}
		url := conv.Map{
			"Upstream": p.URL,
			"Download": conf.String("main.repourl") + p.Repository + "/" + p.Filename,
		}
		if p.GitID != 0 {
			g := p.Git
			gurl := conf.String("main.giturl") + p.Repository
			url["Bugs"] = gurl + "/issues/"
			url["Sources"] = gurl + "/tree/master/" + g.Folder
			url["PKGBUILD"] = gurl + "/blob/master/" + g.Folder + "/PKGBUILD"
			url["Commits"] = gurl + "/commits/master/" + g.Folder + "/PKGBUILD"
		}
		data["URL"] = url
		writeResponse(r, w, conv.Map{"data": data})
	},
	"/package/list": func(w http.ResponseWriter, r *http.Request) {
		getPackages(w, r, "")
	},
	"/repo/list": func(w http.ResponseWriter, r *http.Request) {
		repos := conf.Slice("repository.include")
		repo := getString(r, "repo")
		for _, e := range repos {
			if e == repo {
				getPackages(w, r, repo)
				return
			}
		}
		writeResponse(r, w, conv.Map{"data": nil}, http.StatusNotFound)
	},
}
