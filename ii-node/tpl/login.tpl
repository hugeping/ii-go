{{template "header.tpl"}}
<form method="post" enctype="multipart/form-data" action="/login">
<input type="text" name="username" class="login" placeholder="username"><br>
<input type="password" name="password" class="passwd" placeholder="password"><br>
<button class="form-button">Login</button>
{{template "footer.tpl"}}
