{{template "header.tpl" $}}
{{template "pager.tpl" $}}
<div id="topic">
{{ range .Msg }}
{{if eq $.Selected .MsgId }}
<a name="{{.MsgId}}"></a>
<div class="msg selected">
{{else}}
<div class="msg">
{{end}}
<a class="msgid" href="/{{.MsgId}}#{{.MsgId}}">#</a><span class="subj"> <a href="/{{. | repto}}#{{. | repto}}">{{with .Subj}}{{.}}{{else}}No subject{{end}}</a></span><br>
<span class="info">{{.From}}({{.Addr}}) &mdash; {{.To}}<br>{{.Date | fdate}}</span><br>
<div class="text">
<br>
{{with .Text}}
{{. | msg_format}}
{{end}}
<br>
{{if $.User.Name}}
<span class="reply"><a href="/{{.MsgId}}/reply">Reply</a></span>
{{end}}
{{ if msg_access . $.User }}
 :: <span class="reply"><a href="/{{.MsgId}}/edit">Edit</a></span>
{{ end }}
{{if $.User.Name}}
<br>
{{end}}
</div>
</div>
{{ end }}
</div>
{{template "pager.tpl" $}}

{{template "footer.tpl"}}
