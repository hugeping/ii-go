{{template "header.tpl"}}
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
<td>{{.Date}}</td>
</tr>
{{ end }}
</table>


{{template "footer.tpl"}}
