package ii

import (
	"encoding/base64"
	"fmt"
	"testing"
)

var Test_msg string = "a5OX4lC8uB8OIzzzGQ5B:" +
	"aWkvb2svcmVwdG8va2N3UlBEQWNuNkxsQlVRWVhMY0sKc3RkLmdhbWUKMTU5ODE5NjE1MQ" +
	"pQZXRlcgpzeXNjYWxsLDEKdzIwMTQwMwpSZTog0JvQuNC00LjRjyDigJQg0L3QtSDQvNC+0LPRgyDQv9GA0L7QudGC0Lg" +
	"g0LTQsNC70YzRiNC1LiDQntGI0LjQsdC60LA/Cgo+INCT0LTQtSDQvNGLINGB0LXQudGH0LDRgSDQvtCx0YHRg9C20LTQ" +
	"sNC10Lwg0L7RiNC40LHQutC4INCyINC40LPRgNCw0YU/DQrQnNC+0LbQvdC+INC90LAg0YTQvtGA0YPQvNC1IGh0dHA6L" +
	"y9pbnN0ZWFkLWdhbWVzLnJ1INC40LvQuCDQsiDQutCw0YDRgtC+0YfQutC1INC40LPRgNGLLCDQuNC70Lgg0LfQtNC10Y" +
	"HRjC4uLiDQkiDQu9GO0LHQvtC8INGB0LvRg9GH0LDQtSwg0L3Rg9C20LXQvSBzYXZlINC4INC+0L/QuNGB0LDQvdC40LU" +
	"g0YHQuNGC0YPQsNGG0LjQuC4NCg0KUC5TPiDQmNCz0YDQsCDRgtC+0YfQvdC+INC/0YDQvtGF0L7QtNC40LzQsCwg0L3Q" +
	"tSDRgtCw0Log0LTQsNCy0L3QviDQtdGRINC/0YDQvtGI0LvQviDQvdC10YHQutC+0LvRjNC60L4g0YfQtdC70L7QstC10" +
	"LouINCd0L4sINC60L7QvdC10YfQvdC+LCDQsdCw0LPQuCDQvNC+0LPRg9GCINCx0YvRgtGMLg=="

func TestParse(t *testing.T) {
	var m *Msg
	m, _ = DecodeBundle(Test_msg)
	if m == nil {
		t.Error("Can not decode msg")
	}
	text := m.MsgId + ":" + m.Encode()
	decoded, _ := base64.StdEncoding.DecodeString(text)
	decoded2, _ := base64.StdEncoding.DecodeString(Test_msg)
	if string(decoded) != string(decoded2) {
		t.Error("Encoded not as etalon")
	}
	fmt.Println(m.String())
}

func TestMsgline(t *testing.T) {
	var m *Msg
	m = DecodeMsgline(`test.area
All
hello world!

@repto: 12345678901234567890
This is my
message!

wbr, Anonymous!
`, false)
	if m == nil {
		t.Error("Can not decode msg")
	}
}
func TestMake(t *testing.T) {
	m := Msg{
		Tags: NewTags("ii/ok/repto/aaaaa"),
		Echo: "test.echo",
		Text: "Hello world!",
	}
	msg := m.Encode()
	if msg == "" {
		t.Error("Can not encode msg")
	}
	m2, _ := DecodeBundle(msg)
	if m2 == nil {
		t.Error("Can not decode encoded msg")
	}
}
