package main

import (
	"../ii"
	"flag"
	"fmt"
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

func main() {
	ii.OpenLog(ioutil.Discard, os.Stdout, os.Stderr)

	db_opt := flag.String("db", "./db", "II database path (directory)")
	listen_opt := flag.String("L", ":8080", "Listen address")
	db := open_db(*db_opt)
	flag.Parse()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s\n", r.URL.Path)
	})
	http.HandleFunc("/list.txt", func(w http.ResponseWriter, r *http.Request) {
		echoes := db.Echoes(nil)
		for _, v := range echoes {
			fmt.Fprintf(w, "%s:%d:\n", v.Name, v.Count)
		}
	})
	http.HandleFunc("/x/c/", func(w http.ResponseWriter, r *http.Request) {
		enames := strings.Split(r.URL.Path[5:], "/")
		echoes := db.Echoes(enames)
		for _, v := range echoes {
			fmt.Fprintf(w, "%s:%d:\n", v.Name, v.Count)
		}
	})
	http.HandleFunc("/u/m/", func(w http.ResponseWriter, r *http.Request) {
		ids := strings.Split(r.URL.Path[5:], "/")
		for _, i := range ids {
			m := db.GetBundle(i)
			if m != "" {
				fmt.Fprintf(w, "%s\n", m)
			}
		}
	})
	http.HandleFunc("/u/e/", func(w http.ResponseWriter, r *http.Request) {
		echoes := strings.Split(r.URL.Path[5:], "/")
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
			ids := db.SelectIDS(ii.Query{Echo: e, Start: idx, Lim: lim})
			for _, id := range ids {
				fmt.Fprintf(w, "%s\n", id)
			}
		}
	})
	http.HandleFunc("/m/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[3:]
		if !ii.IsMsgId(id) {
			return
		}
		m := db.Get(id)
		ii.Info.Printf("/m/%s %s", id, m)
		if m != nil {
			fmt.Fprintf(w, m.String())
		}
	})
	http.HandleFunc("/e/", func(w http.ResponseWriter, r *http.Request) {
		e := r.URL.Path[3:]
		if !ii.IsEcho(e) {
			return
		}
		ids := db.SelectIDS(ii.Query{Echo: e})
		for _, id := range ids {
			fmt.Fprintf(w, "%s\n", id)
		}
	})
	http.HandleFunc("/x/features", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "list.txt\nu/e\nx/c\n")
	})
	ii.Info.Printf("Listening on %s", *listen_opt)
	if err := http.ListenAndServe(*listen_opt, nil); err != nil {
		ii.Error.Printf("Error running web server: %s", err)
	}
}
