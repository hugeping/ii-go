{{template "header.tpl" $}}
<form method="post" enctype="application/x-www-form-urlencoded" action="/register">
<table id="login" cellspacing=0 cellpadding=0>

<tr class="odd"><td>
<input type="text" name="auth" class="login" placeholder="authstr" value="{{.User.Secret}}"><br>
</td></tr>

<tr class="even"><td>
<input type="password" name="password" class="passwd" placeholder="password"><br>
</td></tr>

<tr class="odd"><td class="links">
<button class="form-button">Register</button>
</td></tr>

</table>

</form>
{{template "footer.tpl"}}
