{{template "header.tpl"}}

<table class="echolist">
<tr>
<th>Name</th>
<th>Topics</th>
<th>Posts</th>
<th>Last</th>
</tr>
{{range $k, $v := .Echoes }}
<tr>
<td><a href="{{$v.Name}}/-1">{{$v.Name}}</a></td>
<td>{{$v.Topics}}</td>
<td>{{$v.Count}}</td>
<td>{{with index $.Msg $k}}[{{.From}}] {{.Subj}}{{end}}</td>
</tr>
{{ end }}
</table>

{{template "footer.tpl"}}
