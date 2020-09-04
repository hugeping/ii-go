{{template "header.tpl" $}}

<table id="profile" cellspacing=0 cellpadding=0>
<tr class="odd"><td>Login:</td><td>{{.User.Name}}</td></tr>
<tr class="even"><td>Auth:</td><td>{{.User.Secret}}</td></tr>
<tr class="odd"><td>e-mail:</td><td>{{.User.Mail}}</td></tr>
<!-- <tr class="even"><td class="links" colspan="2"><a href="/logout">Logout</a>
</td></tr> -->
</table>
{{template "footer.tpl"}}
