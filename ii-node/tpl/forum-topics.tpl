{{template "header.tpl" $}}
{{template "pager.tpl" $}}
<table id="topiclist" cellspacing=0 cellpadding=0>
<tr class="title">
<th>Topics</th>
<th class="extra">Replies</th>
<th>Last post</th>
</tr>
{{range $k, $v := .Topics }}
{{ if is_even $k }}
<tr class="even">
{{ else }}
<tr class="odd">
{{ end }}
<td class="topic"><a href="/{{.Head.MsgId}}/1">{{with .Head.Subj}}{{.}}{{else}}No subject{{end}}</a></td>
<td class="posts extra">{{.Count}}</td>
<td class="info"><span class="subj">{{.Tail.Subj}}</span><br><a href="/{{.Tail.MsgId}}#{{.Tail.MsgId}}">{{.Tail.Date | fdate}}</a><br>by {{.Tail.From}}</td>
</tr>
{{ end }}
</table>
{{template "pager.tpl" $}}
{{template "footer.tpl"}}
