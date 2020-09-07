{{template "header.tpl" $}}
<table id="edit">
{{ with index .Msg 0 }}
<form method="post" enctype="application/x-www-form-urlencoded" action="/{{.Echo}}/new">
<tr><td class="odd">
<input type="text" name="to" class="to" placeholder="{{.To}}" value="{{.To}}"><br>
<input type="text" name="subj" class="subj" placeholder="{{.Subj}}" value="{{.Subj}}"><br>
<input type="hidden" name="repto" value="{{.|repto}}">
<textarea type="text" name="msg" class="message" cols=60 row=16 placeholder="Enter text here.">{{.Text}}</textarea>
</td></tr>

<tr><td class="odd center">
<button class="form-button" type="submit" name="action" value="Submit">Submit</button>
<button class="form-button" type="submit" name="action" value="Preview">Preview</button>
</td></tr>
</form>
{{ end }}

</table>
{{template "footer.tpl"}}
