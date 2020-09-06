{{template "header.tpl" $}}
{{template "pager.tpl" $}}
<a class="rss" href="/{{.BasePath}}/rss">RSS</a>
<div id="topic">
{{ range .Msg }}
<div class="msg">
<a class="msgid" href="/{{.MsgId}}#{{.MsgId}}">#</a><span class="subj"> <a href="/{{. | repto}}#{{. | repto}}">{{with .Subj}}{{.}}{{else}}No subject{{end}}</a></span><br>
<span class="echo"><a href="/{{.Echo}}">{{.Echo}}</a></span><br>
<span class="info">{{.From}}({{.Addr}}) &mdash; {{.To}}<br>{{.Date | fdate}}</span><br>
<div class="text">
<br>
{{with .Text}}
{{. | msg_format}}
{{end}}
<br>
</div>
</div>
{{ end }}
</div>
{{template "pager.tpl" $}}

{{template "footer.tpl"}}
