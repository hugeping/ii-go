package ii

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestOpenDB(t *testing.T) {
	var db *DB
	dir, err := ioutil.TempDir(os.TempDir(), "ii.test.*")
	if err != nil {
		t.Error("Can not create temp dir")
		return
	}
	defer os.RemoveAll(dir)
	path := dir + "/db"
	db = OpenDB(path)
	if db == nil {
		t.Error("Can not open db")
		return
	}
	var m *Msg
	if m, err = DecodeBundle(Test_msg); m == nil {
		t.Error("Can not decode msg", err)
		return
	}
	if err := db.Store(m); err != nil {
		t.Error("Can not save msg", err)
		return
	}
	m2 := db.Get(m.MsgId)
	if m2 == nil || m2.Text != m.Text {
		t.Error("Can not lookup msg")
		return
	}

	os.Remove(db.IndexPath())

	db = OpenDB(path) // reopen
	m2 = db.Get(m.MsgId)
	if m2 == nil || m2.Text != m.Text {
		t.Error("Can not lookup msg (create new index)")
		return
	}
	m2.Text = "Edited"
	if err := db.Edit(m2); err != nil {
		t.Error("Can not save duplicate msg", err)
		return
	}
	m3 := db.Get(m2.MsgId)
	if m3 == nil || m3.Text != m2.Text {
		t.Error("Can not lookup msg (edited)")
		return
	}
	db = OpenDB(path) // reopen
	m3 = db.Get(m.MsgId)
	if m3 == nil || m3.Text != m2.Text {
		t.Error("Can not lookup msg (reopen, edited msg)", m3.Text)
		return
	}
	os.Remove(db.IndexPath())
	db = OpenDB(path) // reopen
	m3 = db.Get(m.MsgId)
	if m3 == nil || m3.Text != m2.Text {
		t.Error("Can not lookup msg (create new index, edited msg)", m3.Text)
		return
	}
}
