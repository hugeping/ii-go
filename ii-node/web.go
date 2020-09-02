package main

import (
	"../ii"
	"errors"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"
)

type WebContext struct {
	Echoes []ii.Echo
	Topics []Topic
	Msg    []ii.Msg
	Render func(string) template.HTML
}

func www_index(www WWW, w http.ResponseWriter, r *http.Request) error {
	var ctx WebContext
	ii.Trace.Printf("www index")
	ctx.Echoes = www.db.Echoes(nil)
	//	ctx.Msg = make([]*ii.Msg, len(ctx.Echoes))
	// for k, e := range ctx.Echoes {
	// 	ctx.Msg[k] = db.Get(e.Last.Id)
	// }
	err := www.tpl.ExecuteTemplate(w, "index.tpl", ctx)
	return err
}

func getParent(db *ii.DB, i *ii.MsgInfo) *ii.MsgInfo {
	return db.Lookup(i.Repto)
}

func getTopics(db *ii.DB, mi []*ii.MsgInfo) map[string][]string {
	intopic := make(map[string]string)
	topics := make(map[string][]string)
	for _, m := range mi {
		if _, ok := intopic[m.Id]; ok {
			continue
		}
		var l []string
		for p := m; p != nil; p = getParent(db, p) {
			l = append(l, p.Id)
		}
		t := l[len(l) - 1]
		for _, id := range l {
			topics[t] = append(topics[t], id)
			intopic[id] = t
		}
	}
	return topics
}

type Topic struct {
	Ids     []string
	Count   int
	Date    string
	LastUpd string
	Head    *ii.Msg
	Last    *ii.Msg
}

func www_topics(www WWW, w http.ResponseWriter, r *http.Request, echo string) error {
	db := www.db
	var ctx WebContext
	mis := db.LookupIDS(db.SelectIDS(ii.Query{Echo: echo}))
	ii.Trace.Printf("www topics: %s", echo)
	topics := getTopics(db, mis)
	ii.Trace.Printf("Start to generate topics")
	for _, t := range topics {
		//		if mi.Repto != "" || db.Exists(mi.Repto) != nil {
		//	continue
		//}
		topic := Topic{}
		topic.Ids = t
		m := db.Get(topic.Ids[0])
		if m == nil {
			ii.Error.Printf("Skip wrong message: %s\n", topic.Ids[0])
			continue
		}
		topic.Count = len(topic.Ids) - 1
		topic.Head = m
		topic.Last = db.Get(topic.Ids[topic.Count])
		topic.Date = time.Unix(topic.Last.Date, 0).Format("2006-01-02 15:04:05")
		ctx.Topics = append(ctx.Topics, topic)
	}
	sort.SliceStable(ctx.Topics, func(i, j int) bool {
		return ctx.Topics[i].Last.Date > ctx.Topics[j].Last.Date
	})
	ii.Trace.Printf("Stop to generate topics")
	err := www.tpl.ExecuteTemplate(w, "topics.tpl", ctx)
	return err
}
func msg_format(txt string) template.HTML {
	txt = strings.Replace(txt, "&", "&amp;", -1)
	txt = strings.Replace(txt, "<", "&lt;", -1)
	txt = strings.Replace(txt, ">", "&gt;", -1)
	return template.HTML(strings.Replace(txt, "\n", "<br/>", -1))
}
func www_topic(www WWW, w http.ResponseWriter, r *http.Request, id string) error {
	db := www.db
	var ctx WebContext
	ctx.Render = msg_format
	mi := db.Lookup(id)
	if mi == nil {
		return errors.New("No such message")
	}
	mis := db.LookupIDS(db.SelectIDS(ii.Query{Echo: mi.Echo, Repto: ""}))
	ids := getTopics(db, mis)[id]
	ii.Trace.Printf("www topic: %s", id)
	for _, i := range ids {
		m := db.Get(i)
		if m == nil {
			ii.Error.Printf("Skip wrong message: %s", i)
			continue
		}
		ctx.Msg = append(ctx.Msg, *m)
	}
	err := www.tpl.ExecuteTemplate(w, "topic.tpl", ctx)
	return err
}

func Web(www WWW, w http.ResponseWriter, r *http.Request) error {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		return www_index(www, w, r)
	}
	if ii.IsEcho(path) {
		return www_topics(www, w, r, path)
	}
	if ii.IsMsgId(path) {
		return www_topic(www, w, r, path)
	}
	return nil
}
