{{ range .Pager }}
{{ if eq . $.Page }}
[{{.}}]
{{ else }}
<a href="/{{$.BasePath}}/{{.}}">{{.}}</a>
{{ end }}
{{ end }}
