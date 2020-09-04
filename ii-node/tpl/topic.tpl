{{template "header.tpl" $}}
{{template "pager.tpl" $}}

<div class="topic">
{{ range .Msg }}
{{if eq $.Selected .MsgId }}
<a name="{{.MsgId}}"></a>
<div class="msg selected">
{{else}}
<div class="msg">
{{end}}
<span class="msgid"><a href="/{{$.BasePath}}/reply">{{.MsgId}}</a></span><br>
<span class="msgsubj">{{.Subj}}</span>
<br/>
<span class="msginfo">{{.From}}({{.Addr}}) -> {{.To}}</span>
<br/>
<span class="msgtext">
{{with .Text}}
{{. | msg_format}}
{{end}}
</span>
</div>
{{ end }}
</div>

{{template "footer.tpl"}}
