{{template "header.tpl" $}}

<table class="echolist">
<tr>
<th>Name</th>
<th>Topics</th>
<th>Posts</th>
<th>Last</th>
</tr>
{{range .Echoes }}
<tr>
<td><a href="{{.Name}}/-1">{{.Name}}</a></td>
<td>{{.Topics}}</td>
<td>{{.Count}}</td>
<td>{{with .Msg}}{{.Date | fdate}}[{{.From}}] {{.Subj}}{{end}}</td>
</tr>
{{ end }}
</table>

{{template "footer.tpl"}}
