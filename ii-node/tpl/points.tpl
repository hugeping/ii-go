{{template "header.tpl" $}}

<table id="profile" cellspacing=0 cellpadding=0>
{{range $k, $_ := .Users.List }}
{{ if is_even $k }}
<tr class="even">
{{ else }}
<tr class="odd">
{{ end }}
{{ with index $.Users.Names . }}
<td>{{.Name}}</td>
<td>{{.Mail}}</td>
<td>{{user_tag .Name "info"}}
{{if user_tag .Name "status"}}
:: {{user_tag .Name "status"}}
{{end}}
{{if user_tag .Name "limit"}}
 Lim:{{user_tag .Name "limit"}}
{{end}}
</td>

<td>
{{ if eq (user_tag .Name "status") "new" }}
<a href="{{$.PfxPath}}/points/approve/{{.Name}}">Approve</a> |
<a href="{{$.PfxPath}}/points/moderate/{{.Name}}">Moderate</a> |
<a href="{{$.PfxPath}}/points/remove/{{.Name}}">Remove</a>
{{ else }}

{{ if eq (user_tag .Name "limit") "0" }}
<a href="{{$.PfxPath}}/points/unblock/{{.Name}}">Unblock</a>
{{ else }}
<a href="{{$.PfxPath}}/points/block/{{.Name}}">Block</a>
{{ end }}

{{ end }}
</td>

{{ end }}
</tr>
{{ end }}
</table>

{{template "footer.tpl"}}
