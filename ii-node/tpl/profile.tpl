{{template "header.tpl" $}}

<table id="profile" cellspacing=0 cellpadding=0>
{{if has_avatar .User.Name}}<img class="avatar" src="/avatar/{{.User.Name}}">{{end}}
<tr class="odd"><td>Login:</td><td>{{.User.Name}}</td></tr>
<tr class="even"><td>Auth:</td><td>{{.User.Secret}}</td></tr>
<tr class="odd"><td>e-mail:</td><td>{{.User.Mail}}</td></tr>
<tr class="even"><td>Addr:</td><td>{{.Selected}}</td></tr>
<tr class="odd"><td class="links" colspan="2"><a href="{{.PfxPath}}/from/{{.User.Name}}">/from/{{.User.Name}}</a> :: <a href="{{.PfxPath}}/to/{{.User.Name}}">/to/{{.User.Name}}</a>
</td></tr>

<tr><td class="even" colspan="2">
<form method="post" enctype="application/x-www-form-urlencoded" action="{{.PfxPath}}/avatar/{{.User.Name}}">
<textarea type="text" name="avatar" class="message" cols=60 row=16 placeholder="Add XPM avatar here">{{.Info}}</textarea>
</td></tr>
<tr><td class="odd center" colspan="2">
<button class="form-button" type="submit" name="action" value="Submit">Submit</button>
</td></tr>
</form>

</table>
{{template "footer.tpl"}}
