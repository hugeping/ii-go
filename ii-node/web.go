package main

import (
	"../ii"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const PAGE_SIZE = 100

type WebContext struct {
	Echoes   []*ii.Echo
	Topics   []*Topic
	Topic    string
	Msg      []*ii.Msg
	Error    string
	Echo     string
	Page     int
	Pages    int
	Pager    []int
	BasePath string
	User     *ii.User
	Echolist *ii.EDB
	Selected string
	Ref      string
	Info     string
	Sysname  string
	Host     string
	www      *WWW
}

func www_register(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	ii.Trace.Printf("www register")
	switch r.Method {
	case "GET":
		err := ctx.www.tpl.ExecuteTemplate(w, "register.tpl", ctx)
		return err
	case "POST":
		if err := r.ParseForm(); err != nil {
			ii.Error.Printf("Error in POST request: %s", err)
			return err
		}
		user := r.FormValue("username")
		password := r.FormValue("password")
		email := r.FormValue("email")

		udb := ctx.www.udb
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

func www_login(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	ii.Trace.Printf("www login")
	switch r.Method {
	case "GET":
		err := ctx.www.tpl.ExecuteTemplate(w, "login.tpl", ctx)
		return err
	case "POST":
		if err := r.ParseForm(); err != nil {
			ii.Error.Printf("Error in POST request: %s", err)
			return err
		}
		user := r.FormValue("username")
		password := r.FormValue("password")
		udb := ctx.www.udb
		if !udb.Auth(user, password) {
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

func www_profile(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	ii.Trace.Printf("www profile")
	if ctx.User.Name == "" {
		ii.Error.Printf("Access denied")
		return errors.New("Access denied")
	}
	ctx.Selected = fmt.Sprintf("%s,%d", ctx.www.db.Name, ctx.User.Id)
	ava, _ := ctx.User.Tags.Get("avatar")
	if ava != "" {
		if data, err := base64.URLEncoding.DecodeString(ava); err == nil {
			ctx.Info = string(data)
		}
	}
	err := ctx.www.tpl.ExecuteTemplate(w, "profile.tpl", ctx)
	return err
}

func www_logout(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	ii.Trace.Printf("www logout: %s", ctx.User.Name)
	if ctx.User.Name == "" {
		ii.Error.Printf("Access denied")
		return errors.New("Access denied")
	}
	cookie := http.Cookie{Name: "pauth", Value: "", Expires: time.Unix(0, 0)}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

func www_index(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	ii.Trace.Printf("www index")
	ctx.Echoes = ctx.www.db.Echoes(nil, ii.Query { User: *ctx.User })
	err := ctx.www.tpl.ExecuteTemplate(w, "index.tpl", ctx)
	return err
}

func parse_ava(txt string) *image.RGBA {
	txt = msg_clean(txt)
	lines := strings.Split(txt, "\n")
	img, _ := ParseXpm(lines)
	return img
}

var magicTable = map[string]string {
	"\xff\xd8\xff":      "image/jpeg",
	"\x89PNG\r\n\x1a\n": "image/png",
	"GIF87a":            "image/gif",
	"GIF89a":            "image/gif",
}

func check_image(incipit []byte) string {
	incipitStr := string(incipit)
	for magic, mime := range magicTable {
		if strings.HasPrefix(incipitStr, magic) {
			return mime
		}
	}
	return ""
}

func www_base64(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	id := ctx.BasePath
	m := ctx.www.db.Get(id)
	if m == nil {
		return errors.New("No such message")
	}
	lines := strings.Split(msg_clean(m.Text), "\n")
	start := false
	b64 := ""
	fname := ""
	for _, v := range lines {
		if !start && !strings.HasPrefix(v, "@base64:") {
			continue
		}
		if start {
			v = strings.Replace(v, " ", "", -1)
			if !base64Regex.MatchString(v) {
				break
			}
			b64 += v
			continue
		}
		v = strings.TrimPrefix(v, "@base64:")
		v = strings.Trim(v, " ")
		fname = v
		if fname == "" {
			fname = "file"
		}
		start = true
	}
	if b64 == "" {
		return nil
	}
	b, err := base64.RawStdEncoding.DecodeString(b64)
	if err != nil {
		if b, err = base64.StdEncoding.DecodeString(b64); err != nil {
			if b, err = base64.URLEncoding.DecodeString(b64); err != nil {
				return err
			}
		}
	}
	//	w.Header().Set("Content-Type", "image/jpeg")
	if check_image(b) != "" {
		w.Header().Set("Content-Disposition", "inline")
	} else {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fname))
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(b)))
	_, err = w.Write(b)
	return err
}
func www_avatar(ctx *WebContext, w http.ResponseWriter, r *http.Request, user string) error {
	if r.Method == "POST" { /* upload avatar */
		if ctx.User.Name == "" || ctx.User.Name != user {
			ii.Error.Printf("Access denied")
			return errors.New("Access denied")
		}
		if err := r.ParseForm(); err != nil {
			ii.Error.Printf("Error in POST request: %s", err)
			return err
		}
		ava := r.FormValue("avatar")
		if len(ava) > 2048 {
			ii.Error.Printf("Avatar is too big.")
			return errors.New("Avatar is too big (>2048 bytes)")
		}
		if ava == "" {
			ii.Trace.Printf("Delete avatar for %s", ctx.User.Name)
			ctx.User.Tags.Del("avatar")
		} else {
			img := parse_ava(ava)
			if img == nil {
				ii.Error.Printf("Wrong xpm format for avatar: " + user)
				return errors.New("Wrong xpm format")
			}
			b64 := base64.URLEncoding.EncodeToString([]byte(ava))
			ii.Trace.Printf("New avatar for %s: %s", ctx.User.Name, b64)
			ctx.User.Tags.Add("avatar/" + b64)
		}
		if err := ctx.www.udb.Edit(ctx.User); err != nil {
			ii.Error.Printf("Error saving avatar: " + user)
			return errors.New("Error saving avatar")
		}
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return nil
	}
	// var id int32
	// if !strings.HasPrefix(user, ctx.www.db.Name) {
	// 	return nil
	// }
	// user = strings.TrimPrefix(user, ctx.www.db.Name)
	// user = strings.TrimPrefix(user, ",")
	// if _, err := fmt.Sscanf(user, "%d", &id); err != nil {
	// 	return nil
	// }
	// u := ctx.www.udb.UserInfoId(id)
	u := ctx.www.udb.UserInfoName(user)
	if u == nil {
		return nil
	}
	ava, _ := u.Tags.Get("avatar")
	if ava == "" {
		return nil
	}
	if data, err := base64.URLEncoding.DecodeString(ava); err == nil {
		img := parse_ava(string(data))
		if img == nil {
			ii.Error.Printf("Wrong xpm in avatar: %s\n", u.Name)
			return nil
		}
		b := new(bytes.Buffer)
		if err := png.Encode(b, img); err == nil {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(b.Bytes())))
			if _, err := w.Write(b.Bytes()); err != nil {
				return nil
			}
			return nil
		}
		ii.Error.Printf("Can't encode avatar in png: %s\n", u.Name)
	} else {
		ii.Error.Printf("Can't decode avatar: %s\n", u.Name)
	}
	return nil
}

type Topic struct {
	Ids   []string
	Count int
	Last  *ii.MsgInfo
	Head  *ii.Msg
	Tail  *ii.Msg
}

func makePager(ctx *WebContext, count int, page int) int {
	ctx.Pages = count / PAGE_SIZE
	if count%PAGE_SIZE != 0 {
		ctx.Pages++
	}
	if page == 0 {
		page++
	} else if page < 0 {
		page = ctx.Pages + page + 1
	}
	start := (page - 1) * PAGE_SIZE
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

func Select(ctx *WebContext, q ii.Query) []string {
	q.User = *ctx.User
	return ctx.www.db.SelectIDS(q)
}

func www_query(ctx *WebContext, w http.ResponseWriter, r *http.Request, q ii.Query, page int, rss bool) error {
	db := ctx.www.db
	req := ctx.BasePath

	if rss {
		q.Start = -100
	}
	mis := db.LookupIDS(Select(ctx, q))
	ii.Trace.Printf("www query")

	sort.SliceStable(mis, func(i, j int) bool {
		return mis[i].Off > mis[j].Off
	})
	count := len(mis)
	start := makePager(ctx, count, page)
	nr := PAGE_SIZE
	for i := start; i < count && nr > 0; i++ {
		m := db.GetFast(mis[i].Id)
		if m == nil {
			ii.Error.Printf("Can't get msg: %s\n", mis[i].Id)
			continue
		}
		ctx.Msg = append(ctx.Msg, m)
		nr--
	}
	if rss {
		ctx.Topic = db.Name + " :: " + req
		fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
		fmt.Fprintf(w,
`<rss version="2.0">
<channel>
<title>%s</title>
<description>RSS feed with last messages</description>
<link>%s</link>
`,
			str_esc(ctx.Topic), ctx.www.Host)
		for _, m := range(ctx.Msg) {
			fmt.Fprintf(w,
`<item>
	<title>%s</title>
	<guid>%s</guid>
	<link>%s/%s#%s</link>
	<pubDate>%s</pubDate>
	<description>%s\n\n%s</description>
	<author>%s</author>
</item>
`,
				str_esc(m.Subj), m.MsgId, ctx.www.Host, m.MsgId, m.MsgId,
				time.Unix(m.Date, 0).Format("2006-01-02 15:04:05"),
				str_esc(fmt.Sprintf("%s -> %s\n\n", m.From, m.To)),
				str_esc(msg_text(m)),
				str_esc(m.From))
		}
		fmt.Fprintf(w,
`</channel>
</rss>
`)
		return nil
	}
	return ctx.www.tpl.ExecuteTemplate(w, "query.tpl", ctx)
}

func www_topics(ctx *WebContext, w http.ResponseWriter, r *http.Request,  page int) error {
	db := ctx.www.db
	echo := ctx.BasePath
	ctx.Echo = echo
	mis := db.LookupIDS(Select(ctx, ii.Query{Echo: echo}))
	ii.Trace.Printf("www topics: %s", echo)
	topicsIds := db.GetTopics(mis)
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
	start := makePager(ctx, tcount, page)
	nr := PAGE_SIZE
	for i := start; i < tcount && nr > 0; i++ {
		t := topics[i]
		t.Head = db.GetFast(t.Ids[0])
		t.Tail = db.GetFast(t.Ids[t.Count])
		if t.Head == nil || t.Tail == nil {
			ii.Error.Printf("Skip wrong message: %s\n", t.Ids[0])
			continue
		}
		ctx.Topics = append(ctx.Topics, topics[i])
		nr--
	}
	ii.Trace.Printf("Stop to generate topics")
	err := ctx.www.tpl.ExecuteTemplate(w, "topics.tpl", ctx)
	return err
}


func www_topic(ctx *WebContext, w http.ResponseWriter, r *http.Request, page int) error {
	id := ctx.BasePath
	db := ctx.www.db

	mi := db.Lookup(id)
	if mi == nil {
		return errors.New("No such message")
	}
	if page == 0 {
		ctx.Selected = id
	}
	ctx.Echo = mi.Echo
	mis := db.LookupIDS(Select(ctx, ii.Query{Echo: mi.Echo}))

	topic := mi.Id
	for p := mi; p != nil; p = db.LookupFast(p.Repto, false) {
		if p.Echo != mi.Echo {
			continue
		}
		topic = p.Id
	}
	ctx.Topic = topic
	ids := db.GetTopics(mis)[topic]
	if len(ids) == 0 {
		ids = append(ids, id)
	} else if topic != mi.Id {
		for k, v := range ids {
			if v == mi.Id {
				page = k/PAGE_SIZE + 1
				ctx.Selected = mi.Id
				break
			}
		}
	}
	ii.Trace.Printf("www topic: %s", id)
	start := makePager(ctx, len(ids), page)
	nr := PAGE_SIZE
	for i := start; i < len(ids) && nr > 0; i++ {
		id := ids[i]
		m := db.Get(id)
		if m == nil {
			ii.Error.Printf("Skip wrong message: %s", id)
			continue
		}
		ctx.Msg = append(ctx.Msg, m)
		nr--
	}
	err := ctx.www.tpl.ExecuteTemplate(w, "topic.tpl", ctx)
	return err
}

func www_blacklist(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	id := ctx.BasePath
	m := ctx.www.db.Get(id)
	ii.Trace.Printf("www blacklist: %s", id)
	if m == nil {
		ii.Error.Printf("No such msg: %s", id)
		return errors.New("No such msg")
	}
	if !msg_access(ctx.www, *m, *ctx.User) {
		ii.Error.Printf("Access denied")
		return errors.New("Access denied")
	}
	err := ctx.www.db.Blacklist(m)
	if err != nil {
		ii.Error.Printf("Error blacklisting: %s", id)
		return err
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

func www_edit(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	id := ctx.BasePath
	switch r.Method {
	case "GET":
		m := ctx.www.db.Get(id)
		if m == nil {
			ii.Error.Printf("No such msg: %s", id)
			return errors.New("No such msg")
		}
		msg := *m
		ln := strings.Split(msg_clean(msg.Text), "\n")
		if len(ln) > 0 {
			if strings.HasPrefix(ln[len(ln)-1], "P.S. Edited: ") {
				msg.Text = strings.Join(ln[:len(ln)-1], "\n")
			}
		}
		msg.Text = msg.Text + "\nP.S. Edited: " + time.Now().Format("2006-01-02 15:04:05")
		ctx.Msg = append(ctx.Msg, &msg)
		err := ctx.www.tpl.ExecuteTemplate(w, "edit.tpl", ctx)
		return err
	case "POST":
		ctx.BasePath = ""
		return www_new(ctx, w, r)
	}
	return nil
}

func www_new(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	echo := ctx.BasePath
	ctx.Echo = echo

	switch r.Method {
	case "GET":
		err := ctx.www.tpl.ExecuteTemplate(w, "new.tpl", ctx)
		return err
	case "POST":
		edit := (echo == "")
		ii.Trace.Printf("www new topic in %s", echo)
		if err := r.ParseForm(); err != nil {
			ii.Error.Printf("Error in POST request: %s", err)
			return err
		}
		if ctx.User.Name == "" {
			ii.Error.Printf("Access denied")
			return errors.New("Access denied")
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
		if !ctx.www.edb.Allowed(echo) && ctx.User.Id != 1{
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
		m.From = ctx.User.Name
		m.Addr = fmt.Sprintf("%s,%d", ctx.www.db.Name, ctx.User.Id)
		if repto != "" {
			m.Tags.Add("repto/" + repto)
		}
		if id != "" {
			om := ctx.www.db.Get(id)
			if (om == nil || m.Addr != om.Addr) && ctx.User.Id != 1 {
				ii.Error.Printf("Access denied")
				return errors.New("Access denied")
			}
			m.MsgId = id
			m.From = om.From
			m.Addr = om.Addr
		}
		if action == "Submit" { // submit
			if edit {
				err = ctx.www.db.Edit(m)
			} else {
				err = ctx.www.db.Store(m)
			}
			if err != nil {
				ii.Error.Printf("Error while storig new topic %s: %s", m.MsgId, err)
				return err
			}
			http.Redirect(w, r, "/"+m.MsgId+"#"+m.MsgId, http.StatusSeeOther)
			return nil
		}
		if !edit {
			m.MsgId = ""
		}
		ctx.Msg = append(ctx.Msg, m)
		err = ctx.www.tpl.ExecuteTemplate(w, "preview.tpl", ctx)
		return err
	}
	return nil
}

func www_reply(ctx *WebContext, w http.ResponseWriter, r *http.Request, quote bool) error {
	id := ctx.BasePath
	m := ctx.www.db.Get(id)
	if m == nil {
		ii.Error.Printf("No such msg: %s", id)
		return errors.New("No such msg")
	}
	msg := *m
	msg.To = msg.From
	msg.Subj = "Re: " + strings.TrimPrefix(msg.Subj, "Re: ")
	msg.Tags.Add("repto/" + id)
	if quote {
		msg.Text = msg_quote(msg.Text)
	} else {
		msg.Text = ""
	}
	ctx.Msg = append(ctx.Msg, &msg)
	err := ctx.www.tpl.ExecuteTemplate(w, "reply.tpl", ctx)
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
var url2Regex = regexp.MustCompile(`{{{href=[0-9]+}}}`)
var urlIIRegex = regexp.MustCompile(`ii://[a-zA-Z0-9]{20}`)
var base64Regex = regexp.MustCompile(`^([A-Za-z0-9+/]{4})*([A-Za-z0-9+/]{3}=|[A-Za-z0-9+/]{2}==)?$`)

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
		if strings.Trim(l, " ") == "" {
			f += l + "\n"
			continue
		}
		if strings.HasPrefix(l, ">") {
			f += ">" + l + "\n"
		} else {
			f += "> " + l + "\n"
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

func msg_esc(l string) string {
	var links []string
	link := 0
	l = string(urlIIRegex.ReplaceAllFunc([]byte(l),
		func(line []byte) []byte {
			s := string(line)
			url := strings.TrimPrefix(s, "ii://")
			links = append(links, fmt.Sprintf(`<a href="/%s#%s" class="url">%s</a>`,
				url, url, str_esc(s)))
			link++
			return []byte(fmt.Sprintf(`{{{href=%d}}}`, link-1))
		}))
	l = string(urlRegex.ReplaceAllFunc([]byte(l),
		func(line []byte) []byte {
			s := string(line)
			links = append(links, fmt.Sprintf(`<a href="%s" class="url">%s</a>`,
				s, str_esc(s)))
			link++
			return []byte(fmt.Sprintf(`{{{href=%d}}}`, link-1))
		}))
	l = str_esc(l)
	l = string(url2Regex.ReplaceAllFunc([]byte(l),
		func(line []byte) []byte {
			s := string(line)
			var n int
			fmt.Sscanf(s, "{{{href=%d}}}", &n)
			return []byte(links[n])
		}))

	return l
}

func msg_text(m *ii.Msg) string {
	if m == nil {
		return ""
	}
	txt := m.Text
	txt = msg_clean(txt)
	f := ""
	pre := false
	skip := 0
	lines := strings.Split(txt, "\n")
	for k, l := range lines {
		if skip > 0 {
			skip--
			continue
		}
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
		if strings.HasPrefix(l, "/* XPM */") || strings.HasPrefix(l, "! XPM2") {
			var img *image.RGBA
			img, skip = ParseXpm(lines[k:])
			if img != nil {
				skip--
				/* embed xpm */
				b := new(bytes.Buffer)
				if err := png.Encode(b, img); err == nil {
					b64 := base64.StdEncoding.EncodeToString(b.Bytes())
					l = fmt.Sprintf("<img class=\"img\" src=\"data:image/png;base64,%s\"><br>\n",
						b64)
					f += l
					continue
				}
			}
			skip = 0
			l = msg_esc(l)
		} else if strings.HasPrefix(l, "P.S.") || strings.HasPrefix(l, "PS:") ||
			strings.HasPrefix(l, "//") || strings.HasPrefix(l, "#") ||
			strings.HasPrefix(l, "+++ ") {
			l = fmt.Sprintf("<span class=\"comment\">%s</span>", str_esc(l))
		} else if strings.HasPrefix(l, "@spoiler:") {
			l = fmt.Sprintf("<span class=\"spoiler\">%s</span>", str_esc(ReverseStr(l)))
		} else if quoteRegex.MatchString(l) {
			l = fmt.Sprintf("<span class=\"quote\">%s</span>", str_esc(l))
		} else if strings.HasPrefix(l, "@base64:") {
			fname := strings.TrimPrefix(l, "@base64:")
			fname = strings.Trim(fname, " ")
			if fname == "" {
				fname = "file"
			}
			f += fmt.Sprintf("<a class=\"attach\" href=\"%s/base64\">%s</a><br>\n", m.MsgId, str_esc(fname))
			return f
		} else {
			l = msg_esc(l)
		}
		f += l + "<br>\n"
	}
	if pre {
		pre = false
		f += "</pre>\n"
	}
	return f
}

func msg_access(www *WWW, m ii.Msg, u ii.User) bool {
	addr := fmt.Sprintf("%s,%d", www.db.Name, u.Id)
	return addr == m.Addr || u.Id == 1
}

func WebInit(www *WWW) {
	funcMap := template.FuncMap{
		"fdate": func(date int64) template.HTML {
			if time.Now().Unix() - date < 60 * 60 * 24 {
				return template.HTML("<span class='today'>"+time.Unix(date, 0).Format("2006-01-02 15:04:05") + "</span>")
			}
			return template.HTML(time.Unix(date, 0).Format("2006-01-02 15:04:05"))
		},
		"msg_text": func(m *ii.Msg) template.HTML {
			return template.HTML(msg_text(m))
		},
		"repto": func(m ii.Msg) string {
			r, _ := m.Tag("repto")
			return r
		},
		"msg_quote": msg_quote,
		"msg_access": func(m ii.Msg, u ii.User) bool {
			return msg_access(www, m, u)
		},
		"is_even": func(i int) bool {
			return i%2 == 0
		},
		"unescape": func(s string) template.HTML {
			return template.HTML(s)
		},
		"has_avatar": func(user string) bool {
			ui := www.udb.UserInfoName(user)
			if ui != nil {
				_, ok := ui.Tags.Get("avatar")
				return ok
			}
			return false
		},
	}
	www.tpl = template.Must(template.New("main").Funcs(funcMap).ParseGlob("tpl/*.tpl"))
}

func handleErr(ctx *WebContext, w http.ResponseWriter, err error) {
	ctx.Error = err.Error()
	ctx.www.tpl.ExecuteTemplate(w, "error.tpl", ctx)
}

func handleWWW(www *WWW, w http.ResponseWriter, r *http.Request) {
	var ctx WebContext
	var user *ii.User = &ii.User{}
	ctx.User = user
	ctx.www = www
	ctx.Sysname = www.db.Name
	ctx.Host = www.Host
	www.udb.LoadUsers()
	err := _handleWWW(&ctx, w, r)
	if err != nil {
		handleErr(&ctx, w, err)
	}
}

func _handleWWW(ctx *WebContext, w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie("pauth")
	if err == nil {
		udb := ctx.www.udb
		if udb.Access(cookie.Value) {
			if user := udb.UserInfo(cookie.Value); user != nil {
				ctx.User = user
			}
		}
	}
	ii.Trace.Printf("[%s] GET %s", ctx.User.Name, r.URL.Path)
	path := strings.TrimPrefix(r.URL.Path, "/")
	args := strings.Split(path, "/")
	ctx.Echolist = ctx.www.edb
	ctx.Ref = r.Header.Get("Referer")
	if path == "" {
		ctx.BasePath = ""
		return www_index(ctx, w, r)
	} else if path == "login" {
		ctx.BasePath = "login"
		return www_login(ctx, w, r)
	} else if path == "logout" {
		ctx.BasePath = "logout"
		return www_logout(ctx, w, r)
	} else if path == "profile" {
		ctx.BasePath = "profile"
		return www_profile(ctx, w, r)
	} else if path == "register" {
		ctx.BasePath = "register"
		return www_register(ctx, w, r)
	} else if args[0] == "avatar" {
		ctx.BasePath = "avatar"
		if len(args) < 2 {
			return errors.New("Wrong request")
		}
		return www_avatar(ctx, w, r, args[1])
	} else if ii.IsMsgId(args[0]) {
		page := 0
		if len(args) > 1 {
			if args[1] == "reply" {
				ctx.BasePath = args[0]
				return www_reply(ctx, w, r, !(len(args) > 2 && args[2] == "new"))
			} else if args[1] == "edit" {
				ctx.BasePath = args[0]
				return www_edit(ctx, w, r)
			} else if args[1] == "blacklist" {
				ctx.BasePath = args[0]
				return www_blacklist(ctx, w, r)
			} else if args[1] == "base64" {
				ctx.BasePath = args[0]
				return www_base64(ctx, w, r)
			}
			fmt.Sscanf(args[1], "%d", &page)
		}
		ctx.BasePath = args[0]
		return www_topic(ctx, w, r, page)
	} else if path == "new" {
		ctx.BasePath = ""
		return www_new(ctx, w, r)
	} else if args[0] == "to" {
		page := 1
		rss := false
		if len(args) < 2 {
			return errors.New("Wrong request")
		}
		if len(args) > 2 {
			if args[2] == "rss" {
				rss = true
			} else {
				fmt.Sscanf(args[2], "%d", &page)
			}
		}
		ctx.BasePath = "to/" + args[1]
		return www_query(ctx, w, r, ii.Query { To: args[1] }, page, rss)
	} else if args[0] == "from" {
		page := 1
		rss := false
		if len(args) < 2 {
			return errors.New("Wrong request")
		}
		if len(args) > 2 {
			if args[2] == "rss" {
				rss = true
			} else {
				fmt.Sscanf(args[2], "%d", &page)
			}
		}
		ctx.BasePath = "from/" + args[1]
		return www_query(ctx, w, r, ii.Query { From: args[1] }, page, rss)
	} else if args[0] == "echo" {
		page := 1
		rss := false
		if len(args) < 2 {
			return errors.New("Wrong request")
		}
		if len(args) > 2 {
			if args[2] == "rss" {
				rss = true
			} else {
				fmt.Sscanf(args[2], "%d", &page)
			}
		}
		q := ii.Query { Echo: args[1] }
		if args[1] == "all" {
			q.Echo = ""
			q.Start = -100
		}
		ctx.BasePath = "echo/" + args[1]
		return www_query(ctx, w, r, q, page, rss)
	} else if ii.IsEcho(args[0]) {
		page := 1
		if len(args) > 1 {
			if args[1] == "new" {
				ctx.BasePath = args[0]
				return www_new(ctx, w, r)
			}
			fmt.Sscanf(args[1], "%d", &page)
		}
		ctx.BasePath = args[0]
		return www_topics(ctx, w, r, page)
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404\n")
	}
	return nil
}
