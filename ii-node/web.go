package main

import (
	"../ii"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"
)

type WebContext struct {
	Echoes []ii.Echo
	Topics []Topic
	Msg    []ii.Msg
	Echo string
	Page int
	Pages int
	Pager []int
	BasePath string
}

func www_login(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request) error {
	var ctx WebContext
	ii.Trace.Printf("www login")
	switch r.Method {
	case "GET":
		err := www.tpl.ExecuteTemplate(w, "login.tpl", ctx)
		return err
	case "POST":
		if err := r.ParseForm(); err != nil {
			ii.Error.Printf("Error in POST request: %s", err)
			return  err
		}
		user := r.FormValue("username")
		password := r.FormValue("password")
		udb := ii.LoadUsers(*users_opt)
		if udb == nil || !udb.Auth(user, password) {
			ii.Info.Printf("Access denied for user: %s", user)
			return nil
		}
		exp := time.Now().Add(10 * 365 * 24 * time.Hour)
		cookie := http.Cookie{Name: "pauth", Value: udb.Secret(user), Expires: exp}
		http.SetCookie(w, &cookie)
		ii.Info.Printf("User logged in: %s:%s\n", user, password)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		return nil
	}
	return nil
}

func www_index(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request) error {
	var ctx WebContext
	ii.Trace.Printf("www index")

	ctx.Echoes = www.db.Echoes(nil)
	ctx.Msg = make([]ii.Msg, len(ctx.Echoes))
	www.db.LoadIndex()
	www.db.Sync.RLock()
	defer www.db.Sync.RUnlock()
	for k, e := range ctx.Echoes {
		if m := www.db.GetFast(e.Last.Id); m != nil {
			ctx.Msg[k] = *m
		}
	}
	err := www.tpl.ExecuteTemplate(w, "index.tpl", ctx)
	return err
}

func getParent(db *ii.DB, i *ii.MsgInfo) *ii.MsgInfo {
	return db.LookupFast(i.Repto, false)
}

func getTopics(db *ii.DB, mi []*ii.MsgInfo) map[string][]string {
	db.Sync.RLock()
	defer db.Sync.RUnlock()

	intopic := make(map[string]string)
	topics := make(map[string][]string)

	db.LoadIndex()
	for _, m := range mi {
		if _, ok := intopic[m.Id]; ok {
			continue
		}
		var l []string
		for p := m; p != nil; p = getParent(db, p) {
			l = append(l, p.Id)
		}
		if len(l) == 0 {
			continue
		}
		t := l[len(l) - 1]
		for _, id := range l {
			topics[t] = append(topics[t], id)
			intopic[id] = t
		}
	}
	return topics
}

type Topic struct {
	Ids     []string
	Count   int
	Last    *ii.MsgInfo
	Head    *ii.Msg
	Tail    *ii.Msg
}

const PAGE_SIZE = 100
func makePager(ctx *WebContext, count int, page int) int {
	ctx.Pages = count / PAGE_SIZE
	if count % PAGE_SIZE != 0 {
		ctx.Pages ++
	}
	if page == 0 {
		page ++
	} else if page < 0 {
		page = ctx.Pages + page + 1
	}
	start := (page - 1)* PAGE_SIZE
	if start < 0 {
		start = 0
		page = 1
	}
	ctx.Page = page
	if ctx.Pages > 1 {
		for i := 1; i <= ctx.Pages; i++ {
			ctx.Pager = append(ctx.Pager, i)
		}
	}
	return start
}

func www_topics(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request, echo string, page int) error {
	db := www.db
	var ctx WebContext

	mis := db.LookupIDS(db.SelectIDS(ii.Query{Echo: echo}))
	ii.Trace.Printf("www topics: %s", echo)
	topicsIds := getTopics(db, mis)
	var topics []*Topic
	ii.Trace.Printf("Start to generate topics")

	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.LoadIndex()
	for _, t := range topicsIds {
		topic := Topic{}
		topic.Ids = t
		topic.Count = len(topic.Ids) - 1
		topic.Last = db.LookupFast(topic.Ids[topic.Count], false)
		if topic.Last == nil {
			ii.Error.Printf("Skip wrong message: %s\n", t[0])
			continue
		}
		topics = append(topics, &topic)
	}

	sort.SliceStable(topics, func(i, j int) bool {
		return topics[i].Last.Off > topics[j].Last.Off
	})
	ctx.BasePath = echo
	tcount := len(topics)
	start := makePager(&ctx, tcount, page)
	nr := PAGE_SIZE
	for i := start; i < tcount && nr > 0; i ++ {
		t := topics[i]
		t.Head = db.GetFast(t.Ids[0])
		t.Tail = db.GetFast(t.Ids[t.Count])
		if t.Head == nil || t.Tail == nil {
			ii.Error.Printf("Skip wrong message: %s\n", t.Ids[0])
			continue
		}
		ctx.Topics = append(ctx.Topics, *topics[i])
		nr --
	}
	ii.Trace.Printf("Stop to generate topics")
	err := www.tpl.ExecuteTemplate(w, "topics.tpl", ctx)
	return err
}

func www_topic(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request, id string, page int) error {
	db := www.db

	var ctx WebContext

	mi := db.Lookup(id)
	if mi == nil {
		return errors.New("No such message")
	}
	mis := db.LookupIDS(db.SelectIDS(ii.Query{Echo: mi.Echo, Repto: ""}))
	ids := getTopics(db, mis)[id]
	ii.Trace.Printf("www topic: %s", id)
	start := makePager(&ctx, len(ids), page)
	nr := PAGE_SIZE
	for i := start; i < len(ids) && nr > 0; i++ {
		id := ids[i]
		m := db.Get(id)
		if m == nil {
			ii.Error.Printf("Skip wrong message: %s", id)
			continue
		}
		ctx.Msg = append(ctx.Msg, *m)
		nr --
	}
	ctx.BasePath = id
	err := www.tpl.ExecuteTemplate(w, "topic.tpl", ctx)
	return err
}

func msg_format(txt string) template.HTML {
	txt = strings.Replace(txt, "&", "&amp;", -1)
	txt = strings.Replace(txt, "<", "&lt;", -1)
	txt = strings.Replace(txt, ">", "&gt;", -1)
	return template.HTML(strings.Replace(txt, "\n", "<br/>", -1))
}

func WebInit(www *WWW, db *ii.DB) {
	funcMap := template.FuncMap{
		"fdate": func (date int64) string {
			return time.Unix(date, 0).Format("2006-01-02 15:04:05")
		},
		"msg_format": msg_format,
	}
	www.db = db
	www.tpl = template.Must(template.New("main").Funcs(funcMap).ParseGlob("tpl/*.tpl"))
}

func handleWWW(www WWW, w http.ResponseWriter, r *http.Request) error {
	var user *ii.User
	cookie, err := r.Cookie("pauth")
	if err == nil {
		udb := ii.LoadUsers(*users_opt) /* per each request */
		if udb.Access(cookie.Value) {
			user = udb.UserInfo(cookie.Value)
			ii.Info.Printf("User: %s GET %s", user.Name, r.URL.Path)
		}
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	args := strings.Split(path, "/")
	if path == "" {
		return www_index(user, www, w, r)
	} else if path == "login" {
		return www_login(user, www, w, r)
	} else if ii.IsMsgId(args[0]) {
		page := 1
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &page)
		}
		return www_topic(user, www, w, r, args[0], page)
	} else if ii.IsEcho(args[0]) {
		page := 1
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &page)
		}
		return www_topics(user, www, w, r, args[0], page)
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404\n")
	}
	return nil
}
