{{ if eq $.PfxPath "/forum" }}
{{ template "forum-topics.tpl" $ }}
{{ else }}
{{ template "digest-topics.tpl" $ }}
{{ end }}
