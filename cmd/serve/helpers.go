package serve

import (
	"fmt"
	"net/http"
	"pmanager/database"
	"pmanager/log"
	"strings"
	"time"

	"pmanager/conf.new"
	"pmanager/util.new/conv"
	"pmanager/util.new/mail"
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
		limit = conf.Int("api.pagination")
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
