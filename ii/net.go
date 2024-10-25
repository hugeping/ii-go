// Network operations: fetch, post/get message from point
// Check node extensions.

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

// Node object. Use Connect to create it.
// Host: url node
// Features: extensions map
// Force: force sync even last message is not new
type Node struct {
	Host     string
	Features map[string]bool
	Force    bool
}

// utility function to make get request and call fn
// for every line. Stops on EOF or fn return false.
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
			if line != "" { /* node do not send final \n */
				fn(line)
			}
			break
		}
		if !fn(line) {
			break
		}
	}
	return nil
}

// short variant of http_get_lines. Read one line and
// interpret it as message id. Return it.
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

// Fetcher internal goroutine.
// DB: db to write
// Echo: echo to fetch
// wait: sync for Fetch master to detect finishing of work
// cond: used for wake-up new goroutines
// Can work in different modes.
// If limit > 0, just fetch last [limit] messages (-limit:limit slice)
// if limit < 0, use adaptive mode, probe (-(2*n)* limit:1) messages
// untill find old message.
// if node does not support u/e slices, than full sync performed
// if node connection is not in Force mode, do not perform sync if not needed
func (n *Node) Fetcher(db *DB, Echo string, limit int, wait *sync.WaitGroup, cond *sync.Cond) {
	defer func() {
		cond.L.Lock()
		cond.Broadcast()
		cond.L.Unlock()
	}()
	defer wait.Done()
	if n.IsFeature("u/e") { /* fast path */
		if !n.Force {
			id, err := http_get_id(n.Host + "/u/e/" + Echo + "/-1:1")
			if err != nil || !IsMsgId(id) {
				Info.Printf("%s: no valid MsgId (%s)", Echo, id)
				limit = 0
			} else if db.Exists(id) != nil { /* no sync needed */
				Info.Printf("%s: no sync needed", Echo)
				return
			}
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
				if err != nil || !IsMsgId(id) { /* fallback to old scheme */
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

// Do not run more then MaxConnections goroutines in the same time
var MaxConnections = 6

// Send point message to node using GET method of /u/point scheme.
// pauth: secret string. msg - raw message in plaintext
// returns error
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

// Send point message to node using POST method of /u/point scheme.
// pauth: secret string. msg - raw message in plaintext
// returns error
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

// Return list.txt in []string if node supports it.
// WARNING: Only echo names are returned! Each string is just echoarea.
// Used for fetch all mode.
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

// Fetch and write selected messages in db.
// ids: selected message ids.
// db: Database.
// This function make /u/m request, decodes bundles, checks,
// and write them to db (line by line).
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

// This is Fetcher master function. It makes fetch from node
// and run goroutines in parralel mode (one goroutine per echo).
// Echolist: list with echoarea names. If list is empty,
// function will try to get list via list.txt request.
// limit: see Fetcher function. Describe fetching mode/limit.
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
		if !IsEcho(v) {
			if strings.Trim(v, " ") != "" {
				Trace.Printf("Skip echo: %s", v)
			}
			continue
		}
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

// Check if node has feature?
// Features are gets while Connect call.
func (n *Node) IsFeature(f string) bool {
	_, ok := n.Features[f]
	return ok
}

// Connect to node, get features and returns
// pointer to Node object.
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
// commented out routine to send e-mails ;)
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
