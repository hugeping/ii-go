{{template "header.tpl" $}}
{{ $msg := index .Msg 0 }}
{{ with $msg }}
<table id="edit">
<form method="post" enctype="application/x-www-form-urlencoded" action="{{$.PfxPath}}/{{.MsgId}}/edit">
<tr><td class="odd">
<input type="hidden" name="id" value="{{.MsgId}}">
<input type="hidden" name="repto" value="{{ . | repto}}">
<input type="text" name="echo" class="echo" placeholder="{{.Echo}}" value="{{.Echo}}"><br>
<input type="text" name="to" class="to" placeholder="{{.To}}" value="{{.To}}"><br>
<input type="text" name="subj" class="subj" placeholder="{{.Subj}}" value="{{.Subj}}"><br>
<textarea type="text" name="msg" class="message" cols=60 row=16 placeholder="Text here.">
{{.Text}}
</textarea>
</td></tr>

<tr><td class="odd center">
<button class="form-button" type="submit" name="action" value="Submit">Submit</button>
<button class="form-button" type="submit" name="action" value="Preview">Preview</button>
</td></tr>
</form>

</table>
{{ end }}
{{template "footer.tpl"}}
