{{template "header.tpl" $}}
{{range .Msg}}
<form method="post" enctype="application/x-www-form-urlencoded" action="/{{.MsgId}}/reply">
<input type="text" name="subj" class="subj" placeholder="subject" value="{{.Subj}}"><br>
<input type="text" name="to" class="to" placeholder="To" value="{{.From}}"><br>
<textarea type="text" name="msg" class="message" cols=60 row=10 placeholder="Text here.">{{.Text}}</textarea><br>
<br>
<button class="form-button">Submit</button>
</form>
{{end}}
{{template "footer.tpl"}}
