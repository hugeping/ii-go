package main

import (
	"../ii"
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
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
	db := ii.OpenUsers(path)
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

func main() {
	ii.OpenLog(ioutil.Discard, os.Stdout, os.Stderr)

	db_opt := flag.String("db", "./db", "II database path (directory)")
	lim_opt := flag.Int("lim", 0, "Fetch last N messages")
	verbose_opt := flag.Bool("v", false, "Verbose")
	force_opt := flag.Bool("f", false, "Force full sync")
	users_opt := flag.String("u", "points.txt", "Users database")
	conns_opt := flag.Int("j", 6, "Maximum parallel jobs")

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
	cc <name> [[start]:lim]       - get msgs to name
	index                         - recreate index
	blacklist <msgid>             - blacklist msg
	useradd <name> <e-mail> <password> - adduser
Options:
        -db=<path>                    - database path
        -lim=<lim>                    - fetch lim last messages
        -u=<path>                     - points account file
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
			if strings.Contains(m.Text, args[1]) {
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
		if err := db.Add(args[1], args[2], args[3]); err != nil {
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
			nr ++
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
				hash[a[0]] ++
				dup ++
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
		f, err := os.OpenFile(*db_opt + ".new", os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		skip := 0
		err = ii.FileLines(*db_opt, func(line string) bool {
			nr ++
			a := strings.Split(line, ":")
			id := a[0]
			if len(a) != 2 {
				fmt.Printf("Error in line: %d\n", nr)
				skip ++
				return true
			}
			if !ii.IsMsgId(id) {
				fmt.Printf("Error in line: %d\n", nr)
				skip ++
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
				skip ++
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
		fmt.Printf("%d messages removed. File %s created.\n", skip, *db_opt + ".new")
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
				echolist = append(echolist, strings.Split(v, ":")[0])
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
		m := db.Get(args[1])
		if m != nil {
			fmt.Println(m)
		}
	case "cc":
		if len(args) < 2 {
			fmt.Printf("No echo supplied\n")
			os.Exit(1)
		}
		db := open_db(*db_opt)
		req := ii.Query{To: args[1]}
		if len(args) > 2 {
			fmt.Sscanf(args[2], "%d:%d", &req.Start, &req.Lim)
		}
		resp := db.SelectIDS(req)
		for _, v := range resp {
			if *verbose_opt {
				fmt.Println(db.Get(v))
			} else {
				fmt.Println(v)
			}
		}
	case "select":
		if len(args) < 2 {
			fmt.Printf("No echo supplied\n")
			os.Exit(1)
		}
		db := open_db(*db_opt)
		req := ii.Query{Echo: args[1]}
		if len(args) > 2 {
			fmt.Sscanf(args[2], "%d:%d", &req.Start, &req.Lim)
		}
		resp := db.SelectIDS(req)
		for _, v := range resp {
			if *verbose_opt {
				fmt.Println(db.Get(v))
			} else {
				fmt.Println(v)
			}
		}
	case "index":
		db := open_db(*db_opt)
		if err := db.CreateIndex(); err != nil {
			fmt.Printf("Can not rebuild index: %s\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Wrong cmd: %s\n", cmd)
		os.Exit(1)
	}
}
