{{ range .Pager }}
{{ if eq . $.Page }}
<a href="/{{$.BasePath}}/{{.}}">[{{.}}]</a>
{{ else }}
<a href="/{{$.BasePath}}/{{.}}">{{.}}</a>
{{ end }}
{{ end }}
