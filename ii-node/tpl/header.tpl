<!DOCTYPE html>
<head>
<meta name="Robots" content="index,follow">
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width; initial-scale=1.0">
<link rel="icon" href="/lib/icon.png" type="image/png">
<link rel="stylesheet" type="text/css" href="/lib/style.css">

{{ if eq .Template "query.tpl" }}
<link href="{{.PfxPath}}/{{.BasePath}}/rss" type="application/rss+xml" rel="alternate" title="{{.Sysname}} {{.BasePath}} :: RSS feed" />
{{ else if eq .Template "blog.tpl" }}
<link href="{{.PfxPath}}/{{.BasePath}}+topics/rss" type="application/rss+xml" rel="alternate" title="{{.Sysname}} {{.BasePath}} :: RSS feed" />
{{ else if eq .Template "topics.tpl" }}
<link href="{{.PfxPath}}/{{.Echo}}/rss" type="application/rss+xml" rel="alternate" title="{{.Sysname}} {{.Echo}} :: RSS feed" />
{{ else if eq .Template "index.tpl" }}
<link href="{{.PfxPath}}/echo/all/rss" type="application/rss+xml" rel="alternate" title="{{.Sysname}} Posts :: RSS feed" />
{{ end }}


<title>
{{ if eq .Template "new.tpl" }}
  {{ if eq .Echo "" }}
  New
  {{ else }}
  New: {{.Echo}}
  {{ end }}
{{ else if eq .Template "edit.tpl" }}
  Edit: {{(index .Msg 0).Subj}}
{{ else if eq .Template "reply.tpl" }}
  Reply: {{(index .Msg 0).Subj}}
{{ else if eq .Template "topics.tpl" }}
{{ .Echo }}
{{ else if eq .Template "query.tpl" }}
  {{ if eq .Echo "" }}
  Feed
  {{ else }}
  {{ .Echo }}
  {{ end }}
{{ else if eq .Template "topic.tpl" }}
  {{ if gt (len .Msg) (0) }}
  {{ $desc := (index .Msg 0).Subj }}
  {{.Echo}} / {{ $desc }}
  {{ else }}
  {{.Echo}} / ???
  {{ end }}
{{ else }}
{{.Sysname}}
{{ end }}
</title>
</head>
<body>
<div id="body">
<table id="header">
  <tr>
    <td class="title">
{{ if eq .Template "topics.tpl"}}
      <span class="logo"><a href="{{$.PfxPath}}/"><img class="logo" src="/lib/icon.png">{{.Sysname}}</a></span>
      <span class="info">II/IDEC networks {{ with .Echo }} :: <a href="{{$.PfxPath}}/{{.}}">{{.}}</a> :: <span class="info">{{index $.Echolist.Info .}}</span>{{end}}
{{ else if eq .Template "query.tpl" }}
      <span class="logo"><a href="{{$.PfxPath}}/"><img class="logo" src="/lib/icon.png">{{.Sysname}}</a></span>
      <span class="info">II/IDEC networks {{ with .Echo }} :: <a href="{{$.PfxPath}}/{{.}}">{{.}}</a> :: <span class="info">{{index $.Echolist.Info .}} / feed</span>{{end}}
{{ else if eq .Template "topic.tpl" }}
      <span class="logo"><a href="{{$.PfxPath}}/"><img class="logo" src="/lib/icon.png">{{.Sysname}}</a></span>
      {{ if gt (len .Msg) (0) }}
      {{ $desc := (index .Msg 0).Subj }}
      <span class="info">II/IDEC networks {{ with .Echo }} :: <a href="{{$.PfxPath}}/{{.}}">{{.}}</a> :: <span class="info">{{index $.Echolist.Info .}} / {{ $desc }}</span>{{end}}
      {{ end }}
{{ else }}
      <span class="logo"><a href="/"><img class="logo" src="/lib/icon.png">{{.Sysname}}</a></span>
      <span class="info">II/IDEC networks</span>
{{ end }}
</span>
    </td>
    <td class="links">
      <span>
      {{ template "links.tpl" }}
      {{ if and (eq .User.Id 1) (gt .Users.NewUsers 0) }}
      <span class="info">+{{.Users.NewUsers}} <a href="{{$.PfxPath}}/points">users</a> :: </span>
      {{ end }}
      {{ if .User.Name }}
      {{ if eq .BasePath "profile" }}
      <a href="/logout">Logout</a>
      {{ else }}
      <a href="/profile">{{.User.Name}}</a>
      {{ end }}

      {{ with .Echo }}
      {{ if $.Topic }}
      :: <a href="{{$.PfxPath}}/{{$.Topic}}/reply/new">New</a>
      {{ else }}
      :: <a href="{{$.PfxPath}}/{{.}}/new">New</a>
      {{ end }}
      {{ end }}

      {{ else if eq .BasePath "login" }}
      <a href="/register">Register</a>
      {{ else }}
      <a href="/login">Login</a>
      {{ end }}

      </span>
    </td>
</tr>
</table>
