{{template "header.tpl" $}}
<table id="edit">
<form method="post" enctype="application/x-www-form-urlencoded" action="/{{.Echo}}/new">
<tr><td class="even">
<input type="text" name="to" class="to" placeholder="To" value="All"><br>
<input type="text" name="subj" class="subj" placeholder="Subject"><br>
<textarea type="text" name="msg" class="message" cols=60 row=16 placeholder="Hi, All!">
</textarea>
</td></tr>

<tr><td class="odd center">
<button class="form-button" type="submit" name="action" value="Submit">Submit</button>
<button class="form-button" type="submit" name="action" value="Preview">Preview</button>
</td></tr>
</form>

</table>
{{template "footer.tpl"}}
