{{template "header.tpl" $}}
{{ $odd := false }}
{{if .User.Name }}
<a href="/{{.BasePath}}/new">New topic</a><br>
{{end}}
{{template "pager.tpl" $}}
<table id="topiclist" cellspacing=0 cellpadding=0>
<tr class="title">
<th>Topics</th>
<th>Posts</th>
<th>Last post</th>
</tr>
{{range .Topics }}
{{ if $odd }}
<tr class="odd">
{{ else }}
<tr class="even">
{{ end }}
<td class="topic"><a href="/{{.Head.MsgId}}">{{.Head.Subj}}</a></td>
<td class="posts">{{.Count}}</td>
<td class="info"><a href="/{{.Tail.MsgId}}#{{.Tail.MsgId}}">{{.Tail.Date | fdate}}</a><br>by {{.Tail.From}}</td>
</tr>
{{ $odd = not $odd }}
{{ end }}
</table>


{{template "footer.tpl"}}
