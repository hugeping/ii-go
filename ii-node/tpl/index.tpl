{{ if eq $.PfxPath "/forum" }}
{{ template "forum.tpl" $ }}
{{ else }}
{{ template "digest.tpl" $ }}
{{ end }}
