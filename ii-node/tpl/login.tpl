{{template "header.tpl"}}

<form method="post" enctype="application/x-www-form-urlencoded" action="/login">
<table id="login" cellspacing=0 cellpadding=0>

<tr class="odd"><td>
<input type="text" name="username" class="login" placeholder="username">
</td></tr>

<tr class="even"><td>
<input type="password" name="password" class="passwd" placeholder="password">
</td></tr>

<tr><td class="links odd" colspan="2">
<button>Login</button>
</td></tr>

</table>
</form>

{{template "footer.tpl"}}
