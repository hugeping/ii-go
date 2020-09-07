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
	"time"
)

type MsgInfo struct {
	Id    string
	Echo  string
	To    string
	Off   int64
	Repto string
	From  string
}

type Index struct {
	Hash     map[string]MsgInfo
	List     []string
	FileSize int64
}

type DB struct {
	Path string
	Idx  Index
	Sync sync.RWMutex
	IdxSync sync.RWMutex
	Name string
}

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

func (db *DB) Lock() bool {
	try := 16
	for try > 0 {
		if err := os.Mkdir(db.LockPath(), 0777); err == nil || os.IsExist(err) {
			return true
		}
		time.Sleep(time.Second)
		try -= 1
	}
	Error.Printf("Can not acquire lock for 16 seconds")
	return false
}

func (db *DB) Unlock() {
	os.Remove(db.LockPath())
}

func (db *DB) IndexPath() string {
	return fmt.Sprintf("%s.idx", db.Path)
}

func (db *DB) BundlePath() string {
	return fmt.Sprintf("%s", db.Path)
}

func (db *DB) LockPath() string {
	pat := strings.Replace(db.Path, "/", "_", -1)
	return fmt.Sprintf("%s/%s-bundle.lock", os.TempDir(), pat)
}

// var MaxMsgLen int = 128 * 1024 * 1024

func (db *DB) CreateIndex() error {
	db.Sync.Lock()
	defer db.Sync.Unlock()
	db.Lock()
	defer db.Unlock()

	return db._CreateIndex()
}
func file_lines(path string, fn func(string) bool) error {
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

func (db *DB) _CreateIndex() error {
	fidx, err := os.OpenFile(db.IndexPath(), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fidx.Close()
	var off int64
	return file_lines(db.BundlePath(), func(line string) bool {
		msg, _ := DecodeBundle(line)
		if msg == nil {
			off += int64(len(line) + 1)
			return true
		}
		repto, _ := msg.Tag("repto")
		fidx.WriteString(fmt.Sprintf("%s:%s:%d:%s:%s:%s\n",
			msg.MsgId, msg.Echo, off, msg.To, msg.From, repto))
		off += int64(len(line) + 1)
		return true
	})
}
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
	err = f_lines(file, func(line string) bool {
		linenr++
		info := strings.Split(line, ":")
		if len(info) < 6 {
			err2 = errors.New("Wrong format on line:" + fmt.Sprintf("%d", linenr))
			return false
		}
		mi := MsgInfo{Id: info[0], Echo: info[1], To: info[3], From: info[4] }
		if _, err := fmt.Sscanf(info[2], "%d", &mi.Off); err != nil {
			err2 = errors.New("Wrong offset on line: " + fmt.Sprintf("%d", linenr))
			return false
		}
		mi.Repto = info[5]
		if _, ok := Idx.Hash[mi.Id]; !ok { // new msg
			Idx.List = append(Idx.List, mi.Id)
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

func (db *DB) LookupFast(Id string, bl bool) *MsgInfo {
	if Id == "" {
		return nil
	}
	return db._Lookup(Id, bl, false)
}

func (db *DB) Lookup(Id string) *MsgInfo {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.Lock()
	defer db.Unlock()

	return db._Lookup(Id, false, true)
}

func (db *DB) Exists(Id string) *MsgInfo {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.Lock()
	defer db.Unlock()

	return db._Lookup(Id, true, true)
}

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

func (db *DB) GetBundle(Id string) string {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.Lock()
	defer db.Unlock()

	return db._GetBundle(Id, true)
}

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

type Query struct {
	Echo        string
	Repto       string
	From        string
	To          string
	Start       int
	Lim         int
	Blacklisted bool
}

func prependStr(x []string, y string) []string {
	x = append(x, "")
	copy(x[1:], x)
	x[0] = y
	return x
}

func (db *DB) Match(info MsgInfo, r Query) bool {
	if r.Blacklisted {
		return info.Off < 0
	}
	if r.Echo != "" && r.Echo != info.Echo {
		return false
	}
	if r.Repto != "" && r.Repto != info.Repto {
		return false
	}
	if r.To != "" && r.To != info.To {
		return false
	}
	if r.From != "" && r.From != info.From {
		return false
	}
	return true
}

type Echo struct {
	Name   string
	Count  int
	Topics int
	Last   MsgInfo
	Msg    *Msg
}

func (db *DB) Echoes(names []string) []*Echo {
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

func (db *DB)GetTopics(mi []*MsgInfo) map[string][]string {
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
			return l[i].Off < l[j].Off
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

func (db *DB) Store(m *Msg) error {
	return db._Store(m, false)
}

func (db *DB) Edit(m *Msg) error {
	return db._Store(m, true)
}

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

type User struct {
	Id     int32
	Name   string
	Mail   string
	Secret string
	Tags   Tags
}

type UDB struct {
	Path    string
	Names   map[string]User
	ById    map[int32]string
	Secrets map[string]string
	List    []string
	Sync    sync.RWMutex
	FileSize    int64
}

func IsUsername(u string) bool {
	return !strings.ContainsAny(u, ":\n\r\t/") &&
		!strings.HasPrefix(u, " ") &&
		!strings.HasSuffix(u, " ") &&
		len(u) <= 16 && len(u) > 2
}

func IsPassword(u string) bool {
	return len(u) >= 1
}

func MakeSecret(msg string) string {
	h := sha256.Sum256([]byte(msg))
	s := base64.URLEncoding.EncodeToString(h[:])
	return s[0:10]
}

func (db *UDB) Secret(User string) string {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	ui, ok := db.Names[User]
	if !ok {
		return ""
	}
	return ui.Secret
}

func (db *UDB) Auth(User string, Passwd string) bool {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	ui, ok := db.Names[User]
	if !ok {
		return false
	}
	return ui.Secret == MakeSecret(User+Passwd)
}

func (db *UDB) Access(Secret string) bool {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	_, ok := db.Secrets[Secret]
	return ok
}

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
func (db *UDB) UserInfoName(name string) *User {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	v, ok := db.Names[name]
	if ok {
		return &v
	}
	Error.Printf("No user: %s", name)
	return nil
}

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

func OpenUsers(path string) *UDB {
	var db UDB
	db.Path = path
	return &db
}
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
		if err := append_file(db.Path + ".tmp", fmt.Sprintf("%d:%s:%s:%s:%s",
			ui.Id, Name, ui.Mail, ui.Secret, ui.Tags.String())); err != nil {
			return err
		}
	}
	if err := os.Rename(db.Path + ".tmp", db.Path); err != nil {
		return err
	}
	db.FileSize = 0 // force to reload
	return nil
}

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
	err = file_lines(db.Path, func(line string) bool {
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

type EDB struct {
	Info map[string]string
	List []string
	Path string
}

func (db *EDB) Allowed(name string) bool {
	if len(db.List) == 0 {
		return true
	}
	if _, ok := db.Info[name]; ok {
		return true
	}
	return false
}
func LoadEcholist(path string) *EDB {
	var db EDB
	db.Path = path
	db.Info = make(map[string]string)

	err := file_lines(path, func(line string) bool {
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
