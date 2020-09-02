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
	Msg    []*ii.Msg
	Topics []*Topic
	Render func(string) template.HTML
}

func www_index(db *ii.DB, t *template.Template, w http.ResponseWriter, r *http.Request) error {
	var ctx WebContext
	ii.Trace.Printf("www index")
	ctx.Echoes = db.Echoes(nil)
	err := t.ExecuteTemplate(w, "index.tpl", ctx)
	return err
}

func getTopicFor(head *ii.MsgInfo, mi []*ii.MsgInfo) []string {
	var ids []string
	hash := make(map[string]bool)

	hash[head.Id] = true
	hit := true
	ids = append(ids, head.Id)
	for hit {
		hit = false
		for _, m := range mi {
			if _, ok := hash[m.Id]; ok {
				continue
			}
			if _, ok := hash[m.Repto]; ok {
				hash[m.Id] = true
				hit = true
				ids = append(ids, m.Id)
			}
		}
	}
	return ids
}

type Topic struct {
	Ids     []string
	Count   int
	Date    string
	LastUpd string
	Head    *ii.Msg
	Last    *ii.Msg
}

func www_topics(db *ii.DB, echo string, t *template.Template, w http.ResponseWriter, r *http.Request) error {
	var ctx WebContext
	mis := db.LookupIDS(db.SelectIDS(ii.Query{Echo: echo}))
	ii.Trace.Printf("www topics: %s", echo)
	for _, mi := range mis {
		if mi.Repto != "" || db.Exists(mi.Repto) != nil {
			continue
		}
		topic := Topic{}
		topic.Ids = getTopicFor(mi, mis)
		m := db.Get(topic.Ids[0])
		if m == nil {
			ii.Error.Printf("Skip wrong message: %s\n", mi.Id)
			continue
		}
		topic.Count = len(topic.Ids) - 1
		topic.Head = m
		topic.Last = db.Get(topic.Ids[topic.Count])
		topic.Date = time.Unix(topic.Last.Date, 0).Format("2006-01-02 15:04:05")
		ctx.Topics = append(ctx.Topics, &topic)
	}
	sort.SliceStable(ctx.Topics, func(i, j int) bool {
		return ctx.Topics[i].Last.Date > ctx.Topics[j].Last.Date
	})
	err := t.ExecuteTemplate(w, "topics.tpl", ctx)
	return err
}
func msg_format(txt string) template.HTML {
	txt = strings.Replace(txt, "&", "&amp;", -1)
	txt = strings.Replace(txt, "<", "&lt;", -1)
	txt = strings.Replace(txt, ">", "&gt;", -1)
	return template.HTML(strings.Replace(txt, "\n", "<br/>", -1))
}
func www_topic(db *ii.DB, id string, t *template.Template, w http.ResponseWriter, r *http.Request) error {
	var ctx WebContext
	ctx.Render = msg_format
	mi := db.Lookup(id)
	if mi == nil {
		return errors.New("No such message")
	}
	mis := db.LookupIDS(db.SelectIDS(ii.Query{Echo: mi.Echo, Repto: ""}))
	ids := getTopicFor(mi, mis)
	ii.Trace.Printf("www topic: %s", id)
	for _, i := range ids {
		m := db.Get(i)
		if m == nil {
			ii.Error.Printf("Skip wrong message: %s", i)
			continue
		}
		ctx.Msg = append(ctx.Msg, m)
	}
	err := t.ExecuteTemplate(w, "topic.tpl", ctx)
	return err
}

func Web(db *ii.DB, t *template.Template, w http.ResponseWriter, r *http.Request) error {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		return www_index(db, t, w, r)
	}
	if ii.IsEcho(path) {
		return www_topics(db, path, t, w, r)
	}
	if ii.IsMsgId(path) {
		return www_topic(db, path, t, w, r)
	}
	return nil
}
