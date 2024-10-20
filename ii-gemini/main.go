package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/hugeping/ii-go/ii"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
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

var urlRegex = regexp.MustCompile(`(http|ftp|https|gemini)://[^ <>"]+`)

func gemini(f io.Writer, m *ii.Msg) {
	fmt.Fprintln(f, "# "+m.Subj)
	if m.To != "All" && m.To != m.From {
		fmt.Fprintf(f, "To: %s\n\n", m.To)
	}
	d := time.Unix(m.Date, 0).Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "by %s on %s\n\n", m.From, d)
	temp := strings.Split(m.Text, "\n")
	pre := false
	xpm := false
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
		} else {
			if l == "====" {
				l = "```"
				pre = true
			} else if strings.HasPrefix(l, "/* XPM */") {
				fmt.Fprintln(f, "```")
				xpm = true
			}
		}
		if !pre && !xpm {
			l = string(urlRegex.ReplaceAllFunc([]byte(l),
				func(line []byte) []byte {
					link++
					s := string(line)
					links = append(links, fmt.Sprintf("=> %s %s [%d]",
						s, s, link))
					return []byte(fmt.Sprintf("%s [%d]", s, link))
				}))
		}
		fmt.Fprintln(f, l)
	}
	for _, v := range links {
		fmt.Fprintln(f, v)
	}
}

func str_esc(l string) string {
	l = strings.Replace(l, "&", "&amp;", -1)
	l = strings.Replace(l, "<", "&lt;", -1)
	l = strings.Replace(l, ">", "&gt;", -1)
	return l
}

func main() {
	ii.OpenLog(ioutil.Discard, os.Stdout, os.Stderr)

	db_opt := flag.String("db", "./db", "II database path (directory)")
	data_opt := flag.String("data", "./data", "Output path (directory)")
	url_opt := flag.String("url", "localhost", "Url of station")
	base_opt := flag.String("base-url", "/", "Base Url for msgs")
	verbose_opt := flag.Bool("v", false, "Verbose")
	title_opt := flag.String("title", "ii/idec networks", "Title")
	author_opt := flag.String("author", "anonymous", "Author")
	flag.Parse()
	if *verbose_opt {
		ii.OpenLog(os.Stdout, os.Stdout, os.Stderr)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf(`Help: %s [options] command [arguments]
Commands:
	-data <path> gemini - generate gemini data
Options:
        -db=<path>                    - database path
`, os.Args[0])
		os.Exit(1)
	}
	switch cmd := args[0]; cmd {
	case "gemini":
		db := open_db(*db_opt)
		db.Lock()
		defer db.Unlock()
		db.LoadIndex()

		scanner := bufio.NewScanner(os.Stdin)
		var mis []*ii.Msg
		for scanner.Scan() {
			mi := db.LookupFast(scanner.Text(), false)
			if mi != nil {
				mis = append(mis, db.Get(mi.Id))
			}
		}
		sort.SliceStable(mis, func(i, j int) bool {
			return mis[i].Date > mis[j].Date
		})
		data := strings.TrimSuffix(*data_opt, "/")
		atom, err := os.Create(data + "/atom.xml")
		if err != nil {
			return
		}
		defer atom.Close()
		fmt.Fprintf(atom, `<?xml version='1.0' encoding='UTF-8'?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>gemini://%s/</id>
  <title>%s</title>
  <updated>%s</updated>
  <author>
    <name>%s</name>
  </author>
  <link href="gemini://%s/atom.xml" rel="self"/>
  <link href="gemini://%s/" rel="alternate"/>
`, *url_opt, *title_opt, time.Now().Format(time.RFC3339), *author_opt, *url_opt, *url_opt)
		for _, v := range mis {
			m := v
			if m != nil {
				f, err := os.Create(data + "/" + m.MsgId + ".gmi")
				if err == nil {
					gemini(f, m)
					d := time.Unix(m.Date, 0).Format("2006-01-02")
					fmt.Println("=> " + *base_opt + m.MsgId + ".gmi " + d + " - " + m.Subj)
				}
				f.Close()
				fmt.Fprintf(atom, `<entry>
  <id>gemini://%s%s%s.gmi</id>
  <title>%s</title>
  <updated>%s</updated>
  <link href="gemini://%s%s%s.gmi" rel="alternate"/>
</entry>
`, *url_opt, *base_opt, m.MsgId, str_esc(m.Subj),
					time.Unix(m.Date, 0).Format(time.RFC3339), *url_opt, *base_opt, m.MsgId)
			}
		}
		fmt.Fprintf(atom, `</feed>
`)
	default:
		fmt.Printf("Wrong cmd: %s\n", cmd)
		os.Exit(1)
	}
}
