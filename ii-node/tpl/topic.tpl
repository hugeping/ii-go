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
{{ if and (msg_local .) (has_avatar .From)}}
<img class="avatar" src="/avatar/{{.From}}">
{{ end }}
<span class="subj"> <a href="{{$.PfxPath}}/{{. | repto}}#{{. | repto}}">{{with .Subj}}{{.}}{{else}}No subject{{end}}</a></span>
{{ if eq $.User.Id 1 }}
<a class="blacklist" href="{{$.PfxPath}}/{{.MsgId}}/blacklist">blacklist</a>
{{ end }}
<br>
<!-- <span class="echo"><a href="{{$.PfxPath}}/{{ .Echo }}">{{.Echo}}</a></span><br> -->
<span class="info"><a href="{{$.PfxPath}}/from/{{.From}}">{{.From}}</a>({{.Addr}}) &mdash; {{.To}}<br>{{.Date | fdate}}</span><br>
<div class="text">
<br>
{{. | msg_text}}
<br>
{{if $.User.Name}}
<span class="reply"><a href="{{$.PfxPath}}/{{.MsgId}}/reply/new">Reply</a> :: </span>
<span class="reply"><a href="{{$.PfxPath}}/{{.MsgId}}/reply">Quote</a></span>
{{end}}
{{ if msg_access . $.User }}
 :: <span class="reply"><a href="{{$.PfxPath}}/{{.MsgId}}/edit">Edit</a></span>
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
