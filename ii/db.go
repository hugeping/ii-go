package ii

import (
	"bufio"
	"path/filepath"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type MsgInfo struct {
	Id    string
	Echo  string
	Off   int64
	Repto string
}

type Index struct {
	Hash map[string]MsgInfo
	List []string
}

type DB struct {
	Path string
	Idx  Index
	Sync sync.RWMutex
}

func mkdir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
	}
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
	return file_lines(db.BundlePath(), func (line string) bool {
		msg, _ := DecodeBundle(line)
		if msg == nil {
			off += int64(len(line) + 1)
			return true
		}
		repto, _ := msg.Tag("repto")
		if repto != "" {
			repto = ":" + repto
		}
		fidx.WriteString(fmt.Sprintf("%s:%s:%d%s\n", msg.MsgId, msg.Echo, off, repto))
		off += int64(len(line) + 1)
		return true
	})
}

func (db *DB) LoadIndex() error {
	if db.Idx.Hash != nil { // already loaded
		return nil
	}

	file, err := os.Open(db.IndexPath())
	if err != nil {
		if os.IsNotExist(err) {
			err = db._CreateIndex()
			if err != nil {
				return err
			}
			file, err = os.Open(db.IndexPath())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	defer file.Close()

	var Idx Index
	Idx.Hash = make(map[string]MsgInfo)
	//	Idx.List = make([]string)
	var err2 error
	err = f_lines(file, func (line string) bool {
		info := strings.Split(line, ":")
		if len(info) < 3 {
			err2 = errors.New("Wrong format")
			return false
		}
		mi := MsgInfo{Id: info[0], Echo: info[1]}
		if _, err := fmt.Sscanf(info[2], "%d", &mi.Off); err != nil {
			err2 = errors.New("Wrong offset")
			return false
		}
		if len(info) > 3 {
			mi.Repto = info[3]
		}
		if _, ok := Idx.Hash[mi.Id]; !ok { // new msg
			Idx.List = append(Idx.List, mi.Id)
		}
		Idx.Hash[mi.Id] = mi
		return true
	})
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	db.Idx = Idx
	return nil
}

func (db *DB) _Lookup(Id string) *MsgInfo {
	if err := db.LoadIndex(); err != nil {
		return nil
	}
	info, ok := db.Idx.Hash[Id]
	if !ok {
		return nil
	}
	return &info
}

func (db *DB) Lookup(Id string) *MsgInfo {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.Lock()
	defer db.Unlock()

	return db._Lookup(Id)
}

func (db *DB) Get(Id string) *Msg {
	db.Sync.RLock()
	defer db.Sync.RUnlock()
	db.Lock()
	defer db.Unlock()

	info := db._Lookup(Id)
	if info == nil {
		return nil
	}
	f, err := os.Open(db.BundlePath())
	if err != nil {
		return nil
	}
	_, err = f.Seek(info.Off, 0)
	if err != nil {
		return nil
	}
	var m *Msg;
	err = f_lines(f, func (line string)bool {
		m, _ = DecodeBundle(line)
		return false
	})
	if err != nil {
		return nil
	}
	return m
}

type Query struct {
	Echo string
	Repto string
	Start int
	Lim   int
}

func prependStr(x []string, y string) []string {
	x = append(x, "")
	copy(x[1:], x)
	x[0] = y
	return x
}

func (db *DB)Match(info MsgInfo, r Query) bool {
	if r.Echo != "" && r.Echo != info.Echo {
		return false
	}
	if r.Repto != "" && r.Repto != info.Repto {
		return false
	}
	return true
}

func (db *DB)SelectIDS(r Query) []string {
	var Resp []string;
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

func (db *DB)Store(m *Msg) error {
	return db._Store(m, false)
}

func (db *DB)Edit(m *Msg) error {
	return db._Store(m, true)
}

func (db *DB) _Store(m *Msg, edit bool) error {
	db.Sync.Lock()
	defer db.Sync.Unlock()
	db.Lock()
	defer db.Unlock()
	if _, ok := db.Idx.Hash[m.MsgId]; ok && !edit { // exist and not edit
		return errors.New("Already exists")
	}
	repto, _ := m.Tag("repto")
	if err := db.LoadIndex(); err != nil {
		return err
	}
	fi, err := os.Stat(db.BundlePath())
	var off int64
	if err == nil {
		off = fi.Size()
	}
	if err := append_file(db.BundlePath(), m.Encode()); err != nil {
		return err
	}

	mi := MsgInfo{Id: m.MsgId, Echo: m.Echo, Off: off, Repto: repto}

	if repto != "" {
		repto = ":" + repto
	}
	if err := append_file(db.IndexPath(),
		fmt.Sprintf("%s:%s:%d%s", m.MsgId, m.Echo, off, repto)); err != nil {
		return err
	}
	if _, ok := db.Idx.Hash[m.MsgId]; !ok { // new msg
		db.Idx.List = append(db.Idx.List, m.MsgId)
	}
	db.Idx.Hash[m.MsgId] = mi
	return nil
}

func OpenDB(path string) *DB {
	var db DB
	db.Path = path
	info, err := os.Stat(filepath.Dir(path))
	if err != nil || !info.IsDir() {
		return nil
	}
	//	db.Idx = make(map[string]Index)
	return &db
}
