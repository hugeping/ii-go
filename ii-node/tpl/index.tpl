{{template "header.tpl"}}

<table class="echolist">
<tr>
<th>Name</th>
<th>Topics</th>
<th>Posts</th>
</tr>
{{range .Echoes }}
<tr>
<td><a href="{{.Name}}">{{.Name}}</a></td>
<td>{{.Topics}}</td>
<td>{{.Count}}</td>
</tr>
{{ end }}
</table>

{{template "footer.tpl"}}
