{{template "header.tpl" $}}
<form method="post" enctype="application/x-www-form-urlencoded" action="/register">
<table id="login" cellspacing=0 cellpadding=0>

<tr class="odd"><td>
<input type="text" name="username" class="login" placeholder="username"><br>
</td></tr>

<tr class="even"><td>
<input type="password" name="password" class="passwd" placeholder="password"><br>
</td></tr>

<tr><td class="odd">
<input type="text" name="email" class="email" placeholder="email">
</td></tr>

<tr class="even"><td class="links">
<button class="form-button">Register</button>
</td></tr>

</table>

</form>
{{template "footer.tpl"}}
