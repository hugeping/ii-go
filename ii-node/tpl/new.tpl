{{template "header.tpl" $}}
<form method="post" enctype="application/x-www-form-urlencoded" action="/{{.Echo}}/new">
<table id="edit"><tr><td class="even">
<input type="text" name="to" class="to" placeholder="To" value="All"><br>
<input type="text" name="subj" class="subj" placeholder="Subject"><br>
<textarea type="text" name="msg" class="message" cols=60 row=16 placeholder="Hi, All!"></textarea>
</td></tr><tr><td class="odd center">
<button class="form-button">Submit</button>
</td></tr></table>
</form>
{{template "footer.tpl"}}
