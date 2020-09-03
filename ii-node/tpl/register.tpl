{{template "header.tpl"}}
<form method="post" enctype="application/x-www-form-urlencoded" action="/register">
<input type="text" name="username" class="login" placeholder="username"><br>
<input type="password" name="password" class="passwd" placeholder="password"><br>
<input type="text" name="email" class="email" placeholder="email"><br>
<button class="form-button">Submit</button>
</form>
{{template "footer.tpl"}}
