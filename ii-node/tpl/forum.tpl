{{template "header.tpl" $}}
{{template "pager.tpl" $}}

<table id="echolist" cellspacing=0 cellpadding=0>
<tr class="title">
<th>Echo</th>
<th class="extra">Topics</th>
<th class="extra">Posts</th>
<th>Last</th>
</tr>
{{range $k, $_ := .Echoes }}
{{ if is_even $k }}
<tr class="even">
{{ else }}
<tr class="odd">
{{ end }}
<td class="echo"><a href="{{$.PfxPath}}/{{.Name}}/">{{.Name}}</a><br>
<span class="info">{{ index $.Echolist.Info .Name }}</span>
</td>
<td class="topics extra">{{.Topics}}</td>
<td class="count extra">{{.Count}}</td>
<td class="info">{{with .Msg}}<span class="subj">{{.Subj}}</span><br><a href="{{$.PfxPath}}/echo/{{.MsgId}}#{{.MsgId}}">{{.Date | fdate}}</a> by {{.From}}{{end}}</td>
</tr>
{{ end }}
</table>

{{template "pager.tpl" $}}
{{template "footer.tpl"}}
