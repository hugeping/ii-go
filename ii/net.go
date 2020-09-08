package ii

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	//	"net/smtp"
	"net/url"
	"strings"
	"sync"
)

type Node struct {
	Host     string
	Features map[string]bool
}

func http_req_lines(url string, fn func(string) bool) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		line = strings.TrimSuffix(line, "\n")
		if err == io.EOF {
			break
		}
		if !fn(line) {
			break
		}
	}
	return nil
}
func http_get_id(url string) (string, error) {
	res := ""
	if err := http_req_lines(url, func(line string) bool {
		if strings.Contains(line, ".") {
			return true
		}
		res += line
		return true
	}); err != nil {
		return "", err
	}
	return res, nil
}

func (n *Node) Fetcher(db *DB, Echo string, limit int, wait *sync.WaitGroup, cond *sync.Cond) {
	defer func() {
		cond.L.Lock()
		cond.Broadcast()
		cond.L.Unlock()
	}()
	defer wait.Done()
	if n.IsFeature("u/e") { /* fast path */
		id, err := http_get_id(n.Host + "/u/e/" + Echo + "/-1:1")
		if !IsMsgId(id) {
			Info.Printf("%s: no valid MsgId", Echo)
			return
		}
		if err == nil && db.Exists(id) != nil { /* no sync needed */
			Info.Printf("%s: no sync needed", Echo)
			return
		}
		if limit < 0 {
			limit = -limit
			try := 0
			for { // adaptive
				if try > 16 { /* fallback to old scheme */
					limit = 0
					break
				}
				id, err := http_get_id(fmt.Sprintf("%s/u/e/%s/%d:1",
					n.Host, Echo, -limit))
				if err != nil { /* fallback to old scheme */
					limit = 0
					break
				}
				if db.Exists(id) != nil {
					break
				}
				try++
				limit *= 2
			}
		}
	} else {
		limit = 0
	}
	req := fmt.Sprintf("%s/u/e/%s", n.Host, Echo)
	if limit > 0 {
		req = fmt.Sprintf("%s/%d:%d", req, -limit, limit)
	}
	Info.Printf("Get %s", req)
	var res []string
	if err := http_req_lines(req, func(line string) bool {
		if strings.Contains(line, ".") {
			return true
		}
		if db.Exists(line) == nil {
			res = append(res, line)
		}
		return true
	}); err != nil {
		return
	}
	n.Store(db, res)
}

var MaxConnections = 6

func (n *Node) Send(pauth string, msg string) error {
	msg = base64.URLEncoding.EncodeToString([]byte(msg))
	//	msg = url.QueryEscape(msg)
	req := fmt.Sprintf("%s/u/point/%s/%s", n.Host, pauth, msg)
	resp, err := http.Get(req)
	Trace.Printf("Get %s", req)
	if err != nil {
		return err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if strings.HasPrefix(string(buf), "msg ok") {
		Trace.Printf("Server responced msg ok")
		return nil
	} else if len(buf) > 0 {
		err = errors.New(string(buf))
	}
	if err == nil {
		err = errors.New("Server did not response with ok")
	}
	return err
}

func (n *Node) Post(pauth string, msg string) error {
	msg = base64.StdEncoding.EncodeToString([]byte(msg))
	// msg = url.QueryEscape(msg)
	postData := url.Values{
		"pauth": {pauth},
		"tmsg":  {msg},
	}
	resp, err := http.PostForm(n.Host+"/u/point", postData)
	Trace.Printf("Post %s/u/point", n.Host)
	if err != nil {
		return err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if strings.HasPrefix(string(buf), "msg ok") {
		Trace.Printf("Server responced msg ok")
		return nil
	} else if len(buf) > 0 {
		err = errors.New(string(buf))
	}
	if err == nil {
		err = errors.New("Server did not response with ok")
	}
	return err
}

func (n *Node) List() ([]string, error) {
	var list []string
	if !n.IsFeature("list.txt") {
		return list, nil
	}
	if err := http_req_lines(n.Host+"/list.txt", func(line string) bool {
		list = append(list, strings.Split(line, ":")[0])
		return true
	}); err != nil {
		return list, err
	}
	return list, nil
}

func (n *Node) Store(db *DB, ids []string) error {
	req := ""
	var nreq int
	count := len(ids)
	Trace.Printf("Get and store messages")
	for i := 0; i < count; i++ {
		req = req + "/" + string(ids[i])
		nreq++
		if nreq < 8 && i < count-1 {
			continue
		}
		if err := http_req_lines(n.Host+"/u/m"+req, func(b string) bool {
			m, e := DecodeBundle(b)
			if e != nil {
				Error.Printf("Can not decode message %s (%s)\n", b, e)
				return true
			}
			if e := db.Store(m); e != nil {
				Error.Printf("Can not write message %s (%s)\n", m.MsgId, e)
			}
			return true
		}); err != nil {
			return err
		}
		nreq = 0
		req = ""
	}
	return nil
}

func (n *Node) Fetch(db *DB, Echolist []string, limit int) error {
	if len(Echolist) == 0 {
		Echolist, _ = n.List()
	}
	if Echolist == nil {
		return nil
	}
	var wait sync.WaitGroup
	cond := sync.NewCond(&sync.Mutex{})
	num := 0
	Info.Printf("Start fetcher(s) for %s", n.Host)
	for _, v := range Echolist {
		wait.Add(1)
		num += 1
		if num >= MaxConnections { /* add per one */
			cond.L.Lock()
			Trace.Printf("Start fetcher for: %s", v)
			go n.Fetcher(db, v, limit, &wait, cond)
			Trace.Printf("Waiting free thread")
			cond.Wait()
			cond.L.Unlock()
		} else {
			Trace.Printf("Start fetcher for: %s", v)
			go n.Fetcher(db, v, limit, &wait, cond)
		}
	}
	Trace.Printf("Waiting thread(s)")
	wait.Wait()
	return nil
}

func (n *Node) IsFeature(f string) bool {
	_, ok := n.Features[f]
	return ok
}

func Connect(addr string) (*Node, error) {
	var n Node
	n.Host = strings.TrimSuffix(addr, "/")
	n.Features = make(map[string]bool)
	if err := http_req_lines(n.Host+"/x/features", func(line string) bool {
		n.Features[line] = true
		Trace.Printf("%s supports %s", n.Host, line)
		return true
	}); err != nil {
		return nil, err
	}
	return &n, nil
}

/*
func SendMail(email string, login string, passwd string, server string) error {
	aserv := strings.Split(server, ":")[0]
	auth := smtp.PlainAuth("", login, passwd, aserv)
	msg := "Hello!"
	msg = "From: noreply@ii-go\n" +
		"To: " + email + "\n" +
		"Subject: Hello there\n\n" +
		msg
	err := smtp.SendMail(server, auth, "noreply@ii-go",[]string{email}, []byte(msg))
	if err != nil {
		Error.Printf("Can't send message to: %s", email)
		return err
	}
	Info.Printf("Sent message to: %s", email)
	return nil
}
*/
