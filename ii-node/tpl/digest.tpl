{{template "header.tpl" $}}
<a class="rss" href="{{$.PfxPath}}/forum/">Forum</a> :: <a class="rss" href="{{$.PfxPath}}/echo/all/">Feed</a>
{{template "pager.tpl" $}}

<div id="topic">
{{ range $k, $_ := .Echoes }}
{{ $count := .Count }}
{{ with .Msg }}


<span class="title"><a href="{{$.PfxPath}}/{{.Echo}}">{{.Echo}} :: {{ index $.Echolist.Info .Echo }} [{{ $count }}]</a></span><br>
<div class="msg">
{{ if and (msg_local .) (has_avatar .From)}}
<img class="avatar" src="/avatar/{{.From}}">
{{ end }}
<span class="subj"> <a href="{{$.PfxPath}}/echo/{{.MsgId}}#{{.MsgId}}">{{with .Subj}}{{.}}{{else}}No subject{{end}}</a></span><br>
<span class="info">{{.From}}({{.Addr}}) &mdash; {{.To}}<br>{{.Date | fdate}}</span><br>
<div class="text">
<br>
{{$more := (print " ... <a class='more' href='" $.PfxPath "/echo/" .Echo "/" .MsgId "#" .MsgId "'>[&gt;&gt;&gt;]</a>")}}
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
