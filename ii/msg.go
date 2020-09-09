package ii

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Tags struct {
	Hash map[string]string
	List []string
}

type Msg struct {
	MsgId string
	Tags  Tags
	Echo  string
	Date  int64
	From  string
	Addr  string
	To    string
	Subj  string
	Text  string
}

func MsgId(msg string) string {
	h := sha256.Sum256([]byte(msg))
	id := base64.StdEncoding.EncodeToString(h[:])
	id = strings.Replace(id, "+", "A", -1)
	id = strings.Replace(id, "/", "Z", -1)
	return id[0:20]
}

func IsMsgId(id string) bool {
	return len(id) == 20 && !strings.Contains(id, ".")
}

func IsPrivate(e string) bool {
	return strings.HasPrefix(e, ".")
}

func IsEcho(e string) bool {
	l := len(e)
	return l >= 3 && l <= 120 && strings.Contains(e, ".") && !strings.Contains(e, ":")
}

func IsSubject(s string) bool {
	return true // len(strings.TrimSpace(s)) > 0
}

func IsEmptySubject(s string) bool {
	return len(strings.TrimSpace(s)) > 0
}

func DecodeMsgline(msg string, enc bool) (*Msg, error) {
	var m Msg
	var data []byte
	var err error
	if len(msg) > 65536 {
		return nil, errors.New("Message too long")
	}
	if enc {
		if data, err = base64.StdEncoding.DecodeString(msg); err != nil {
			if data, err = base64.URLEncoding.DecodeString(msg); err != nil {
				return nil, err
			}
		}
	} else {
		data = []byte(msg)
	}
	text := strings.Split(string(data), "\n")
	if len(text) < 5 {
		return nil, errors.New("Wrong message format")
	}
	if text[3] != "" {
		return nil, errors.New("No body delimiter in message")
	}
	m.Echo = strings.TrimSpace(text[0])
	if !IsEcho(m.Echo) {
		return nil, errors.New("Wrong echoarea format")
	}
	m.To = strings.TrimSpace(text[1])
	if len(m.To) == 0 {
		m.To = "All"
	}
	m.Subj = strings.TrimSpace(text[2])
	if !IsEmptySubject(m.Subj) {
		return nil, errors.New("Wrong subject")
	}
	m.Date = time.Now().Unix()
	start := 4
	repto := text[4]
	m.Tags, _ = MakeTags("ii/ok")
	if strings.HasPrefix(repto, "@repto:") {
		start += 1
		repto = strings.Trim(strings.Split(repto, ":")[1], " ")
		m.Tags.Add("repto/" + repto)
		Trace.Printf("Add repto tag: %s", repto)
	}
	for i := start; i < len(text); i++ {
		m.Text += text[i] + "\n"
	}
	m.Text = strings.TrimSuffix(m.Text, "\n")
	Trace.Printf("Final message: %s\n", m.String())
	return &m, nil
}

func DecodeBundle(msg string) (*Msg, error) {
	var m Msg
	if strings.Contains(msg, ":") {
		spl := strings.Split(msg, ":")
		if len(spl) != 2 {
			return nil, errors.New("Wrong bundle format")
		}
		msg = spl[1]
		m.MsgId = spl[0]
		if !IsMsgId(m.MsgId) {
			return nil, errors.New("Wrong MsgId format")
		}
	}
	data, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return nil, err
	}
	if m.MsgId == "" {
		m.MsgId = MsgId(string(data))
	}
	text := strings.Split(string(data), "\n")
	if len(text) <= 8 {
		return nil, errors.New("Wrong message format")
	}
	m.Tags, err = MakeTags(text[0])
	if err != nil {
		return nil, err
	}
	m.Echo = text[1]
	if !IsEcho(m.Echo) {
		return nil, errors.New("Wrong echoarea format")
	}
	_, err = fmt.Sscanf(text[2], "%d", &m.Date)
	if err != nil {
		return nil, err
	}
	m.From = text[3]
	m.Addr = text[4]
	m.To = text[5]
	m.Subj = text[6]
	if !IsSubject(m.Subj) {
		return nil, errors.New("Wrong subject")
	}
	for i := 8; i < len(text); i++ {
		m.Text += text[i] + "\n"
	}
	m.Text = strings.TrimSuffix(m.Text, "\n")
	return &m, nil
}

func MakeTags(str string) (Tags, error) {
	var t Tags
	str = strings.Trim(str, " ")
	if str == "" { // empty
		return t, nil
	}
	tags := strings.Split(str, "/")
	if len(tags)%2 != 0 {
		return t, errors.New("Wrong tags: " + str)
	}
	t.Hash = make(map[string]string)
	for i := 0; i < len(tags); i += 2 {
		t.Hash[tags[i]] = tags[i+1]
		t.List = append(t.List, tags[i])
	}
	return t, nil
}

func NewTags(str string) Tags {
	t, _ := MakeTags(str)
	return t
}

func (t *Tags) Get(n string) (string, bool) {
	if t == nil || t.Hash == nil {
		return "", false
	}
	v, ok := t.Hash[n]
	if ok {
		return v, true
	}
	return "", false
}

func (t *Tags) Add(str string) error {
	tags := strings.Split(str, "/")
	if len(tags)%2 != 0 {
		return errors.New("Wrong tags")
	}
	if t.Hash == nil {
		t.Hash = make(map[string]string)
	}
	for i := 0; i < len(tags); i += 2 {
		_, ok := t.Hash[tags[i]]
		t.Hash[tags[i]] = tags[i+1]
		if !ok { /* new tag */
			t.List = append(t.List, tags[i])
		}
	}
	return nil
}
func (t *Tags) Del(tag string) bool {
	if t.Hash == nil {
		return false
	}
	_, ok := t.Hash[tag]
	if !ok {
		return false
	}
	delete(t.Hash, tag)
	for k, v := range t.List {
		if v == tag {
			copy(t.List[k:], t.List[k+1:])
			t.List[len(t.List)-1] = ""
			t.List = t.List[:len(t.List)-1]
			return true
		}
	}
	return false
}

func (t Tags) String() string {
	var text string
	if t.Hash == nil {
		return ""
	}
	for _, n := range t.List {
		if val, ok := t.Hash[n]; ok {
			text += fmt.Sprintf("%s/%s/", n, val)
		}
	}
	text = strings.TrimSuffix(text, "/")
	return text
}

func (m *Msg) Dump() string {
	if m == nil {
		return ""
	}
	return fmt.Sprintf("id: %s\ntags: %s\nechoarea: %s\ndate: %s\nmsgfrom: %s\naddr: %s\nmsgto: %s\nsubj: %s\n\n%s",
		m.MsgId, m.Tags.String(), m.Echo, time.Unix(m.Date, 0), m.From, m.Addr, m.To, m.Subj, m.Text)
}

func (m *Msg) Tag(n string) (string, bool) {
	return m.Tags.Get(n)
}

func (m *Msg) String() string {
	tags := m.Tags.String()
	text := strings.Join([]string{tags, m.Echo,
		fmt.Sprint(m.Date),
		m.From,
		m.Addr,
		m.To,
		m.Subj,
		"",
		m.Text}, "\n")
	return text
}

func (m *Msg) Encode() string {
	var text string
	if m == nil || m.Echo == "" {
		return ""
	}
	if m.Date == 0 {
		now := time.Now()
		m.Date = now.Unix()
	}
	text = m.String()
	if m.MsgId == "" {
		m.MsgId = MsgId(text)
	}
	return m.MsgId + ":" + base64.StdEncoding.EncodeToString([]byte(text))
}
