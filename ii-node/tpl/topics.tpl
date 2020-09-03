{{template "header.tpl" $}}
{{if .User.Name }}
<a href="/{{.BasePath}}/new">New topic</a><br>
{{end}}
{{template "pager.tpl" $}}
<table class="topiclist">
<tr>
<th>Topics</th>
<th>Posts</th>
<th>Last post</th>
</tr>
{{range .Topics }}
<tr>
<td><a href="/{{.Head.MsgId}}">{{.Head.Subj}}</a></td>
<td>{{.Count}}</td>
<td>{{.Tail.Date | fdate}} {{.Tail.From}}</td>
</tr>
{{ end }}
</table>


{{template "footer.tpl"}}
