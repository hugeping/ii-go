{{template "header.tpl" $}}
<a class="rss" href="{{.PfxPath}}/echo/{{.Echo}}">Echo</a> :: <a class="rss" href="{{.PfxPath}}/forum/{{.Echo}}">Forum</a> :: <a class="rss" href="{{.PfxPath}}/blog/{{.Echo}}">Blog</a>  :: <a class="rss" href="{{.PfxPath}}/echo/{{.Echo}}/rss">RSS</a>
{{template "pager.tpl" $}}

<div id="topic">
{{ range $k, $_ := .Topics }}

<span class="title"><a href="{{$.PfxPath}}/{{.Head.MsgId}}#{{.Head.MsgId}}">{{ .Head.Subj }} [{{ .Count }}]</a></span><br>
{{ with .Tail }}
<div class="msg">
{{ if and (msg_local .) (has_avatar .From)}}
<img class="avatar" src="/avatar/{{.From}}">
{{ end }}
<span class="subj"> <a href="{{$.PfxPath}}/{{.MsgId}}#{{.MsgId}}">{{with .Subj}}{{.}}{{else}}No subject{{end}}</a></span><br>
<!-- <span class="echo"><a href="{{$.PfxPath}}/echo/{{ .Echo }}">{{.Echo}}</a></span><br> -->
<span class="info">{{.From}}({{.Addr}}) &mdash; {{.To}}<br>{{.Date | fdate}}</span><br>
<div class="text">
<br>
{{$more := (print " ... <a class='more' href='" $.PfxPath "/" .MsgId "#" .MsgId "'>[&gt;&gt;&gt;]</a>")}}
{{msg_trunc . 2048 $more}}
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
