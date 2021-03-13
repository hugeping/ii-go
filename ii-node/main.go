package main

import (
	"../ii"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func open_db(path string) *ii.DB {
	db := ii.OpenDB(path)
	if db == nil {
		ii.Error.Printf("Can no open db: %s\n", path)
		os.Exit(1)
	}
	return db
}

func PointMsg(edb *ii.EDB, db *ii.DB, udb *ii.UDB, pauth string, tmsg string) string {
	udb.LoadUsers()

	if !udb.Access(pauth) {
		ii.Info.Printf("Access denied for pauth: %s", pauth)
		return "Access denied"
	}
	m, err := ii.DecodeMsgline(tmsg, true)
	if err != nil {
		ii.Error.Printf("Receive point msg: %s", err)
		return fmt.Sprintf("%s", err)
	}
	if r, _ := m.Tag("repto"); r != "" {
		if db.Lookup(r) == nil {
			ii.Error.Printf("Receive point msg with wrong repto.")
			return fmt.Sprintf("Receive point msg with wrong repto.")
		}
	}
	if !edb.Allowed(m.Echo) {
		ii.Error.Printf("This echo is disallowed")
		return fmt.Sprintf("This echo is disallowed")
	}

	m.From = udb.Name(pauth)
	m.Addr = fmt.Sprintf("%s,%d", db.Name, udb.Id(pauth))
	if err := db.Store(m); err != nil {
		ii.Error.Printf("Store point msg: %s", err)
		return fmt.Sprintf("%s", err)
	}
	return "msg ok"
}

var users_opt *string = flag.String("u", "points.txt", "Users database")
var db_opt *string = flag.String("db", "./db", "II database path (directory)")
var listen_opt *string = flag.String("L", ":8080", "Listen address")
var sysname_opt *string = flag.String("sys", "ii-go", "Node name")
var host_opt *string = flag.String("host", "http://127.0.0.1:8080", "Node address")
var verbose_opt *bool = flag.Bool("v", false, "Verbose")
var echo_opt *string = flag.String("e", "list.txt", "Echoes list")

type WWW struct {
	Host string
	tpl  *template.Template
	db   *ii.DB
	edb  *ii.EDB
	udb  *ii.UDB
}

func get_ue(echoes []string, db *ii.DB, user ii.User, w http.ResponseWriter, r *http.Request) {
	if len(echoes) == 0 {
		return
	}
	slice := echoes[len(echoes)-1:][0]
	var idx, lim int
	if _, err := fmt.Sscanf(slice, "%d:%d", &idx, &lim); err == nil {
		echoes = echoes[:len(echoes)-1]
	} else {
		idx, lim = 0, 0
	}

	for _, e := range echoes {
		if !ii.IsEcho(e) {
			continue
		}
		fmt.Fprintf(w, "%s\n", e)
		ids := db.SelectIDS(ii.Query{Echo: e, Start: idx, Lim: lim, User: user})
		for _, id := range ids {
			fmt.Fprintf(w, "%s\n", id)
		}
	}

}
func main() {
	var www WWW
	ii.OpenLog(ioutil.Discard, os.Stdout, os.Stderr)

	flag.Parse()

	db := open_db(*db_opt)
	edb := ii.LoadEcholist(*echo_opt)
	udb := ii.OpenUsers(*users_opt)
	if *verbose_opt {
		ii.OpenLog(os.Stdout, os.Stdout, os.Stderr)
	}

	db.Name = *sysname_opt
	www.db = db
	www.edb = edb
	www.udb = udb
	www.Host = *host_opt
	WebInit(&www)

	fs := http.FileServer(http.Dir("lib"))
	http.Handle("/lib/", http.StripPrefix("/lib/", fs))

	http.HandleFunc("/list.txt", func(w http.ResponseWriter, r *http.Request) {
		echoes := db.Echoes(nil, ii.Query{})
		for _, v := range echoes {
			if !ii.IsPrivate(v.Name) {
				fmt.Fprintf(w, "%s:%d:%s\n", v.Name, v.Count, www.edb.Info[v.Name])
			}
		}
	})
	http.HandleFunc("/blacklist.txt", func(w http.ResponseWriter, r *http.Request) {
		ids := db.SelectIDS(ii.Query{Blacklisted: true})
		for _, v := range ids {
			fmt.Fprintf(w, "%s\n", v)
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleWWW(&www, w, r)
	})
	http.HandleFunc("/u/point/", func(w http.ResponseWriter, r *http.Request) {
		var pauth, tmsg string
		switch r.Method {
		case "GET":
			udb.LoadUsers()
			args := strings.Split(r.URL.Path[9:], "/")

			if len(args) >= 3 && args[1] == "u" {
				pauth = args[0]
				if !udb.Access(pauth) {
					ii.Info.Printf("Access denied for pauth: %s", pauth)
					return
				}
				user := udb.UserInfo(pauth)
				if user == nil {
					return
				}
				if args[2] == "e" {
					echoes := args[3:]
					get_ue(echoes, db, *user, w, r)
					return
				}
				if args[2] == "m" {
					ids := args[3:]
					for _, i := range ids {
						m, info := db.GetBundleInfo(i)
						if m == "" || !db.Access(info, user) {
							continue
						}
						fmt.Fprintf(w, "%s\n", m)
					}
					return
				}
				ii.Error.Printf("Wrong /u/point/ get request: %s", r.URL.Path[9:])
				return
			}
			if len(args) != 2 {
				ii.Error.Printf("Wrong /u/point/ get request: %s", r.URL.Path[9:])
				return
			}
			pauth, tmsg = args[0], args[1]
		default:
			return
		}
		ii.Info.Printf("/u/point/%s/%s GET request", pauth, tmsg)
		fmt.Fprintf(w, PointMsg(edb, db, udb, pauth, tmsg))
	})
	http.HandleFunc("/u/point", func(w http.ResponseWriter, r *http.Request) {
		var pauth, tmsg string
		switch r.Method {
		case "POST":
			if err := r.ParseForm(); err != nil {
				ii.Error.Printf("Error in POST request: %s", err)
				return
			}
			pauth = r.FormValue("pauth")
			tmsg = r.FormValue("tmsg")
		default:
			return
		}
		ii.Info.Printf("/u/point/%s/%s POST request", pauth, tmsg)
		fmt.Fprintf(w, PointMsg(edb, db, udb, pauth, tmsg))
	})
	http.HandleFunc("/x/c/", func(w http.ResponseWriter, r *http.Request) {
		enames := strings.Split(r.URL.Path[5:], "/")
		echoes := db.Echoes(enames, ii.Query{})
		for _, v := range echoes {
			if !ii.IsPrivate(v.Name) {
				fmt.Fprintf(w, "%s:%d:\n", v.Name, v.Count)
			}
		}
	})
	http.HandleFunc("/u/m/", func(w http.ResponseWriter, r *http.Request) {
		ids := strings.Split(r.URL.Path[5:], "/")
		for _, i := range ids {
			m, info := db.GetBundleInfo(i)
			if m != "" && !ii.IsPrivate(info.Echo) {
				fmt.Fprintf(w, "%s\n", m)
			}
		}
	})
	http.HandleFunc("/u/e/", func(w http.ResponseWriter, r *http.Request) {
		echoes := strings.Split(r.URL.Path[5:], "/")
		get_ue(echoes, db, ii.User{}, w, r)
	})
	http.HandleFunc("/m/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[3:]
		if !ii.IsMsgId(id) {
			return
		}
		m := db.Get(id)
		ii.Info.Printf("/m/%s %s", id, m)
		if m != nil && !ii.IsPrivate(m.Echo) {
			fmt.Fprintf(w, "%s", m.String())
		}
	})
	http.HandleFunc("/e/", func(w http.ResponseWriter, r *http.Request) {
		e := r.URL.Path[3:]
		if !ii.IsEcho(e) || ii.IsPrivate(e) {
			return
		}
		ids := db.SelectIDS(ii.Query{Echo: e})
		for _, id := range ids {
			fmt.Fprintf(w, "%s\n", id)
		}
	})
	http.HandleFunc("/x/features", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "list.txt\nblacklist.txt\nu/e\nx/c\n")
	})
	ii.Info.Printf("Listening on %s", *listen_opt)

//	http.HandleFunc("hugeping.ru/", func(w http.ResponseWriter, r *http.Request) {
//		http.Redirect(w, r, "//club.hugeping.ru/blog/std.hugeping", http.StatusSeeOther)
//	})

	http.Handle("hugeping.ru/", http.FileServer(http.Dir("/home/pi/Devel/gemini/www")))
	http.Handle("syscall.ru/", http.FileServer(http.Dir("/home/pi/Devel/gemini/www")))

	if err := http.ListenAndServe(*listen_opt, nil); err != nil {
		ii.Error.Printf("Error running web server: %s", err)
	}
}
