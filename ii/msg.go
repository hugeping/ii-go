// Message bundles manipulations (encode/decode).
// Decode message from user (point).
// Some validation functions.

package ii

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// II-tags, encoded in raw message as key1/value1/key2/value2.. string
// When message is decoded into Msg,
// key/value properties of tags associated with it.
// When encoding Msg, all properties will translated to tags string
// List - is the names of properties
// Hash - is the map of properties (Name->Value)
type Tags struct {
	Hash map[string]string
	List []string
}

// Decoded message.
// Has all atrributes of message
// including Tags.
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

// Make MsgId from raw text
// MsgId is unique identificator of message
// It is supposed that there is no collision of MsgId
// It is base64(sha256(text)) transformation
func MsgId(msg string) string {
	h := sha256.Sum256([]byte(msg))
	id := base64.StdEncoding.EncodeToString(h[:])
	id = strings.Replace(id, "+", "A", -1)
	id = strings.Replace(id, "/", "Z", -1)
	return id[0:20]
}

// Check if string is valid MsgId
func IsMsgId(id string) bool {
	return len(id) == 20 && !strings.Contains(id, ".")
}

// Check if Echoarea is private area
// This is ii-go extension, echoareas
// that has "." prefix are for private messaging.
// Those areas can be fetched only with /u/point/auth/u/e/ scheme
func IsPrivate(e string) bool {
	return strings.HasPrefix(e, ".")
}

// Check if string is valid echoarea name
func IsEcho(e string) bool {
	l := len(e)
	return l >= 3 && l <= 120 && strings.Contains(e, ".") && !strings.Contains(e, ":")
}

// Check if string is valid subject
// In fact, it is just return true stub :)
func IsSubject(s string) bool {
	return true // len(strings.TrimSpace(s)) > 0
}

// Check if subject is empty string
// Used when validate msg from points
func IsEmptySubject(s string) bool {
	return len(strings.TrimSpace(s)) > 0
}

// Decode message from point sent with /u/point scheme.
// Try to use URL save and STD base64.
// Returns pointrt to decoded Msg or nil (and error)
// Note: This function adds "ii/ok" to Tags and
// set Date field with UTC Unix time.
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

// Decode bundle line in msgid:message format or just message
// Returns pointer to decoded Msg or nil, error if fail.
// Can parse URL safe and STD BASE64.
// This function does NOT add ii/ok tag and does NOT change Date
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
	msg = strings.Replace(msg, "-", "+", -1) /* if it is URL base64 */
	msg = strings.Replace(msg, "_", "/", -1) /* make it base64 */
	data, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return nil, err
	}
	data_str := strings.Replace(string(data), "\r", "", -1)
	if m.MsgId == "" {
		m.MsgId = MsgId(data_str)
	}
	text := strings.Split(data_str, "\n")
	if len(text) <= 8 {
		return nil, errors.New("Wrong message format")
	}
	m.Tags, err = MakeTags(text[0])
	if err != nil {
		return nil, err
	}
	repto, ok := m.Tags.Get("repto")
	if ok && !IsMsgId(repto) {
		return nil, errors.New("Wrong repto format")
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

// Creates Tags from string in key1/value1/key2/value2/... format
// Can return error (with unfilled Tags) if format is wrong.
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

// Create Tags from string in key1/value1/key2/value2/... format
// ignoring errors. This is useful for creating new "ii/ok" tag.
func NewTags(str string) Tags {
	t, _ := MakeTags(str)
	return t
}

// Returns Tags propertie with name n.
// Returns "", false if such propertie does not exists in Tags.
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

// Add tags in key/value/... format to existing Tags.
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

// Remove tag with name tag from Tags.
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

// Translate Tags to string in key1/value1/key2/value2/... format.
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

// Dump (returns string) decoded message for debug purposes.
func (m *Msg) Dump() string {
	if m == nil {
		return ""
	}
	return fmt.Sprintf("id: %s\ntags: %s\nechoarea: %s\ndate: %s\nmsgfrom: %s\naddr: %s\nmsgto: %s\nsubj: %s\n\n%s",
		m.MsgId, m.Tags.String(), m.Echo, time.Unix(m.Date, 0), m.From, m.Addr, m.To, m.Subj, m.Text)
}

// Get if tag property with name n is associated with Msg
func (m *Msg) Tag(n string) (string, bool) {
	return m.Tags.Get(n)
}

// Translate decoded Msg to raw text format ready to encoding.
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

// Encode Msg into bundle format (msgid:base64text).
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
