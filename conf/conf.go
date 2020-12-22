package conf

import (
	"bufio"
	"os"
	"path"
	"strings"
)

var (
	Args []string
)

func init() {
	Load()
	Args = make([]string, 1, len(os.Args))
	Args[0] = os.Args[0]
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--debug":
			config["main.debug"] = true
		case "--no-debug":
			config["main.debug"] = false
		default:
			Args = append(Args, arg)
		}
	}
}

const (
	BASEDIR  = "/etc/pmanager"
	CONFFILE = "pmanager.conf"
)

var config = Map{
	"main.viewurl":         "https://kaosx.us/packages",
	"main.repourl":         "http://kaosx.tk/repo/",
	"main.giturl":          "https://github.com/KaOSx/",
	"main.debug":           true,
	"main.basedir":         BASEDIR,
	"database.subdir":      "db",
	"database.extension":   "json",
	"repository.basedir":   "/var/www/html/repo",
	"repository.exclude":   []string{"ISO", "kde-next"},
	"repository.extension": "files.tar.gz",
	"api.port":             "9000",
	"api.pagination":       int64(20),
	"smtp.host":            "smtp.example.com",
	"smtp.port":            "465",
	"smtp.use_encryption":  true,
	"smtp.user":            "user@example.com",
	"smtp.password":        "my_password",
	"smtp.send_to":         "user@example.com",
	"smtp.send_from":       "user@example.com",
	"smtp.use_formspree":   false,
	"mirror.main_mirror":   "http://kaosx.tk/repo/",
	"mirror.mirrorlist":    "/etc/pacman.d/mirrorlist",
	"mirror.pacmanconf":    "/etc/pacman.conf",
}

func Load() {
	f, err := os.Open(path.Join(BASEDIR, CONFFILE))
	if err != nil {
		return
	}
	defer f.Close()
	buf := bufio.NewScanner(f)
	var section string
	for buf.Scan() {
		line := strings.TrimSpace(buf.Text())
		l := len(line)
		// Line is comment or blank line
		if l == 0 || line[0] == '#' || line[0] == ';' {
			continue
		}
		// line is section header
		if line[0] == '[' && line[l-1] == ']' {
			section = line[1 : l-1]
			continue
		}
		if i := strings.Index(line, "="); i > 0 && i < l-1 {
			key, value := strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
			if value != "" {
				config[section+"."+key] = value
			}
		}
	}
}

func Read(key string) string { return config.GetString(key) }

func ReadBool(key string) bool { return config.GetBool(key) }

func ReadInt(key string) int64 { return config.GetInt(key) }

func ReadArray(key string) []string { return config.GetSlice(key) }

func Debug() bool { return ReadBool("main.debug") }

func Basedir() string { return Read("main.basedir") }
