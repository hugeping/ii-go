package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/hugeping/ii-go/ii"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"
)

func open_db(path string) *ii.DB {
	db := ii.OpenDB(path)
	if db == nil {
		fmt.Printf("Can no open db: %s\n", path)
		os.Exit(1)
	}
	return db
}

func open_users_db(path string) *ii.UDB {
	db := ii.OpenUsers(path, "")
	if err := db.LoadUsers(); err != nil {
		fmt.Printf("Can no load db: %s\n", path)
		os.Exit(1)
	}
	return db
}

func GetFile(path string) string {
	var file *os.File
	var err error
	if path == "-" {
		file = os.Stdin
	} else {
		file, err = os.Open(path)
		if err != nil {
			fmt.Printf("Can not open file %s: %s\n", path, err)
			os.Exit(1)
		}
		defer file.Close()
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("Can not read file %s: %s\n", path, err)
		os.Exit(1)
	}
	return string(b)
}

func html_esc(l string) string {
	l = strings.Replace(l, "&", "&amp;", -1)
	l = strings.Replace(l, "<", "&lt;", -1)
	l = strings.Replace(l, ">", "&gt;", -1)
	return l
}

func text_clean(txt string) string {
	txt = strings.Replace(txt, "\r", "", -1)
	txt = strings.TrimLeft(txt, "\n")
	txt = strings.TrimRight(txt, "\n")
	return txt
}

func text_trunc(txt string, max int) string {
	txt = text_clean(txt)
	f := ""
	ln := 0
	lines := strings.Split(txt, "\n")
	for _, l := range lines {
		ln++
		l = strings.Replace(l, "\r", "", -1)
		if l == "====" || strings.HasPrefix(l, "/* XPM */") ||
			strings.HasPrefix(l, "! XPM2") ||
			strings.HasPrefix(l, "@base64:") {
			break
		}
		f += l + "\n"
		if ln >= max && l == "" {
			f += "..."
			break
		}
	}
	return f
}

var urlRegex = regexp.MustCompile(`(http|ftp|https|gemini)://[^ <>"]+`)

func gemini(f io.Writer, m *ii.Msg, data string) {
	fmt.Fprintln(f, "# "+m.Subj)
	if m.To != "All" && m.To != m.From {
		fmt.Fprintf(f, "To: %s\n\n", m.To)
	}
	d := time.Unix(m.Date, 0).Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "by %s on %s\n\n", m.From, d)
	temp := strings.Split(m.Text, "\n")
	pre := false
	xpm := false
	b64 := false
	b64str := ""
	b64fname := ""
	link := 0
	var links []string
	for _, l := range temp {
		l = strings.Replace(l, "\r", "", -1)
		if pre {
			if l == "====" {
				l = "```"
				pre = false
			}
		} else if xpm {
			if strings.HasSuffix(l, "};") {
				xpm = false
				fmt.Fprintln(f, l)
				fmt.Fprintln(f, "```")
				continue
			}
		} else if b64 {
			b64str += l
		} else {
			if l == "====" {
				l = "```"
				pre = true
			} else if strings.HasPrefix(l, "/* XPM */") {
				fmt.Fprintln(f, "```")
				xpm = true
			} else if strings.HasPrefix(l, "@base64:") {
				fname := strings.TrimPrefix(l, "@base64:")
				fname = strings.Trim(fname, " ")
				b64 = true
				fname = strings.Replace(fname, "/", "_", -1)
				b64fname = strings.Replace(fname, "\\", "_", -1)
			}
		}
		if !pre && !xpm && !b64 {
			l = string(urlRegex.ReplaceAllFunc([]byte(l),
				func(line []byte) []byte {
					link++
					s := string(line)
					links = append(links, fmt.Sprintf("=> %s %s [%d]",
						s, s, link))
					return []byte(fmt.Sprintf("%s [%d]", s, link))
				}))
		}
		if !b64 {
			fmt.Fprintln(f, l)
		}
	}

	for _, v := range links {
		fmt.Fprintln(f, v)
	}

	if b64 {
		if d, err := base64.StdEncoding.DecodeString(b64str); err == nil {
			if bf, err := os.Create(data + "/" + b64fname); err == nil {
				bf.Write(d)
				bf.Close()
				fmt.Fprintf(f, "=> %s %s\n", b64fname, b64fname)
			}
		}
	}
}

type TplContext struct {
	Msg []*ii.Msg
	Now int64
}

func main() {
	ii.OpenLog(ioutil.Discard, os.Stdout, os.Stderr)

	db_opt := flag.String("db", "./db", "Database path (directory)")
	lim_opt := flag.Int("lim", 0, "Fetch last N messages")
	verbose_opt := flag.Bool("v", false, "Verbose")
	bundle_opt := flag.Bool("b", false, "select: show bundles")
	invert_opt := flag.Bool("i", false, "Invert select")
	force_opt := flag.Bool("f", false, "Force full sync")
	users_opt := flag.String("u", "points.txt", "Users database")
	conns_opt := flag.Int("j", 6, "Maximum parallel jobs")
	topics_opt := flag.Bool("t", false, "select, get: topics only")
	from_opt := flag.String("from", "", "select: from")
	to_opt := flag.String("to", "", "select: to")
	count_opt := flag.Int("count", 0, "select: count <nr> messages")
	skip_opt := flag.Int("skip", 0, "select: skip <nr> messages")

	flag.Parse()
	ii.MaxConnections = *conns_opt
	if *verbose_opt {
		ii.OpenLog(os.Stdout, os.Stdout, os.Stderr)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf(`Help: %s [options] command [arguments]
Commands:
	search <string> [echo]        - search in base
	send <server> <pauth> <msg|-> - send message
	clean                         - cleanup database
	fetch <url> [echofile|-]      - fetch
	store <bundle|->              - import bundle to database
	get <msgid>                   - show message from database
	select <echo> [[start]:lim]   - get slice from echo
	index                         - recreate index
	blacklist <msgid>             - blacklist msg
	useradd <name> <e-mail> <password>
	                              - adduser
	gemini <dir>                  - ids in stdin: export articles/files to dir in .gmi
	sort                          - ids in stdin: sort by date
	template <tpl>                - ids in stdin: do golang template over msgs
Options:
	-db=<path>                    - database path
	-lim=<lim>                    - fetch lim last messages
	-u=<path>                     - points account file
	-t                            - select, get: topics only
	-from=<user>                  - select: from
	-to=<user>                    - select: to
	-skip=<nr>                    - select: skip nr msgs
	-count=<nr>                   - select: count nr msgs
	-b                            - select: show bundles
	-v                            - select, search: verbose show
	-i                            - select, sort: invert
`, os.Args[0])
		os.Exit(1)
	}
	switch cmd := args[0]; cmd {
	case "search":
		echo := ""
		if len(args) < 2 {
			fmt.Printf("No string supplied\n")
			os.Exit(1)
		}
		if len(args) > 2 {
			echo = args[2]
		}
		db := open_db(*db_opt)
		db.Lock()
		defer db.Unlock()
		db.LoadIndex()
		re, _ := regexp.Compile(args[1])
		if re == nil {
			fmt.Printf("Wrong regexp\n")
			os.Exit(1)
		}
		for _, v := range db.Idx.List {
			if echo != "" {
				mi := db.Idx.Hash[v]
				if mi.Echo != echo {
					continue
				}
			}
			m := db.GetFast(v)
			if m == nil {
				continue
			}
			if re.Match([]byte(m.String())) {
				fmt.Printf("%s\n", v)
				if *verbose_opt {
					fmt.Printf("%s\n", m)
				}
			}
		}
	case "blacklist":
		if len(args) < 2 {
			fmt.Printf("No msgid supplied\n")
			os.Exit(1)
		}
		db := open_db(*db_opt)
		m := db.Get(args[1])
		if m != nil {
			if err := db.Blacklist(m); err != nil {
				fmt.Printf("Can not blacklist: %s\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("No such msg")
		}
	case "send":
		if len(args) < 4 {
			fmt.Printf("No argumnet(s) supplied\nShould be: <server> <pauth> and <file|->.\n")
			os.Exit(1)
		}
		msg := GetFile(args[3])
		if _, err := ii.DecodeMsgline(string(msg), false); err != nil {
			fmt.Printf("Wrong message format\n")
			os.Exit(1)
		}
		n, err := ii.Connect(args[1])
		if err != nil {
			fmt.Printf("Can not connect to %s: %s\n", args[1], err)
			os.Exit(1)
		}
		if err := n.Post(args[2], msg); err != nil {
			fmt.Printf("Can not send message: %s\n", err)
			os.Exit(1)
		}
	case "useradd":
		if len(args) < 4 {
			fmt.Printf("No argumnet(s) supplied\nShould be: name, e-mail and password.\n")
			os.Exit(1)
		}
		db := open_users_db(*users_opt)
		if err := db.Add(args[1], args[2], args[3], "status/verified/info/ii-tool"); err != nil {
			fmt.Printf("Can not add user: %s\n", err)
			os.Exit(1)
		}
	case "clean":
		hash := make(map[string]int)
		last := make(map[string]string)
		nr := 0
		dup := 0
		fmt.Printf("Pass 1...\n")
		err := ii.FileLines(*db_opt, func(line string) bool {
			nr++
			a := strings.Split(line, ":")
			if len(a) != 2 {
				ii.Error.Printf("Error in line: %d", nr)
				return true
			}
			if !ii.IsMsgId(a[0]) {
				ii.Error.Printf("Error in line: %d", nr)
				return true
			}
			if _, ok := hash[a[0]]; ok {
				hash[a[0]]++
				dup++
				last[a[0]] = line
			} else {
				hash[a[0]] = 1
			}
			return true
		})
		fmt.Printf("%d lines... %d dups...\n", nr, dup)
		if dup == 0 {
			os.Exit(0)
		}
		fmt.Printf("Pass 2...\n")
		nr = 0
		f, err := os.OpenFile(*db_opt+".new", os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		skip := 0
		err = ii.FileLines(*db_opt, func(line string) bool {
			nr++
			a := strings.Split(line, ":")
			id := a[0]
			if len(a) != 2 {
				fmt.Printf("Error in line: %d\n", nr)
				skip++
				return true
			}
			if !ii.IsMsgId(id) {
				fmt.Printf("Error in line: %d\n", nr)
				skip++
				return true
			}
			if v, ok := hash[id]; !ok || v == 0 {
				fmt.Printf("Error. DB has changed. Aborted.\n")
				os.Exit(1)
			}
			if hash[id] > 0 { // first record
				hash[id] = -hash[id]
				l := line
				if hash[id] < -1 {
					l = last[id]
				}
				if _, err := f.WriteString(l + "\n"); err != nil {
					fmt.Printf("Error: %s\n", err)
					os.Exit(1)
				}
			} else {
				skip++
			}
			hash[id] += 1
			if hash[id] > 0 {
				fmt.Printf("Error. DB has changed. Aborted.\n")
				os.Exit(1)
			}
			return true
		})
		f.Close()
		if err != nil {
			fmt.Printf("Error: %s\n")
			os.Exit(1)
		}
		for _, v := range hash {
			if v != 0 {
				fmt.Printf("Error. DB shrinked. Aborted.\n")
				os.Exit(1)
			}
		}
		fmt.Printf("%d messages removed. File %s created.\n", skip, *db_opt+".new")
	case "fetch":
		var echolist []string
		if len(args) < 2 {
			fmt.Printf("No url supplied\n")
			os.Exit(1)
		}
		db := open_db(*db_opt)
		n, err := ii.Connect(args[1])
		if err != nil {
			fmt.Printf("Can not connect to %s: %s\n", args[1], err)
			os.Exit(1)
		}
		if *force_opt {
			n.Force = true
		}
		if len(args) > 2 {
			str := GetFile(args[2])
			for _, v := range strings.Split(str, "\n") {
				if strings.HasPrefix(v, "-") {
					v = v[1:]
				}
				e := strings.Split(strings.Split(v, ":")[0], "!")[0]
				echolist = append(echolist, e)
			}
		}
		err = n.Fetch(db, echolist, *lim_opt)
		if err != nil {
			fmt.Printf("Can not fetch from %s: %s\n", args[1], err)
			os.Exit(1)
		}
	case "store":
		if len(args) < 2 {
			fmt.Printf("No bundle file supplied\n")
			os.Exit(1)
		}
		db := open_db(*db_opt)
		var f *os.File
		var err error
		if args[1] == "-" {
			f = os.Stdin
		} else {
			f, err = os.Open(args[1])
		}
		if err != nil {
			fmt.Printf("Can no open bundle: %s\n", args[1])
			os.Exit(1)
		}
		defer f.Close()
		reader := bufio.NewReader(f)
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				fmt.Printf("Can read input (%s)\n", err)
				os.Exit(1)
			}
			line = strings.TrimSuffix(line, "\n")
			if err == io.EOF {
				break
			}
			m, err := ii.DecodeBundle(line)
			if m == nil {
				fmt.Printf("Can not parse message: %s (%s)\n", line, err)
				continue
			}
			if db.Lookup(m.MsgId) == nil {
				if err := db.Store(m); err != nil {
					fmt.Printf("Can not store message: %s\n", err)
					os.Exit(1)
				}
			}
		}
	case "get":
		if len(args) < 2 {
			fmt.Printf("No msgid supplied\n")
			os.Exit(1)
		}
		db := open_db(*db_opt)

		if *topics_opt {
			mi := db.Lookup(args[1])
			if mi == nil {
				return
			}
			mis := db.LookupIDS(db.SelectIDS(&ii.Query{Echo: mi.Echo}))
			topic := mi.Id
			for p := mi; p != nil; p = db.LookupFast(p.Repto, false) {
				if p.Repto == p.Id {
					break
				}
				if p.Echo != mi.Echo {
					continue
				}
				topic = p.Id
			}
			ids := db.GetTopics(mis)[topic]
			if len(ids) == 0 {
				ids = append(ids, args[1])
			}
			for _, m := range ids {
				fmt.Println(m)
			}
			return
		}

		m := db.Get(args[1])
		if m != nil {
			fmt.Println(m)
		}
	case "select":
		if len(args) < 2 {
			fmt.Printf("No echo supplied\n")
			os.Exit(1)
		}
		db := open_db(*db_opt)
		req := ii.Query{Echo: args[1], NoAccess: true,
			Invert: *invert_opt, Count: *count_opt, Skip: *skip_opt}
		if *from_opt != "" {
			req.From = *from_opt
		}
		if *to_opt != "" {
			req.To = *to_opt
		}

		if *topics_opt {
			req.Repto = "!"
		}
		if len(args) > 2 {
			fmt.Sscanf(args[2], "%d:%d", &req.Start, &req.Lim)
		}
		resp := db.SelectIDS(&req)
		for _, v := range resp {
			if *verbose_opt {
				fmt.Println(db.Get(v))
			} else if *bundle_opt {
				fmt.Println(db.GetBundleAll(v))
			} else {
				fmt.Println(v)
			}
		}
	case "sort":
		db := open_db(*db_opt)
		db.LoadIndex()
		scanner := bufio.NewScanner(os.Stdin)
		var mm []*ii.Msg
		for scanner.Scan() {
			mi := db.LookupFast(scanner.Text(), false)
			if mi != nil {
				mm = append(mm, db.Get(mi.Id))
			}
		}
		sort.SliceStable(mm, func(i, j int) bool {
			if *invert_opt {
				return mm[i].Date > mm[j].Date
			}
			return mm[i].Date < mm[j].Date
		})
		for _, v := range mm {
			if *verbose_opt {
				fmt.Println(v)
			} else {
				fmt.Println(v.MsgId)
			}
		}
	case "index":
		db := open_db(*db_opt)
		if err := db.CreateIndex(); err != nil {
			fmt.Printf("Can not rebuild index: %s\n", err)
			os.Exit(1)
		}
	case "template":
		var ctx TplContext
		ctx.Now = time.Now().Unix()
		if len(args) < 2 {
			fmt.Printf("No template supplied\n")
			os.Exit(1)
		}
		funcMap := template.FuncMap{
			"replace": func(s string, f string, t string) string {
				return strings.Replace(s, f, t, -1)
			},
			"trunc": func(t string, lines int) string {
				return text_trunc(t, lines)
			},
			"html_esc": func(t string) string {
				return html_esc(t)
			},
			"now": func() int64 {
				return time.Now().Unix()
			},
			"fmt_date": func(date int64, f string) string {
				return time.Unix(date, 0).Format(f)
			},
			"RFC3339": func(date int64) string {
				return time.Unix(date, 0).Format(time.RFC3339)
			},
		}

		tpl := template.Must(template.New("main").Funcs(funcMap).ParseFiles(args[1]))

		db := open_db(*db_opt)
		db.LoadIndex()
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			mi := db.LookupFast(scanner.Text(), false)
			if mi != nil {
				ctx.Msg = append(ctx.Msg, db.Get(mi.Id))
			}
		}
		tpl.ExecuteTemplate(os.Stdout, filepath.Base(args[1]), &ctx)
	case "gemini":
		if len(args) < 2 {
			fmt.Printf("No dir supplied\n")
			os.Exit(1)
		}
		data := strings.TrimSuffix(args[1], "/")

		db := open_db(*db_opt)
		db.LoadIndex()
		var mm []*ii.Msg
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			mi := db.LookupFast(scanner.Text(), false)
			if mi == nil {
				continue
			}
			m := db.Get(mi.Id)
			if m == nil {
				continue
			}
			mm = append(mm, db.Get(mi.Id))
		}
		for _, m := range mm {
			f, err := os.Create(data + "/" + m.MsgId + ".gmi")
			if err == nil {
				gemini(f, m, data)
				if *verbose_opt {
					fmt.Println(m.MsgId)
				}
			}
			f.Close()
		}
	default:
		fmt.Printf("Wrong cmd: %s\n", cmd)
		os.Exit(1)
	}
}
