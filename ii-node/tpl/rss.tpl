<rss version="2.0">
<channel>
<title>{{.Topic}}</title>
<description>RSS feed with last messages</description>
<link>http://127.0.0.1:8080</link>

{{ range .Msg }}
<item>
	<title>{{.Subj}}</title>
	<guid>{{.MsgId}}</guid>
	<link>http://127.0.0.1:8080/{{.MsgId}}#{{.MsgId}}</link>
	<pubDate>{{.Date | fdate }}</pubDate>
	<description>{{.Text}}</description>
	<author>{{.From}}</author>
</item>
{{ end }}
</channel>
</rss>
