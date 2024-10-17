{{template "header.tpl" $}}

<div id="topic">
<div class="msg">
<div class="text">
Для активации учётной записи напишите письмо на {{ $.Admin.Mail }}<br>
В теме письма напишите: регистрация на {{ $.Sysname }}<br>
<hr/>
To activate account send e-mail to {{ $.Admin.Mail }}<br>
In the subject of the e-mail write: {{ $.Sysname }} registration<br>
</div>
</div>
</div>
{{template "pager.tpl" $}}

{{template "footer.tpl"}}
