{{template "header.tpl" $}}

<form method="post" enctype="application/x-www-form-urlencoded" action="/{{.Echo}}/new">
<input type="text" name="subj" class="subj" placeholder="subject"><br>
<input type="text" name="to" class="to" placeholder="To" value="All"><br>
<textarea type="text" name="msg" class="message" cols=60 row=10 placeholder="Text here."></textarea><br>
<br>
<button class="form-button">Submit</button>
</form>

{{template "footer.tpl"}}
