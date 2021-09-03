package serve

import (
	"fmt"
	"net/http"
	"pmanager/conf"
	"pmanager/database"
	"pmanager/log"
	"pmanager/util/conv"
	"pmanager/util/mail"
	"strings"
	"time"
)

func getMailSubjectAndBody(p database.Package, cr string) (subject, body string) {
	pname := p.FullName()
	comment := p.Flag.Comment
	if cr != "\n" {
		comment = strings.ReplaceAll(comment, "\n", cr)
	}
	bodyLines := []string{
		fmt.Sprintf("Package details: %s/view.php?name=%s", conf.String("main.viewurl"), pname),
		"",
		"",
		"---",
		fmt.Sprintf("The package %s has been flagged as outdated.", pname),
		"by: " + p.Flag.Email,
		"",
		"Additional informations:",
		comment,
	}
	subject = fmt.Sprintf("The package %s has been flagged as outdated", pname)
	body = strings.Join(bodyLines, cr)
	return
}

func sendMail(p database.Package) {
	subject, body := getMailSubjectAndBody(p, "\r\n")
	var m mail.Mail
	m.From(conf.String("smtp.send_from")).
		To(conf.String("smtp.send_to")).
		Header("Reply-To", conf.String("smtp.send_to")).
		Header("X-Mailer", "Packages").
		Header("MIME-Version", "1.0").
		Header("Content-Transfer-Encoding", "8bit").
		Header("Content-type", "text/plain; charset=utf-8").
		Subject(subject).
		Body(body)

	if err := mail.Send(m); err != nil {
		log.Errorf("Failed to send mail: %s\n", err)
	}
}

func debugRequest(r *http.Request, code int) {
	log.Debugf("%s %s (%d) %s %s\n", r.Method, r.RequestURI, code, r.RemoteAddr, r.Header.Get("user-agent"))
}

func writeResponse(r *http.Request, w http.ResponseWriter, data interface{}, codes ...int) {
	code := http.StatusOK
	if len(codes) == 1 {
		code = codes[0]
	}
	debugRequest(r, code)
	w.WriteHeader(code)
	b := conv.ToJson(data, log.Debug)
	if _, err := w.Write(b); err == nil {
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Add("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")
	} else {
		log.Debugf("Response error: %s\n", err)
	}
}

func getString(r *http.Request, key string) string  { return r.FormValue(key) }
func getInt(r *http.Request, key string) int64      { return conv.String2Int(getString(r, key)) }
func getBool(r *http.Request, key string) bool      { return conv.String2Bool(getString(r, key)) }
func getDate(r *http.Request, key string) time.Time { return conv.String2Date(getString(r, key)) }

func initPaginationQuery(r *http.Request) *database.Request {
	page := getInt(r, "page")
	if page <= 0 {
		page = 1
	}
	limit := getInt(r, "limit")
	if limit <= 0 {
		limit = defaultPagination
	}
	return (new(database.Request)).SetLimit(limit).SetPage(page)
}

func like(str string) string {
	return "%" + str + "%"
}

func getFilter(r *http.Request, fields ...string) conv.Map {
	m := make(conv.Map)
	for _, f := range fields {
		t := "s"
		if i := strings.Index(f, "|"); i >= 0 {
			f, t = f[:i], f[i+1:]
		}
		if getString(r, f) == "" {
			continue
		}
		switch t {
		case "b":
			m[f] = getBool(r, f)
		case "d":
			if e := getDate(r, f); !e.IsZero() {
				m[f] = e
			}
		case "i":
			m[f] = getInt(r, f)
		default:
			m[f] = getString(r, f)
		}
	}
	return m
}

func getSort(r *http.Request, authorized_fields ...string) conv.Map {
	field := getString(r, "sortby")
	for _, f := range authorized_fields {
		if field == f {
			return conv.Map{
				"field": f,
				"asc":   getString(r, "sortdir") != "desc",
			}
		}
	}
	return nil
}

func getPackages(w http.ResponseWriter, r *http.Request, repository string) {
	q := initPaginationQuery(r)
	ms := getSort(r, "name", "repo", "date", "flagged")
	mf := getFilter(r, "search", "from|d", "to|d", "flagged|b", "exact|b")
	if repository != "" {
		mf["repo"] = repository
	}

	if mf.Exists("search") {
		v, op, f := mf.GetString("search"), "=", "name"
		if !mf.GetBool("exact") {
			v, op, f = like(v), "LIKE", "name||'-'||version"
		}
		q.AddFilter(f, op, v)
	}
	if mf.Exists("from") {
		q.AddFilter("build_date", ">=", mf.GetDate("from"))
	}
	if mf.Exists("to") {
		q.AddFilter("build_date", "<=", mf.GetDate("to"))
	}
	if mf.Exists("repo") {
		q.AddFilter("repository", "=", repository)
	}
	if mf.Exists("flagged") {
		op := "="
		if mf.GetBool("flagged") {
			op = "<>"
		}
		q.AddFilter("flag_id", op, 0)
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
			q.AddSort("build_date", desc)
		case "flagged":
			q.AddSort("CASE flag_id WHEN 0 then 0 ELSE 1 END", desc)
		}
	}

	var packages []database.Package
	pagination, ok := database.Paginate(&packages, q)
	if !ok {
		writeResponse(r, w, conv.Map{"data": nil}, http.StatusInternalServerError)
		return
	}

	totalSize := int64(0)
	if pagination.Total > 0 {
		totalSize = database.SumSizes(q, "package_size")
	}
	data := make([]conv.Map, len(packages))
	for i, p := range packages {
		data[i] = conv.Map{
			"Repository":    p.Repository,
			"Name":          p.Name,
			"Version":       p.Version,
			"Arch":          p.Arch,
			"Description":   p.Description,
			"PackageSize":   conv.ToSize(p.PackageSize),
			"InstalledSize": conv.ToSize(p.InstalledSize),
			"Md5Sum":        p.Md5Sum,
			"Sha256Sum":     p.Sha256Sum,
			"Filename":      p.Filename,
			"BuildDate":     p.BuildDate,
			"Flagged":       p.FlagID != 0,
			"CompleteName":  p.VersionName(),
			"FullName":      p.FullName(),
		}
	}

	writeResponse(r, w, conv.Map{
		"data":     data,
		"filter":   mf,
		"sort":     ms,
		"paginate": pagination,
		"size":     conv.ToSize(totalSize),
	})
}
