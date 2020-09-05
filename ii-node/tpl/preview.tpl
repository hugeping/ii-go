{{template "header.tpl" $}}
{{ with index .Msg 0 }}

<div id="topic">
<div class="msg">
<span class="echo">{{.Echo}}</span><br>
<span class="subj">{{.Subj}}</span><br>
<span class="info">{{.From}}({{.Addr}}) &mdash; {{.To}}<br>{{.Date | fdate}}</span><br>
<div class="text">
<br>
{{with .Text}}
{{. | msg_format}}
{{end}}
<br>
</div>
</div>
</div>

<table id="edit">
{{ if eq $.Echo "" }}
<form method="post" enctype="application/x-www-form-urlencoded" action="/new">
{{ else }}
<form method="post" enctype="application/x-www-form-urlencoded" action="/{{.Echo}}/new">
{{ end }}
<tr><td class="even">
{{ if eq $.Echo "" }}
<input type="text" name="echo" class="echo" placeholder="{{.Echo}}" value="{{.Echo}}"><br>
{{ end }}
<input type="text" name="to" class="to" placeholder="{{.To}}" value="{{.To}}"><br>
<input type="text" name="subj" class="subj" placeholder="{{.Subj}}" value="{{.Subj}}"><br>
<input type="hidden" name="repto" value="{{ . | repto}}">
<textarea type="text" name="msg" class="message" cols=60 row=16 placeholder="Hi, All!">{{.Text}}</textarea>
</td></tr>

<tr><td class="odd center">
<button class="form-button" type="submit" name="action" value="Submit">Submit</button>
<button class="form-button" type="submit" name="action" value="Preview">Preview</button>
</td></tr>
</form>

</table>
{{end}}
{{template "footer.tpl"}}
