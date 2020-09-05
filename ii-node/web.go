package main

import (
	"../ii"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const PAGE_SIZE = 100

type WebContext struct {
	Echoes []*ii.Echo
	Topics []*Topic
	Msg    []*ii.Msg
	Error string
	Echo string
	Page int
	Pages int
	Pager []int
	BasePath string
	User *ii.User
	Echolist *ii.EDB
	Selected string
	Ref string
}

func www_register(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request) error {
	ctx := WebContext{ User: user, Echolist: www.edb, Ref: r.Header.Get("Referer") }
	ii.Trace.Printf("www register")
	switch r.Method {
	case "GET":
		err := www.tpl.ExecuteTemplate(w, "register.tpl", ctx)
		return err
	case "POST":
		if err := r.ParseForm(); err != nil {
			ii.Error.Printf("Error in POST request: %s", err)
			return  err
		}
		user := r.FormValue("username")
		password := r.FormValue("password")
		email := r.FormValue("email")

		udb := ii.LoadUsers(*users_opt)
		err := udb.Add(user, email, password)
		if err != nil {
			ii.Info.Printf("Can not register user %s: %s", user, err)
			return err
		}
		ii.Info.Printf("Registered user: %s", user)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		return nil
	}
	return nil
}

func www_login(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request) error {
	ctx := WebContext{ User: user, BasePath: "login", Echolist: www.edb, Ref: r.Header.Get("Referer") }
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
			return errors.New("Access denied")
		}
		exp := time.Now().Add(10 * 365 * 24 * time.Hour)
		cookie := http.Cookie{Name: "pauth", Value: udb.Secret(user), Expires: exp}
		http.SetCookie(w, &cookie)
		ii.Info.Printf("User logged in: %s\n", user)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil
	}
	return errors.New("Wrong method")
}

func www_profile(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request) error {
	ctx := WebContext{ User: user, BasePath: "profile", Echolist: www.edb, Ref: r.Header.Get("Referer") }
	ii.Trace.Printf("www profile")
	if user.Name == "" {
		ii.Error.Printf("Access denied")
		return  errors.New("Access denied")
	}
	err := www.tpl.ExecuteTemplate(w, "profile.tpl", ctx)
	return err
}

func www_logout(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request) error {
	ii.Trace.Printf("www logout: %s", user.Name)
	if user.Name == "" {
		ii.Error.Printf("Access denied")
		return  errors.New("Access denied")
	}
	cookie := http.Cookie{Name: "pauth", Value: "", Expires: time.Unix(0, 0)}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

func www_index(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request) error {
	ctx := WebContext{ User: user, Echolist: www.edb, Ref: r.Header.Get("Referer") }

	ii.Trace.Printf("www index")

	ctx.Echoes = www.db.Echoes(nil)
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
		var l [] *ii.MsgInfo
		for p := m; p != nil; p = getParent(db, p) {
			if m.Echo != p.Echo {
				continue
			}
			l = append(l, p)
		}
		if len(l) == 0 {
			continue
		}
		t := l[len(l) - 1]
		if len(topics[t.Id]) == 0 {
			topics[t.Id] = append(topics[t.Id], t.Id)
		}
		sort.SliceStable(l, func(i int, j int) bool {
			return l[i].Off < l[j].Off
		})
		for _, i := range l {
			if i.Id == t.Id  {
				continue
			}
			if _, ok := intopic[i.Id]; ok {
				continue
			}
			topics[t.Id] = append(topics[t.Id], i.Id)
			intopic[i.Id] = t.Id
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
	ctx := WebContext{ User: user, Echo: echo, Echolist: www.edb, Ref: r.Header.Get("Referer") }
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
		ctx.Topics = append(ctx.Topics, topics[i])
		nr --
	}
	ii.Trace.Printf("Stop to generate topics")
	err := www.tpl.ExecuteTemplate(w, "topics.tpl", ctx)
	return err
}

func www_topic(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request, id string, page int) error {
	db := www.db
	ctx := WebContext{ User: user, Echolist: www.edb, Ref: r.Header.Get("Referer") }

	mi := db.Lookup(id)
	if mi == nil {
		return errors.New("No such message")
	}
	if page == 0 {
		ctx.Selected = id
	}
	ctx.Echo = mi.Echo
	mis := db.LookupIDS(db.SelectIDS(ii.Query{Echo: mi.Echo}))

	topic := mi.Id
	for p := mi; p != nil; p = getParent(db, p) {
		if p.Echo != mi.Echo {
			continue
		}
		topic = p.Id
	}
	ids := getTopics(db, mis)[topic]
	if len(ids) == 0 {
		ids = append(ids, id)
	} else if topic != mi.Id {
		for k, v := range ids {
			if v == mi.Id {
				page = k / PAGE_SIZE + 1
				ctx.Selected = mi.Id
				break
			}
		}
	}
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
		ctx.Msg = append(ctx.Msg, m)
		nr --
	}
	ctx.BasePath = id
	err := www.tpl.ExecuteTemplate(w, "topic.tpl", ctx)
	return err
}

func www_blacklist(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request, id string) error {
	m := www.db.Get(id)
	ii.Trace.Printf("www blacklist: %s", id)
	if m == nil {
		ii.Error.Printf("No such msg: %s", id)
		return  errors.New("No such msg")
	}
	if ! msg_access(&www, *m, *user) {
		ii.Error.Printf("Access denied")
		return  errors.New("Access denied")
	}
	err := www.db.Blacklist(m)
	if err != nil {
		ii.Error.Printf("Error blacklisting: %s", id)
		return  err
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

func www_edit(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request, id string) error {
	ctx := WebContext{ User: user, Echolist: www.edb, Ref: r.Header.Get("Referer") }
	ctx.BasePath = id
	switch r.Method {
	case "GET":
		m := www.db.Get(id)
		if m == nil {
			ii.Error.Printf("No such msg: %s", id)
			return  errors.New("No such msg")
		}
		msg := *m
		ln := strings.Split(msg_clean(msg.Text), "\n")
		if len(ln) > 0 {
			fmt.Printf("%s\n", ln[len(ln) - 1])
			if strings.HasPrefix(ln[len(ln) - 1], "P.S. Edited: ") {
				msg.Text = strings.Join(ln[:len(ln) - 1], "\n")
			}
		}
		msg.Text = msg.Text + "\nP.S. Edited: " + time.Now().Format("2006-01-02 15:04:05")
		ctx.Msg = append(ctx.Msg, &msg)
		err := www.tpl.ExecuteTemplate(w, "edit.tpl", ctx)
		return err
	case "POST":
		return www_new(user, www, w, r, "")
	}
	return nil
}

func www_new(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request, echo string) error {
	ctx := WebContext{ User: user, Echolist: www.edb, Ref: r.Header.Get("Referer") }
	ctx.BasePath = echo
	ctx.Echo = echo

	switch r.Method {
	case "GET":
		err := www.tpl.ExecuteTemplate(w, "new.tpl", ctx)
		return err
	case "POST":
		edit := (echo == "")
		ii.Trace.Printf("www new topic in %s", echo)
		if err := r.ParseForm(); err != nil {
			ii.Error.Printf("Error in POST request: %s", err)
			return  err
		}
		if user.Name == "" {
			ii.Error.Printf("Access denied")
			return  errors.New("Access denied")
		}
		subj := r.FormValue("subj")
		to := r.FormValue("to")
		msg := r.FormValue("msg")
		repto := r.FormValue("repto")
		id := r.FormValue("id")
		newecho := r.FormValue("echo")
		if newecho != "" {
			echo = newecho
		}
		if ! www.edb.Allowed(echo) {
			ii.Error.Printf("This echo is disallowed")
			return errors.New("This echo is disallowed")
		}
		action := r.FormValue("action")
		text := fmt.Sprintf("%s\n%s\n%s\n\n%s", echo, to, subj, msg)
		m, err := ii.DecodeMsgline(text, false)
		if err != nil {
			ii.Error.Printf("Error while posting new topic: %s", err)
			return err
		}
		m.From = user.Name
		m.Addr = fmt.Sprintf("%s,%d", www.db.Name, user.Id)
		if repto != "" {
			m.Tags.Add("repto/" + repto)
		}
		if id != "" {
			om := www.db.Get(id)
			if (om == nil || m.Addr != om.Addr) && user.Id != 1 {
				ii.Error.Printf("Access denied")
				return  errors.New("Access denied")
			}
			m.MsgId = id
			m.From = om.From
			m.Addr = om.Addr
		}
		if action == "Submit" { // submit
			if edit {
				err = www.db.Edit(m)
			} else {
				err = www.db.Store(m)
			}
			if err != nil {
				ii.Error.Printf("Error while storig new topic %s: %s", m.MsgId, err)
				return err
			}
			http.Redirect(w, r, "/"+m.MsgId+"#" + m.MsgId, http.StatusSeeOther)
			return nil
		}
		if ! edit {
			m.MsgId = ""
		}
		ctx.Msg = append(ctx.Msg, m)
		err = www.tpl.ExecuteTemplate(w, "preview.tpl", ctx)
		return err
	}
	return nil
}

func www_reply(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request, id string) error {
	ctx := WebContext{ User: user, Echolist: www.edb, Ref: r.Header.Get("Referer") }
	ctx.BasePath = id
	m := www.db.Get(id)
	if m == nil {
		ii.Error.Printf("No such msg: %s", id)
		return  errors.New("No such msg")
	}
	msg := *m
	msg.To = msg.From
	msg.Subj = "Re: " + strings.TrimPrefix(msg.Subj, "Re: ")
	msg.Tags.Add("repto/" + id)
	msg.Text = msg_quote(msg.Text)
	ctx.Msg = append(ctx.Msg, &msg)
	err := www.tpl.ExecuteTemplate(w, "reply.tpl", ctx)
	return err
}

func str_esc(l string) string {
	l = strings.Replace(l, "&", "&amp;", -1)
	l = strings.Replace(l, "<", "&lt;", -1)
	l = strings.Replace(l, ">", "&gt;", -1)
	return l
}

var quoteRegex = regexp.MustCompile("^[^ >]*>")
var urlRegex = regexp.MustCompile(`(http|ftp|https)://[^ <>"]+`)

func msg_clean(txt string) string {
	txt = strings.Replace(txt, "\r", "", -1)
	txt = strings.TrimLeft(txt, "\n")
	txt = strings.TrimRight(txt, "\n")
	txt = strings.TrimSuffix(txt, "\n")
	return txt
}
func msg_quote(txt string) string {
	txt = msg_clean(txt)
	f := ""
	for _, l := range strings.Split(txt, "\n") {
		if strings.HasPrefix(l, ">") {
			f += ">"+ l + "\n"
		} else {
			f += "> "+ l + "\n"
		}
	}
	return f
}

func ReverseStr(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func msg_format(txt string) template.HTML {
	txt = msg_clean(txt)
	f := ""
	pre := false
	for _, l := range strings.Split(txt, "\n") {
		if strings.Trim(l, " ") == "====" {
			if !pre {
				pre = true
				f += "<pre class=\"code\">\n"
				continue
			}
			pre = false
			f += "</pre>\n"
			continue
		}
		if pre {
			f += l + "\n"
			continue
		}
		if strings.HasPrefix(l, "P.S.") || strings.HasPrefix(l, "PS:") ||
			strings.HasPrefix(l, "//") || strings.HasPrefix(l, "#")  {
			l = fmt.Sprintf("<span class=\"comment\">%s</span>", str_esc(l))
		} else if strings.HasPrefix(l, "spoiler:") {
			l = fmt.Sprintf("<span class=\"spoiler\">%s</span>", str_esc(ReverseStr(l)))
		} else if quoteRegex.MatchString(l) {
			l = fmt.Sprintf("<span class=\"quote\">%s</span>", str_esc(l))
		} else {
			l = string(urlRegex.ReplaceAllFunc([]byte(l),
				func (line []byte) []byte {
				s := string(line)
					return []byte(fmt.Sprintf(`<a href="%s" class="url">%s</a>`,
						s, str_esc(s)))
			}))
		}
		f += l + "<br>\n"
	}
	if pre {
		pre = false
		f += "</pre>\n"
	}
	return template.HTML(f)
}

func msg_access(www *WWW, m ii.Msg, u ii.User) bool {
	addr := fmt.Sprintf("%s,%d", www.db.Name, u.Id)
	return addr == m.Addr
}

func WebInit(www *WWW) {
	funcMap := template.FuncMap{
		"fdate": func (date int64) string {
			return time.Unix(date, 0).Format("2006-01-02 15:04:05")
		},
		"msg_format": msg_format,
		"repto": func (m ii.Msg) string {
			r, _ := m.Tag("repto")
			return r
		},
		"msg_quote": msg_quote,
		"msg_access": func (m ii.Msg, u ii.User) bool {
			return msg_access(www, m, u)
		},
		"is_even": func (i int) bool {
			return i % 2 == 0
		},
	}
	www.tpl = template.Must(template.New("main").Funcs(funcMap).ParseGlob("tpl/*.tpl"))
}

func handleErr(user *ii.User, www WWW, w http.ResponseWriter, err error) {
	ctx := WebContext{ Error: err.Error(), User: user, Echolist: www.edb }
	www.tpl.ExecuteTemplate(w, "error.tpl", ctx)
}

func handleWWW(www WWW, w http.ResponseWriter, r *http.Request) {
	var user *ii.User = &ii.User {}
	err := _handleWWW(user, www, w, r)
	if err != nil {
		handleErr(user, www, w, err)
	}
}

func _handleWWW(user *ii.User, www WWW, w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie("pauth")
	if err == nil {
		udb := ii.LoadUsers(*users_opt) /* per each request */
		if udb.Access(cookie.Value) {
			user = udb.UserInfo(cookie.Value)
		}
	}
	if user != nil {
		ii.Trace.Printf("[%s] GET %s", user.Name, r.URL.Path)
	} else {
		ii.Trace.Printf("GET %s", r.URL.Path)
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	args := strings.Split(path, "/")
	if path == "" {
		return www_index(user, www, w, r)
	} else if path == "login" {
		return www_login(user, www, w, r)
	} else if path == "logout" {
		return www_logout(user, www, w, r)
	} else if path == "profile" {
		return www_profile(user, www, w, r)
	} else if path == "register" {
		return www_register(user, www, w, r)
	} else if ii.IsMsgId(args[0]) {
		page := 0
		if len(args) > 1 {
			if args[1] == "reply" {
				return www_reply(user, www, w, r, args[0])
			} else if args[1] == "edit" {
				return www_edit(user, www, w, r, args[0])
			} else if args[1] == "blacklist" {
				return www_blacklist(user, www, w, r, args[0])
			}
			fmt.Sscanf(args[1], "%d", &page)
		}
		return www_topic(user, www, w, r, args[0], page)
	} else if path == "new" {
		return www_new(user, www, w, r, "")
	} else if ii.IsEcho(args[0]) {
		page := 1
		if len(args) > 1 {
			if args[1] == "new" {
				return www_new(user, www, w, r, args[0])
			}
			fmt.Sscanf(args[1], "%d", &page)
		}
		return www_topics(user, www, w, r, args[0], page)
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404\n")
	}
	return nil
}
