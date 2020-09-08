<!DOCTYPE html>
<head>
<meta name="Robots" content="index,follow">
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width; initial-scale=1.0">
<link rel="icon" href="/lib/icon.png" type="image/png">
<link rel="stylesheet" type="text/css" href="/lib/style.css">
<title>{{.Sysname}}</title>
</head>
<body>
<div id="body">
<table id="header">
  <tr>
    <td class="title">
      <span class="logo"><a href="/">{{.Sysname}}</a></span>
      {{ if eq .BasePath "" }}
      <span class="info">II/IDEC networks :: <a href="/echo/all">New posts</a>
      {{ else if gt (len .Topics) 0}}
      <span class="info">II/IDEC networks {{ with .Echo }} :: <a href="/echo/{{.}}">{{.}}</a> :: <span class="info">{{index $.Echolist.Info .}}</span>{{end}}
{{ else }}
      <span class="info">II/IDEC networks {{ with .Echo }} :: <a href="/{{.}}">{{.}}</a> :: <span class="info">{{index $.Echolist.Info .}}</span>{{end}}
{{ end }}
</span>
    </td>
    <td class="links">
      <span>
      {{ template "links.tpl" }}
      {{ if .User.Name }}
      {{ if eq .BasePath "profile" }}
      <a href="/logout">Logout</a>
      {{ else }}
      <a href="/profile">{{.User.Name}}</a>
      {{ end }}

      {{ with .Echo }}
      {{ if $.Topic }}
      :: <a href="/{{$.Topic}}/reply/new">New post</a>
      {{ else }}
      :: <a href="/{{.}}/new">New topic</a>
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
