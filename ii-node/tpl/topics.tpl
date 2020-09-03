{{template "header.tpl"}}
<a href="/{{.Echo}}/1">1</a><a href="/{{.Echo}}/{{.Pages}}">{{.Pages}}</a>
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
<td>{{.Date}}</td>
</tr>
{{ end }}
</table>


{{template "footer.tpl"}}
