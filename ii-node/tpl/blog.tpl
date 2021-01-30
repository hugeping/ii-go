{{template "header.tpl" $}}
{{template "pager.tpl" $}}
<a class="rss" href="{{.PfxPath}}/echo+topics/{{.Echo}}/rss">RSS</a>
<div id="topic">
{{range $k, $v := .Topics }}
{{ with .Head }}
<a name="{{.MsgId}}"></a>

<div class="msg">
{{ if has_avatar .From }}
<img class="avatar" src="/avatar/{{.From}}">
{{ end }}
<a class="msgid" href="/blog/{{.MsgId}}#{{.MsgId}}">#</a><span class="subj"> <a href="/blog/{{. | repto}}#{{. | repto}}">{{with .Subj}}{{.}}{{else}}No subject{{end}}</a></span><br>
<span class="info"><a href="{{$.PfxPath}}/from/{{.From}}">{{.From}}</a>({{.Addr}}) &mdash; {{.To}}<br>{{.Date | fdate}}</span><br>

<div class="text">
<br>
{{$more := (print "... <a class='more' href='" $.PfxPath "/" .MsgId "#" .MsgId "'>[ Read it &gt;&gt; ]</a>")}}
{{msg_trunc . 280 $more}}
<br>
{{ end }}
<span class="reply"><a href="/blog/{{.Tail.MsgId}}#{{.Tail.MsgId}}">{{.Count}} Replies</a></span>
</div>
</div>
{{ end }}
</div>

{{template "pager.tpl" $}}
{{template "footer.tpl"}}
