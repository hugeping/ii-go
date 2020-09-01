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
	db := ii.LoadUsers(path)
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

func main() {
	ii.OpenLog(ioutil.Discard, os.Stdout, os.Stderr)

	db_opt := flag.String("db", "./db", "II database path (directory)")
	lim_opt := flag.Int("lim", 0, "Fetch last N messages")
	verbose_opt := flag.Bool("v", false, "Verbose")
	flag.Parse()
	if *verbose_opt {
		ii.OpenLog(os.Stdout, os.Stdout, os.Stderr)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf(`Help: %s [options] command [arguments]
Commands:
	send <server> <pauth> <msg|-> - send message
	fetch <url>      - fetch
	store <bundle|-> - import bundle to database
        get <msgid>      - show message from database
        select <echo> [[start]:lim] - get slice from echo
        index            - recreate index
	useradd <name> <e-mail> <password> - adduser
Options:
        -db=<path>       - database path
        -lim=<lim>       - fetch lim last messages
`, os.Args[0])
		os.Exit(1)
	}
	switch cmd := args[0]; cmd {
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
		db := open_users_db(*db_opt + ".usr")
		if err := db.Add(args[1], args[2], args[3]); err != nil {
			fmt.Printf("Can not add user: %s\n", err)
			os.Exit(1)
		}
	case "fetch":
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
		err = n.Fetch(db, nil, *lim_opt)
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
	os.Exit(0)
}
