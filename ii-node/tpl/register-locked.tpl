{{template "header.tpl" $}}

<div id="topic">
<div class="msg">
<div class="text">
Автоматическая регистрация временно закрыта.<br>
Для получения учётной записи пишите на {{ $.Admin.Mail }}<br>
<hr/>
Automatic registration is temporarily closed.<br>
To get an account write message on {{ $.Admin.Mail }}<br>
</div>
</div>
</div>
{{template "pager.tpl" $}}

{{template "footer.tpl"}}
