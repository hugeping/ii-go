{{template "header.tpl" $}}
<a class="rss" href="{{.PfxPath}}/echo/{{.Echo}}">Echo</a> :: <a class="rss" href="{{.PfxPath}}/blog/{{.Echo}}">Blog</a> :: <a class="rss" href="{{.PfxPath}}/echo/{{.Echo}}/rss">RSS</a>
{{template "pager.tpl" $}}

<div id="topic">
{{ range $k, $_ := .Topics }}

<span class="title"><a href="{{$.PfxPath}}/{{.Head.MsgId}}#{{.Head.MsgId}}">{{ .Head.Subj }} [{{ .Count }}]</a></span><br>
{{ with .Tail }}
<div class="msg">
{{ if has_avatar .From }}
<img class="avatar" src="/avatar/{{.From}}">
{{ end }}
<span class="subj"> <a href="{{$.PfxPath}}/echo/{{.Echo}}/{{.MsgId}}#{{.MsgId}}">{{with .Subj}}{{.}}{{else}}No subject{{end}}</a></span><br>
<span class="echo"><a href="{{$.PfxPath}}/{{ .Echo }}">{{.Echo}}</a></span><br>
<span class="info">{{.From}}({{.Addr}}) &mdash; {{.To}}<br>{{.Date | fdate}}</span><br>
<div class="text">
<br>
{{ msg_text . }}
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
{{ end }}
</div>
{{template "pager.tpl" $}}

{{template "footer.tpl"}}
