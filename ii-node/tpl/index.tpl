{{template "header.tpl" $}}
{{ $odd := false }}
<table id="echolist" cellspacing=0 cellpadding=0>
<tr class="title">
<th>Echo</th>
<th class="extra">Topics</th>
<th class="extra">Posts</th>
<th>Last</th>
</tr>
{{range .Echoes }}
{{ if $odd }}
<tr class="odd">
{{ else }}
<tr class="even">
{{ end }}
<td class="echo"><a href="{{.Name}}">{{.Name}}</a></td>
<td class="topics extra">{{.Topics}}</td>
<td class="count extra">{{.Count}}</td>
<td class="info">{{with .Msg}}<span class="subj">{{.Subj}}</span><br><a href="/{{.MsgId}}#{{.MsgId}}">{{.Date | fdate}}</a> by {{.From}}{{end}}</td>
</tr>
{{ $odd = not $odd }}
{{ end }}
</table>

{{template "footer.tpl"}}
