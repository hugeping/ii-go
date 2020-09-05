{{template "header.tpl" $}}
{{range .Msg}}

<div id="topic">
<div class="msg">
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
<form method="post" enctype="application/x-www-form-urlencoded" action="/{{.Echo}}/new">

<tr><td class="even">
<input type="text" name="to" class="to" placeholder="{{.To}}" value="{{.To}}"><br>
<input type="text" name="subj" class="subj" placeholder="{{.Subj}}" value="{{.Subj}}"><br>
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
