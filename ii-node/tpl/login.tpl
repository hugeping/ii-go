{{template "header.tpl" $}}

<form method="post" enctype="application/x-www-form-urlencoded" action="/login">
<table id="login" cellspacing=0 cellpadding=0>

<tr class="odd"><td>
<input type="text" name="username" class="login" placeholder="username">
</td></tr>

<tr class="even"><td>
<input type="password" name="password" class="passwd" placeholder="password">
</td></tr>

<tr class="odd"><td class="links" colspan="2">
<button class="form-button">Login</button>
</td></tr>

</table>
</form>

{{template "footer.tpl"}}
