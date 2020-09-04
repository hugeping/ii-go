{{template "header.tpl" $}}

<table id="error" cellspacing=0 cellpadding=0>
<tr class="alert"><td>Error!</td></tr>
<tr class="even"><td>{{.Error}}</td></tr>
<tr class="odd"><td class="links"><a href="/">Ok</a></td></tr>
</table>

{{template "footer.tpl"}}
