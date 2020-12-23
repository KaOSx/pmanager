package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"pmanager/conf"
	"pmanager/db"
	"pmanager/util"
	"pmanager/util/mail"
	"strings"
	"time"
)

func getMailSubjectAndBody(f db.Flag, cr string) (subject, body string) {
	pname := f.CompleteName()
	comment := f.Comment
	if cr != "\n" {
		comment = strings.ReplaceAll(comment, "\n", cr)
	}
	bodyLines := []string{
		fmt.Sprintf("Package details: %s/view.php?repo=%s&name=%s", conf.Read("main.viewurl"), f.Repository, pname),
		"",
		"",
		"---",
		fmt.Sprintf("The package %s has been flagged as outdated.", pname),
		"by: " + f.Email,
		"",
		"Additional informations:",
		comment,
	}
	subject = fmt.Sprintf("The package %s has been flagged as outdated", pname)
	body = strings.Join(bodyLines, cr)
	return
}

func sendFormSpree(f db.Flag) {
	subject, body := getMailSubjectAndBody(f, "\n")
	data := new(bytes.Buffer)
	encoder := json.NewEncoder(data)
	encoder.Encode(map[string]interface{}{
		"email":    f.Email,
		"_subject": subject,
		"message":  body,
	})
	var client http.Client
	request, err := http.NewRequest("POST", "https://formspree.io/"+conf.Read("smtp.send_to"), data)
	if err != nil {
		util.Debugln("Request error:", err)
		return
	}
	request.Header.Add("Referer", conf.Read("main.viewurl"))
	request.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(request)
	defer resp.Body.Close()
	if err != nil {
		util.Println("Failed to send formspree mail:", err)
	}
}

func sendMail(f db.Flag) {
	server := mail.Server{
		Host:     conf.Read("smtp.host"),
		Port:     conf.Read("smtp.port"),
		TLS:      conf.ReadBool("smtp.use_encryption"),
		User:     conf.Read("smtp.user"),
		Password: conf.Read("smtp.password"),
	}
	subject, body := getMailSubjectAndBody(f, "\r\n")
	m := mail.New(server, conf.Read("smtp.send_from"), conf.Read("smtp.send_to"), subject, body)
	m.AddHeader("Reply-To", conf.Read("smtp.send_to")).
		AddHeader("X-Mailer", "Packages").
		AddHeader("MIME-Version", "1.0").
		AddHeader("Content-Transfer-Encoding", "8bit").
		AddHeader("Content-type", "text/plain; charset=utf-8")
	if err := m.Send(); err != nil {
		util.Println("Failed to send mail:", err)
	}
}

func readPkginfo(sc *bufio.Scanner, g *db.Git) {
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "gitrepo = ") {
			g.Repository = strings.TrimSpace(line[len("gitrepo = "):])
		} else if strings.HasPrefix(line, "gitfolder = ") {
			g.Folder = strings.TrimSpace(line[len("gitfolder = "):])
		}
	}
}

func getGit(p db.Package, git *db.Git) (ok bool) {
	gpath := path.Join(conf.Read("repository.basedir"), p.Repository, p.FileName())
	f, err := os.Open(gpath)
	if err != nil {
		util.Debugf("\033[1;31mFailed to load %s: %s\033[m\n", gpath, err)
		return
	}
	defer f.Close()
	var xf io.ReadCloser
	if strings.HasSuffix(gpath, ".xz") {
		xf, err = util.ReadXZ(f)
		if err != nil {
			util.Debugf("\033[1;31mFailed to xunzip %s: %s\033[m\n", gpath, err)
		}
	} else {
		xf, err = util.ReadZST(f)
		if err != nil {
			util.Debugf("\033[1;31mFailed to zstd -d %s: %s\033[m\n", gpath, err)
		}
	}
	if err != nil {
		return
	}
	defer xf.Close()
	tf := util.ReadTAR(xf)
	for {
		hdr, err := tf.Next()
		if err != nil {
			break
		}
		if hdr.Name == ".PKGINFO" {
			var buf bytes.Buffer
			if _, err = io.Copy(&buf, tf); err == nil {
				sc := bufio.NewScanner(&buf)
				readPkginfo(sc, git)
				ok = true
			}
			break
		}
	}
	return
}

func writeResponse(w http.ResponseWriter, data interface{}, codes ...int) {
	code := http.StatusOK
	if len(codes) == 1 {
		code = codes[0]
	}
	w.WriteHeader(code)
	b := util.JSON(data)
	if _, err := w.Write(b); err == nil {
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Add("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")
	} else {
		util.Debugln("Response error:", err)
	}
}

func getString(r *http.Request, key string) string  { return r.FormValue(key) }
func getInt(r *http.Request, key string) int64      { return conf.String2Int(getString(r, key)) }
func getBool(r *http.Request, key string) bool      { return conf.String2Bool(getString(r, key)) }
func getDate(r *http.Request, key string) time.Time { return conf.String2Date(getString(r, key)) }

func paginate(r *http.Request) (pagination db.Pagination) {
	page := getInt(r, "page")
	if page <= 0 {
		page = 1
	}
	limit := getInt(r, "limit")
	if limit <= 0 {
		limit = conf.ReadInt("api.pagination")
	}
	return db.NewPagination(limit, page)
}

func sort(r *http.Request) (sorted conf.Map) {
	sorted = make(conf.Map)
	sorted["field"] = getString(r, "sortby")
	sorted["asc"] = getString(r, "sortdir") != "desc"
	return sorted
}

func flagList(w http.ResponseWriter, r *http.Request) {
	pagination := paginate(r)
	var search db.Request

	mfilter := make(conf.Map)
	var filters []db.MatchFunc
	if e := getString(r, "search"); e != "" {
		mfilter["search"] = e
		filters = append(filters, db.FlagFilter2MatchFunc(db.SearchFlagByName(e)))
	}
	if e := getString(r, "repo"); e != "" {
		mfilter["repo"] = e
		filters = append(filters, db.FlagFilter2MatchFunc(db.SearchFlagByRepo(e)))
	}
	if e := getString(r, "email"); e != "" {
		mfilter["email"] = e
		filters = append(filters, db.FlagFilter2MatchFunc(db.SearchFlagByEmail(e)))
	}
	if e := getDate(r, "from"); !e.IsZero() {
		mfilter["from"] = e
		filters = append(filters, db.FlagFilter2MatchFunc(db.SearchFlagByDateFrom(e)))
	}
	if e := getDate(r, "to"); !e.IsZero() {
		mfilter["to"] = e
		filters = append(filters, db.FlagFilter2MatchFunc(db.SearchFlagByDateTo(e)))
	}
	if getString(r, "flagged") != "" {
		e := getBool(r, "flagged")
		mfilter["flagged"] = e
		filters = append(filters, db.FlagFilter2MatchFunc(db.SearchFlagByFlagged(e)))
	}

	msort := sort(r)
	var sorts []db.CmpFunc
	d2f := func(e1, e2 db.Data) (f1, f2 db.Flag) {
		f1, _ = e1.(db.Flag)
		f2, _ = e2.(db.Flag)
		return
	}
	switch msort["field"] {
	case "name":
		sorts = append(sorts, func(e1, e2 db.Data) int {
			f1, f2 := d2f(e1, e2)
			return util.CompareString(f1.CompleteName(), f2.CompleteName())
		})
	case "repo":
		sorts = append(sorts, func(e1, e2 db.Data) int {
			f1, f2 := d2f(e1, e2)
			return util.CompareString(f1.Repository, f2.Repository)
		})
	case "date":
		sorts = append(sorts, func(e1, e2 db.Data) int {
			f1, f2 := d2f(e1, e2)
			return util.CompareDate(f1.Date, f2.Date)
		})
	case "flagged":
		sorts = append(sorts, func(e1, e2 db.Data) int {
			f1, f2 := d2f(e1, e2)
			return util.CompareBool(f1.Flagged, f2.Flagged)
		})
	}

	search = search.
		SetFilter(filters...).
		SetSort(sorts...)
	if !msort.GetBool("asc") {
		search = search.ReverseSort()
	}

	var flags []db.Flag
	db.Paginate("flag", &flags, search, pagination)

	writeResponse(w, conf.Map{
		"data":     flags,
		"filter":   mfilter,
		"sort":     msort,
		"paginate": pagination,
	})
}

func flagAdd(w http.ResponseWriter, r *http.Request) {
	f := db.Flag{
		Name:       getString(r, "name"),
		Version:    getString(r, "version"),
		Repository: getString(r, "repo"),
		Email:      getString(r, "email"),
		Comment:    getString(r, "comment"),
		Flagged:    true,
		Date:       time.Now(),
	}
	db.Add("flag", f, true)
	if m := strings.ToUpper(r.Method); m == "GET" || m == "POST" {
		if conf.ReadBool("smtp.use_formspree") {
			sendFormSpree(f)
		} else {
			sendMail(f)
		}
	}
	writeResponse(w, conf.Map{
		"data": f,
	})
}

func searchPackages(w http.ResponseWriter, r *http.Request, table string) {
	pagination := paginate(r)
	var search db.Request

	mfilter := make(conf.Map)
	var filters []db.MatchFunc
	if e := getString(r, "search"); e != "" {
		exact := getBool(r, "exact")
		mfilter["search"] = e
		mfilter["exact"] = exact
		filters = append(filters, db.PackageFilter2MatchFunc(db.SearchPackageByName(e, exact)))
	}
	if e := getDate(r, "from"); !e.IsZero() {
		mfilter["from"] = e
		filters = append(filters, db.PackageFilter2MatchFunc(db.SearchPackageByDateFrom(e)))
	}
	if e := getDate(r, "to"); !e.IsZero() {
		mfilter["to"] = e
		filters = append(filters, db.PackageFilter2MatchFunc(db.SearchPackageByDateTo(e)))
	}
	if getString(r, "flagged") != "" {
		e := getBool(r, "flagged")
		mfilter["flagged"] = e
		filters = append(filters, db.PackageFilter2MatchFunc(db.SearchPackageByFlagged(e)))
	}

	msort := sort(r)
	var sorts []db.CmpFunc
	d2p := func(e1, e2 db.Data) (p1, p2 db.Package) {
		p1, _ = e1.(db.Package)
		p2, _ = e2.(db.Package)
		return
	}
	switch msort["field"] {
	case "name":
		sorts = append(sorts, func(e1, e2 db.Data) int {
			p1, p2 := d2p(e1, e2)
			return util.CompareString(p1.CompleteName(), p2.CompleteName())
		})
	case "repo":
		if table == "package" {
			sorts = append(sorts, func(e1, e2 db.Data) int {
				p1, p2 := d2p(e1, e2)
				return util.CompareString(p1.Repository, p2.Repository)
			})
		}
	case "date":
		sorts = append(sorts, func(e1, e2 db.Data) int {
			p1, p2 := d2p(e1, e2)
			return util.CompareDate(p1.BuildDate, p2.BuildDate)
		})
	case "flagged":
		sorts = append(sorts, func(e1, e2 db.Data) int {
			p1, p2 := d2p(e1, e2)
			return util.CompareBool(p1.IsFlagged(), p2.IsFlagged())
		})
	}

	search = search.
		SetFilter(filters...).
		SetSort(sorts...)
	if !msort.GetBool("asc") {
		search = search.ReverseSort()
	}

	var packages []db.Package
	db.Paginate(table, &packages, search, pagination)

	data := make([]conf.Map, len(packages))
	var totalSize int64
	for i, p := range packages {
		total_size += p.PackageSize
		m := util.ToMap(p)
		m.Delete("Licenses", "Groups", "Depends", "MakeDepends", "OptDepends", "Files")
		m["PackageSize"] = util.FormatSize(m.GetInt("PackageSize"))
		m["InstalledSize"] = util.FormatSize(m.GetInt("InstalledSize"))
		m["CompleteName"] = p.CompleteName()
		m["RepoName"] = p.RepoName()
		m["FileName"] = p.FileName()
		data[i] = m
	}

	writeResponse(w, conf.Map{
		"data":     data,
		"filter":   mfilter,
		"sort":     msort,
		"paginate": pagination,
		"size":     util.FormatSize(totalSize),
	})
}

func packageList(w http.ResponseWriter, r *http.Request) {
	searchPackages(w, r, "package")
}

func packageView(w http.ResponseWriter, r *http.Request) {
	repo := getString(r, "repo")
	pkgname := getString(r, "name")
	if repo == "" || pkgname == "" {
		writeResponse(w, conf.Map{"data": nil}, http.StatusNotFound)
		return
	}

	var p db.Package
	found := db.Find(
		repo,
		&p,
		db.PackageFilter2MatchFunc(func(p db.Package) bool {
			return p.CompleteName() == pkgname
		}),
	)
	if !found {
		writeResponse(w, conf.Map{"data": nil}, http.StatusNotFound)
		return
	}

	var g db.Git
	if !p.GetGit(&g) {
		if getGit(p, &g) {
			g.Name = p.Name
			db.Add("git", g, true)
		}
	}

	data := util.ToMap(p)
	gurl := conf.Read("main.giturl") + g.Repository
	data["Flagged"] = p.IsFlagged()
	data["PackageSize"] = util.FormatSize(p.PackageSize)
	data["InstalledSize"] = util.FormatSize(p.InstalledSize)
	data["CompleteName"] = p.CompleteName()
	data["RepoName"] = p.RepoName()
	data["FileName"] = p.FileName()
	data["URL"] = conf.Map{
		"Upstream": data["URL"],
		"Download": conf.Read("main.repourl") + p.Repository + "/" + p.FileName(),
		"Bugs":     gurl + "/issues/",
		"Sources":  gurl + "/tree/master/" + g.Folder,
		"PKGBUILD": gurl + "/blob/master/" + g.Folder + "/PKGBUILD",
		"Commits":  gurl + "/commits/master/" + g.Folder + "/PKGBUILD",
	}
	writeResponse(w, conf.Map{"data": data})
}

func repoList(w http.ResponseWriter, r *http.Request) {
	repo := getString(r, "repo")
	if repo == "" || !db.HasTable(repo) {
		writeResponse(w, conf.Map{"data": nil}, http.StatusNotFound)
		return
	}
	searchPackages(w, r, repo)
}

func mirrorList(w http.ResponseWriter, r *http.Request) {
	var countries []db.Country
	db.All("mirror", &countries)
	writeResponse(w, countries)
}

func loadAll(w http.ResponseWriter, r *http.Request) {
	rnames := db.GetRepoNames()
	repos := make([]conf.Map, len(rnames))
	for i, n := range rnames {
		var repo []db.Package
		db.All(n, &repo)
		r := conf.Map{
			"repository_name": n,
			"num_package":     len(repo),
		}
		packages := conf.Map{}
		var lastUpdate time.Time
		for _, p := range repo {
			var g db.Git
			if !p.GetGit(&g) {
				if getGit(p, &g) {
					g.Name = p.Name
					db.Add("git", g, true)
				}
			}
			gurl := conf.Read("main.giturl") + g.Repository
			packages[p.Name] = conf.Map{
				"version":    p.Version,
				"summary":    p.Description,
				"maintainer": "Anke Boersma <demm@kaosx.us>",
				"licenses":   p.Licenses,
				"homepages": conf.Map{
					"upstream": p.URL,
					"bugs":     gurl + "/issues/",
					"sources":  gurl + "/tree/master/" + g.Folder,
					"PKGBUILD": gurl + "/blob/master/" + g.Folder + "/PKGBUILD",
					"commits":  gurl + "/commits/master/" + g.Folder + "/PKGBUILD",
				},
				"download":          conf.Read("main.repourl") + p.Repository + "/" + p.FileName(),
				"categories":        p.Groups,
				"arch":              p.Arch,
				"package_size":      util.FormatSize(p.PackageSize),
				"package_installed": util.FormatSize(p.InstalledSize),
				"depends":           p.Depends,
				"make_depends":      p.MakeDepends,
				"opt_depends":       p.OptDepends,
				"build_date":        p.BuildDate,
			}
			if p.BuildDate.After(lastUpdate) {
				lastUpdate = p.BuildDate
			}
		}
		r["packages"] = packages
		r["last_update"] = lastUpdate
		repos[i] = r
	}
	writeResponse(w, repos)
}

func refresh(w http.ResponseWriter, r *http.Request) {
	name := getString(r, "type")
	if name == "" {
		name = "all"
	}
	db.Load(name, true)
}

func Serve([]string) {
	db.Load("all")
	port := ":" + conf.Read("api.port")
	http.HandleFunc("/flag/list", flagList)
	http.HandleFunc("/flag/add", flagAdd)
	http.HandleFunc("/package/list", packageList)
	http.HandleFunc("/package/view", packageView)
	http.HandleFunc("/repo/list", repoList)
	http.HandleFunc("/mirror", mirrorList)
	http.HandleFunc("/all", loadAll)
	http.HandleFunc("/refresh", refresh)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		util.Fatalln(err)
	}
}
