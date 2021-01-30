// Database functions.
// Database is the file with line-bundles: msgid:base64 encoded msg.
// File db.idx is created and mantained automatically.
// There is also points.txt, db of users.
package ii

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// This is index entry. Information about message that is loaded in memory.
// So, the index could not be very huge.
// Num: sequence number.
// Id: MsgId
// Echo: Echoarea
// To, From, Repto: message attributes
// Off: offset to bundle-line in database (in bytes)
type MsgInfo struct {
	Num   int
	Id    string
	Echo  string
	To    string
	Off   int64
	Repto string
	From  string
}

// Index object. Holds List and Hash for all MsgInfo entries
// FileSize is used to auto reread new entries if it has changed by
// someone.
type Index struct {
	Hash     map[string]MsgInfo
	List     []string
	FileSize int64
}

// Database object. Returns by OpenDB.
// Idx: Index structure (like dictionary).
// Name: database name, 'db' by default.
// Sync: used to syncronize access to DB from goroutines (many readers, one writer).
// IdxSync: same, but for Index.
// LockDepth: used for recursive file lock, to avoid conflict between ii-tool and ii-node.
type DB struct {
	Path      string
	Idx       Index
	Sync      sync.RWMutex
	IdxSync   sync.RWMutex
	Name      string
	LockDepth int32
}

// Utility function. Just append line (text) to file (fn)
func append_file(fn string, text string) error {
	f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(text + "\n"); err != nil {
		return err
	}
	return nil
}

// Recursive file lock. Used to avoid conflicts between ii-tool and ii-node.
// Uses mkdir as atomic operation.
// Note: dirs created as db.LockPath()
// 16 sec is limit.
func (db *DB) Lock() bool {
	if atomic.AddInt32(&db.LockDepth, 1) > 1 {
		return true
	}
	try := 16
	for try > 0 {
		if err := os.Mkdir(db.LockPath(), 0777); err == nil {
			return true
		}
		time.Sleep(time.Second)
		try -= 1
	}
	Error.Printf("Can not acquire lock for 16 seconds: %s", db.LockPath())
	return false
}

// Recursive file lock: unlock
// See Lock comment.
func (db *DB) Unlock() {
	if atomic.AddInt32(&db.LockDepth, -1) > 0 {
		return
	}
	os.Remove(db.LockPath())
}

// Returns path to index file.
func (db *DB) IndexPath() string {
	return fmt.Sprintf("%s.idx", db.Path)
}

// Return path to database itself
func (db *DB) BundlePath() string {
	return fmt.Sprintf("%s", db.Path)
}

// Returns path to lock.
func (db *DB) LockPath() string {
	pat := strings.Replace(db.Path, "/", "_", -1)
	return fmt.Sprintf("%s/%s-bundle.lock", os.TempDir(), pat)
}

// var MaxMsgLen int = 128 * 1024 * 1024

// This function creates index. It locks.
func (db *DB) CreateIndex() error {
	db.Sync.Lock()
	defer db.Sync.Unlock()
	db.Lock()
	defer db.Unlock()

	return db._CreateIndex()
}

// Utility to pass all lines of file (path) to fn(line).
// Stops on EOF or fn returns false.
func FileLines(path string, fn func(string) bool) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	return f_lines(f, fn)
}

// Internal function to implement FileLines. Works with
// file by *File object.
func f_lines(f *os.File, fn func(string) bool) error {
	reader := bufio.NewReader(f)
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
	// scanner := bufio.NewScanner(f)
	// scanner.Buffer(make([]byte, MaxMsgLen), MaxMsgLen)

	// for scanner.Scan() {
	// 	line := scanner.Text()
	// 	if !fn(line) {
	// 		break
	// 	}
	// }
	return nil
}

// Internal function of CreateIndex.
// Does not lock!
func (db *DB) _CreateIndex() error {
	fidx, err := os.OpenFile(db.IndexPath(), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fidx.Close()
	var off int64
	return FileLines(db.BundlePath(), func(line string) bool {
		msg, _ := DecodeBundle(line)
		if msg == nil {
			off += int64(len(line) + 1)
			return true
		}
		repto, _ := msg.Tag("repto")
		ioff := off
		if v, _ := msg.Tag("access"); v == "blacklist" {
			ioff = -1
		}
		fidx.WriteString(fmt.Sprintf("%s:%s:%d:%s:%s:%s\n",
			msg.MsgId, msg.Echo, ioff, msg.To, msg.From, repto))
		off += int64(len(line) + 1)
		return true
	})
}

// Internal function. Create and open new index.
func (db *DB) _ReopenIndex() (*os.File, error) {
	err := db._CreateIndex()
	if err != nil {
		return nil, err
	}
	file, err := os.Open(db.IndexPath())
	if err != nil {
		return nil, err
	}
	return file, nil
}

// Loads index. If index doesent exists, create and load it.
// If index was changed, reread tail.
// This function does lock.
func (db *DB) LoadIndex() error {
	db.IdxSync.Lock()
	defer db.IdxSync.Unlock()
	var Idx Index
	file, err := os.Open(db.IndexPath())
	if err != nil {
		db.Idx = Idx
		if os.IsNotExist(err) {
			file, err = db._ReopenIndex()
			if err != nil {
				Error.Printf("Can not seek to end of index")
				return err
			}
		} else {
			Error.Printf("Can not open index: %s", err)
			return err
		}
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		Error.Printf("Can not stat index: %s", err)
		return err
	}
	fsize := info.Size()

	if db.Idx.Hash != nil { // already loaded
		if fsize > db.Idx.FileSize {
			Trace.Printf("Refreshing index file...%d>%d", fsize, db.Idx.FileSize)
			if _, err := file.Seek(db.Idx.FileSize, 0); err != nil {
				Error.Printf("Can not seek index: %s", err)
				return err
			}
			Idx = db.Idx
		} else if info.Size() < db.Idx.FileSize {
			Info.Printf("Index file truncated, rebuild inndex...")
			file, err = db._ReopenIndex()
			if err != nil {
				Error.Printf("Can not reopen index: %s", err)
				return err
			}
			defer file.Close()
		} else {
			return nil
		}
	} else {
		Idx.Hash = make(map[string]MsgInfo)
	}
	var err2 error
	linenr := 0
	nr := len(Idx.List)
	err = f_lines(file, func(line string) bool {
		linenr++
		info := strings.Split(line, ":")
		if len(info) < 6 {
			err2 = errors.New("Wrong format on line:" + fmt.Sprintf("%d", linenr))
			return false
		}
		mi := MsgInfo{Num: nr, Id: info[0], Echo: info[1], To: info[3], From: info[4]}
		if _, err := fmt.Sscanf(info[2], "%d", &mi.Off); err != nil {
			err2 = errors.New("Wrong offset on line: " + fmt.Sprintf("%d", linenr))
			return false
		}
		mi.Repto = info[5]
		if mm, ok := Idx.Hash[mi.Id]; !ok { // new msg
			Idx.List = append(Idx.List, mi.Id)
			nr++
		} else {
			mi.Num = mm.Num
		}
		Idx.Hash[mi.Id] = mi
		// Trace.Printf("Adding %s to index", mi.Id)
		return true
	})
	if err != nil {
		Error.Printf("Can not parse index: %s", err)
		return err
	}
	if err2 != nil {
		Error.Printf("Can not parse index: %s", err2)
		return err2
	}
	Idx.FileSize = fsize
	db.Idx = Idx
	return nil
}

// Internal function to Lookup message in loaded index.
// If idx parameter is true, load and created index.
// Returns MsgInfo pointer or nil if fails.
// Does lock!
// bl: look in blacklisted messages too?
func (db *DB) _Lookup(Id string, bl bool, idx bool) *MsgInfo {
	if idx {
		if err := db.LoadIndex(); err != nil {
			return nil
		}
	}
	db.IdxSync.RLock()
	defer db.IdxSync.RUnlock()
	info, ok := db.Idx.Hash[Id]
	if !ok || (!bl && info.Off < 0) {
		return nil
	}
	return &info
}

// Lookup variant, but without locking.
// Useful if caller do locking logic himself.
func (db *DB) LookupFast(Id string, bl bool) *MsgInfo {
	if Id == "" {
		return nil
	}
	return db._Lookup(Id, bl, false)
}

// Lookup message in index.
// Do not search blacklisted messages.
// Creates/load index if needed.
// Returns MsgInfo pointer.
// Does lock!
func (db *DB) Lookup(Id string) *MsgInfo {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.Lock()
	defer db.Unlock()

	return db._Lookup(Id, false, true)
}

// Same as Lookup, but checks in blacklisted messages too
func (db *DB) Exists(Id string) *MsgInfo {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.Lock()
	defer db.Unlock()

	return db._Lookup(Id, true, true)
}

// Lookup messages in index.
// Gets: slice of message ids to get.
// Returns slice of MsgInfo pointers.
// Does lock!
func (db *DB) LookupIDS(Ids []string) []*MsgInfo {
	var info []*MsgInfo
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.Lock()
	defer db.Unlock()
	for _, id := range Ids {
		i := db._Lookup(id, false, true)
		if i != nil {
			info = append(info, i)
		}
	}
	return info
}

// Internal function. Gets bundle by message id.
// If idx is true: load/create index.
// Returns: msgid:base64 bundle.
// Does not lock!
func (db *DB) _GetBundle(Id string, idx bool) string {
	info := db._Lookup(Id, false, idx)
	if info == nil {
		Info.Printf("Can not find bundle: %s\n", Id)
		return ""
	}
	f, err := os.Open(db.BundlePath())
	if err != nil {
		Error.Printf("Can not open DB: %s\n", err)
		return ""
	}
	defer f.Close()
	_, err = f.Seek(info.Off, 0)
	if err != nil {
		Error.Printf("Can not seek DB: %s\n", err)
		return ""
	}
	var bundle string
	err = f_lines(f, func(line string) bool {
		bundle = line
		return false
	})
	if err != nil {
		Error.Printf("Can not get %s from DB: %s\n", Id, err)
		return ""
	}
	return bundle
}

// Get bundle line by message id from db.
// Does lock!
// Loads/create index if needed.
func (db *DB) GetBundle(Id string) string {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.Lock()
	defer db.Unlock()

	return db._GetBundle(Id, true)
}

// Get decoded message from db by message id.
// Does lock. Loads/create index if needed.
func (db *DB) Get(Id string) *Msg {
	bundle := db.GetBundle(Id)
	if bundle == "" {
		return nil
	}
	m, err := DecodeBundle(bundle)
	if err != nil {
		Error.Printf("Can not decode bundle on get: %s\n", Id)
	}
	return m
}

// Fast varian (w/o locking) of Get.
// Get decoded message from db by message id.
// Does NOT lock! Loads/create index if needed.
func (db *DB) GetFast(Id string) *Msg {
	bundle := db._GetBundle(Id, false)
	if bundle == "" {
		return nil
	}
	m, err := DecodeBundle(bundle)
	if err != nil {
		Error.Printf("Can not decode bundle on get: %s\n", Id)
	}
	return m
}

// Query used to make queries to Index
// If some field of: Echo, Repto, From, To is not ""
// fields will be matched with MsgInfo entry (logical AND).
// If Match function is not nil, this function will be used for matching.
// Blacklisted: search in blacklisted messages if true.
// User: authorized access to private areas.
// Start & Lim: slice of query. For example: -1, 1 -- get last message in db. 0, 1 -- first.
type Query struct {
	Echo        string
	Repto       string
	From        string
	To          string
	Start       int
	Lim         int
	Blacklisted bool
	User        User
	Match       func(mi MsgInfo, q Query) bool
}

// utility function to add string in front of slice
func prependStr(x []string, y string) []string {
	x = append(x, "")
	copy(x[1:], x)
	x[0] = y
	return x
}

// Default match function for queries.
func (db *DB) Match(info MsgInfo, r Query) bool {
	if r.Blacklisted {
		if info.Off >= 0 {
			return false
		}
	} else if info.Off < 0 {
		return false
	}
	if r.Echo != "" && r.Echo != info.Echo {
		return false
	}
	if r.Repto == "!" {
		if info.Repto != "" {
			return false
		}
	} else if r.Repto != "" && r.Repto != info.Repto {
		return false
	}
	if r.To != "" && r.To != info.To {
		return false
	}
	if r.From != "" && r.From != info.From {
		return false
	}
	if IsPrivate(info.Echo) {
		if r.User.Name == "" {
			return false
		}
		if info.To != "All" && info.From != r.User.Name && info.To != r.User.Name {
			return false
		}
	}
	if r.Match != nil {
		return r.Match(info, r)
	}
	return true
}

// Used to get information about echoarea
// Count: number of messages
// Topics: number of topics
// Last: last MsgInfo
// Msg: last message pointer
type Echo struct {
	Name   string
	Count  int
	Topics int
	Last   MsgInfo
	Msg    *Msg
}

// Make query and select Echoes
// Returns: slice of pointers to Echo.
// names: if not empty, lookup only in theese echoareas
// Does lock.
// Load/create index if needed.
// Echoes sorted by date of last messages.
func (db *DB) Echoes(names []string, q Query) []*Echo {
	db.Sync.Lock()
	defer db.Sync.Unlock()
	db.Lock()
	defer db.Unlock()
	var list []*Echo

	filter := make(map[string]bool)
	for _, n := range names {
		filter[n] = true
	}

	if err := db.LoadIndex(); err != nil {
		return list
	}

	db.IdxSync.RLock()
	defer db.IdxSync.RUnlock()

	hash := make(map[string]Echo)
	size := len(db.Idx.List)
	for i := 0; i < size; i++ {
		id := db.Idx.List[i]
		info := db.Idx.Hash[id]
		if info.Off < 0 {
			continue
		}
		if !db.Match(info, q) {
			continue
		}
		e := info.Echo
		if names != nil { // filter?
			if _, ok := filter[e]; !ok {
				continue
			}
		}
		if v, ok := hash[e]; ok {
			if info.Repto == "" {
				v.Topics++
			}
			v.Count++
			v.Last = info
			hash[e] = v
		} else {
			v := Echo{Name: e, Count: 1, Last: info}
			if info.Repto == "" {
				v.Topics = 1
			}
			hash[e] = v
		}
	}
	if names != nil {
		for _, v := range names {
			n := hash[v]
			list = append(list, &n)
		}
	} else {
		for _, v := range hash {
			n := v
			list = append(list, &n)
		}
	}
	for _, v := range list {
		v.Msg = db.GetFast(v.Last.Id)
		if v.Msg == nil {
			Error.Printf("Can not get echo last message: %s", v.Last.Id)
			v.Msg = &Msg{}
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Msg.Date > list[j].Msg.Date
	})
	return list
}

// Make query and retuen ids as slice of strings.
// Does lock. Can create/load index if needed.
// r: request, see Query
func (db *DB) SelectIDS(r Query) []string {
	var Resp []string
	db.Sync.Lock()
	defer db.Sync.Unlock()
	db.Lock()
	defer db.Unlock()

	if err := db.LoadIndex(); err != nil {
		return Resp
	}
	size := len(db.Idx.List)
	if size == 0 {
		return Resp
	}

	db.IdxSync.RLock()
	defer db.IdxSync.RUnlock()

	if r.Start < 0 {
		start := 0
		for i := size - 1; i >= 0; i-- {
			id := db.Idx.List[i]
			if db.Match(db.Idx.Hash[id], r) {
				Resp = prependStr(Resp, id)
				start -= 1
				if start == r.Start {
					break
				}
			}
		}
		if r.Lim > 0 && len(Resp) > r.Lim {
			Resp = Resp[0:r.Lim]
		}
		return Resp
	}
	found := 0
	for i := r.Start; i < size; i++ {
		id := db.Idx.List[i]
		if db.Match(db.Idx.Hash[id], r) {
			Resp = append(Resp, id)
			found += 1
			if r.Lim > 0 && found == r.Lim {
				break
			}
		}
	}
	return Resp
}

// Internal function. Get slice of MsgInfo pointers
// and create information about topics.
// Information returns in form of: [topicid][]ids
// topic id is the msg id of most old parent in echo
// ids - is the messages in this topic
func (db *DB) GetTopics(mi []*MsgInfo) map[string][]string {
	db.Sync.RLock()
	defer db.Sync.RUnlock()

	intopic := make(map[string]string)
	topics := make(map[string][]string)

	db.LoadIndex()
	for _, m := range mi {
		if _, ok := intopic[m.Id]; ok {
			continue
		}
		var l []*MsgInfo
		for p := m; p != nil; p = db.LookupFast(p.Repto, false) {
			if m.Echo != p.Echo {
				continue
			}
			l = append(l, p)
		}
		if len(l) == 0 {
			continue
		}
		t := l[len(l)-1]
		if len(topics[t.Id]) == 0 {
			topics[t.Id] = append(topics[t.Id], t.Id)
		}
		sort.SliceStable(l, func(i int, j int) bool {
			return l[i].Num < l[j].Num
		})
		for _, i := range l {
			if i.Id == t.Id {
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

// Store decoded message in database
// If message exists, returns error
func (db *DB) Store(m *Msg) error {
	return db._Store(m, false)
}

// Store decoded message in database
// even it is exists. So, it's like Edit operation.
// While index loaded, it got last version of message data.
func (db *DB) Edit(m *Msg) error {
	return db._Store(m, true)
}

// Blacklist decoded message.
// Blacklisting is adding special tag: access/blacklist and Edit operation
// to store it in DB. While loading index, blacklisted messages
// are marked by negative Off field (-1).
func (db *DB) Blacklist(m *Msg) error {
	m.Tags.Add("access/blacklist")
	return db.Edit(m)

	//db.Sync.Lock()
	//defer db.Sync.Unlock()
	//db.Lock()
	//defer db.Unlock()
	// repto, _ := m.Tag("repto")
	// if repto != "" {
	// 	repto = ":" + repto
	// }
	// rec := fmt.Sprintf("%s:%s:%d%s", m.MsgId, m.Echo, -1, repto)
	// if err := append_file(db.IndexPath(), rec); err != nil {
	// 	return err
	// }
	// return nil
}

// Internal function used by Store. See Store comment.
func (db *DB) _Store(m *Msg, edit bool) error {
	db.Sync.Lock()
	defer db.Sync.Unlock()
	db.Lock()
	defer db.Unlock()
	repto, _ := m.Tag("repto")
	if err := db.LoadIndex(); err != nil {
		return err
	}

	db.IdxSync.RLock()
	defer db.IdxSync.RUnlock()

	if _, ok := db.Idx.Hash[m.MsgId]; ok && !edit { // exist and not edit
		return errors.New("Already exists")
	}
	fi, err := os.Stat(db.BundlePath())
	var off int64
	if err == nil {
		off = fi.Size()
	}
	if v, _ := m.Tag("access"); v == "blacklist" {
		off = -1
	}
	if err := append_file(db.BundlePath(), m.Encode()); err != nil {
		return err
	}

	rec := fmt.Sprintf("%s:%s:%d:%s:%s:%s", m.MsgId, m.Echo, off, m.To, m.From, repto)
	if err := append_file(db.IndexPath(), rec); err != nil {
		return err
	}
	return nil
}

// Opens DB and returns pointer to DB object.
// path is the path to db. By default it is ./db
// Index will be named as path + ".idx"
func OpenDB(path string) *DB {
	var db DB
	db.Path = path
	info, err := os.Stat(filepath.Dir(path))
	if err != nil || !info.IsDir() {
		return nil
	}
	db.Name = "node"
	//	db.Idx = make(map[string]Index)
	return &db
}

// User entry in points.txt db
// User with Id == 1 is superuser.
// Tags: custom information (like avatars :) in Tags format
type User struct {
	Id     int32
	Name   string
	Mail   string
	Secret string
	Tags   Tags
}

// User database.
// FileSize - size of points.txt to detect DB changes.
// Names: holds User structure by user name
// ById: holds user name by user id
// Secrets: holds user name by user secret (pauth)
// List: holds user names as list
type UDB struct {
	Path     string
	Names    map[string]User
	ById     map[int32]string
	Secrets  map[string]string
	List     []string
	Sync     sync.RWMutex
	FileSize int64
}

// Check username if it is valid
func IsUsername(u string) bool {
	return !strings.ContainsAny(u, ":\n\r\t/") &&
		!strings.HasPrefix(u, " ") &&
		!strings.HasSuffix(u, " ") &&
		len(u) <= 16 && len(u) > 2
}

// Check password if it is valid to be used
func IsPassword(u string) bool {
	return len(u) >= 1
}

// Make secret from string.
// String is something like id + user + password
func MakeSecret(msg string) string {
	h := sha256.Sum256([]byte(msg))
	s := base64.URLEncoding.EncodeToString(h[:])
	return s[0:10]
}

// Return secret for username or "" if no such user
func (db *UDB) Secret(User string) string {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	ui, ok := db.Names[User]
	if !ok {
		return ""
	}
	return ui.Secret
}

// Returns true if user+password is valid
func (db *UDB) Auth(User string, Passwd string) bool {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	ui, ok := db.Names[User]
	if !ok {
		return false
	}
	return ui.Secret == MakeSecret(User+Passwd)
}

// Returns true if Secret (pauth) is valid
func (db *UDB) Access(Secret string) bool {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	_, ok := db.Secrets[Secret]
	return ok
}

// Return username for given Secret
func (db *UDB) Name(Secret string) string {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	name, ok := db.Secrets[Secret]
	if ok {
		return name
	}
	Error.Printf("No user for secret: %s", Secret)
	return ""
}

// Return User pointer for given Secret
func (db *UDB) UserInfo(Secret string) *User {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	name, ok := db.Secrets[Secret]
	if ok {
		v := db.Names[name]
		return &v
	}
	Error.Printf("No user for secret: %s", Secret)
	return nil
}

// Return User pointer for user id
func (db *UDB) UserInfoId(id int32) *User {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	name, ok := db.ById[id]
	if ok {
		v := db.Names[name]
		return &v
	}
	Error.Printf("No user for Id: %d", id)
	return nil
}

// Return User pointer for given user name
func (db *UDB) UserInfoName(name string) *User {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	v, ok := db.Names[name]
	if ok {
		return &v
	}
	return nil
}

// Return user id for given secret
func (db *UDB) Id(Secret string) int32 {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	name, ok := db.Secrets[Secret]
	if ok {
		v, ok := db.Names[name]
		if !ok {
			return -1
		}
		return v.Id
	}
	Error.Printf("No user for secret: %s", Secret)
	return -1
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// Add (register) user in database
// Mail is optional but someday it will be used in registration process
func (db *UDB) Add(Name string, Mail string, Passwd string) error {
	db.Sync.Lock()
	defer db.Sync.Unlock()

	if _, ok := db.Names[Name]; ok {
		return errors.New("User already exists")
	}
	if !IsUsername(Name) {
		return errors.New("Wrong username")
	}
	if !IsPassword(Passwd) {
		return errors.New("Bad password")
	}
	if !emailRegex.MatchString(Mail) {
		return errors.New("Wrong email")
	}
	var id int32 = 0
	for _, v := range db.Names {
		if v.Id > id {
			id = v.Id
		}
	}
	id++
	var u User
	u.Name = Name
	u.Mail = Mail
	u.Secret = MakeSecret(Name + Passwd)
	u.Tags = NewTags("")
	db.List = append(db.List, u.Name)
	if err := append_file(db.Path, fmt.Sprintf("%d:%s:%s:%s:%s",
		id, Name, Mail, u.Secret, u.Tags.String())); err != nil {
		return err
	}
	return nil
}

// Open user database and return pointer to UDB object
func OpenUsers(path string) *UDB {
	var db UDB
	db.Path = path
	return &db
}

// Change (replace) information about user.
// Gets pointer to User object and write it in DB, replacing old information.
// Works atomically using rename.
func (db *UDB) Edit(u *User) error {
	db.Sync.Lock()
	defer db.Sync.Unlock()
	if _, ok := db.Names[u.Name]; !ok {
		return errors.New("No such user")
	}
	db.Names[u.Name] = *u // new version
	os.Remove(db.Path + ".tmp")
	for _, Name := range db.List {
		ui := db.Names[Name]
		if err := append_file(db.Path+".tmp", fmt.Sprintf("%d:%s:%s:%s:%s",
			ui.Id, Name, ui.Mail, ui.Secret, ui.Tags.String())); err != nil {
			return err
		}
	}
	if err := os.Rename(db.Path+".tmp", db.Path); err != nil {
		return err
	}
	db.FileSize = 0 // force to reload
	return nil
}

// Load user information in memory if it is needed (FileSize changed).
// So, it is safe to call it on every request.
func (db *UDB) LoadUsers() error {
	db.Sync.Lock()
	defer db.Sync.Unlock()
	var fsize int64
	file, err := os.Open(db.Path)
	if err == nil {
		info, err := file.Stat()
		file.Close()
		if err != nil {
			Error.Printf("Can not stat %s file: %s", db.Path, err)
			return err
		}
		fsize = info.Size()
	} else if os.IsNotExist(err) {
		fsize = 0
	} else {
		Error.Printf("Can not open %s file: %s", db.Path, err)
		return err
	}
	if db.FileSize == fsize {
		return nil
	}
	db.Names = make(map[string]User)
	db.Secrets = make(map[string]string)
	db.ById = make(map[int32]string)
	db.List = nil
	err = FileLines(db.Path, func(line string) bool {
		a := strings.Split(line, ":")
		if len(a) < 4 {
			Error.Printf("Wrong entry in user DB: %s", line)
			return true
		}
		var u User
		var err error
		_, err = fmt.Sscanf(a[0], "%d", &u.Id)
		if err != nil {
			Error.Printf("Wrong ID in user DB: %s", a[0])
			return true
		}
		u.Name = a[1]
		u.Mail = a[2]
		u.Secret = a[3]
		u.Tags = NewTags(a[4])
		db.ById[u.Id] = u.Name
		db.Names[u.Name] = u
		db.Secrets[u.Secret] = u.Name
		db.List = append(db.List, u.Name)
		return true
	})
	if err != nil {
		Error.Printf("Can not read user DB: %s", err)
		return errors.New(err.Error())
	}
	db.FileSize = fsize
	return nil
}

// Echo database entry
// Holds echo descriptions in Info hash.
// List - names of echoareas.
type EDB struct {
	Info map[string]string
	List []string
	Path string
}

// Check if echo is exists in echo database
func (db *EDB) Allowed(name string) bool {
	if len(db.List) == 0 {
		return true
	}
	if _, ok := db.Info[name]; ok {
		return true
	}
	return false
}

// Loads echolist database and returns pointer to EDB
// Supposed to be called only once
func LoadEcholist(path string) *EDB {
	var db EDB
	db.Path = path
	db.Info = make(map[string]string)

	err := FileLines(path, func(line string) bool {
		a := strings.SplitN(line, ":", 3)
		if len(a) < 2 {
			Error.Printf("Wrong entry in echo DB: %s", line)
			return true
		}
		db.Info[a[0]] = a[2]
		db.List = append(db.List, a[0])
		return true
	})
	if err != nil {
		Error.Printf("Can not read echo DB: %s", err)
		return nil
	}
	return &db
}
