<!DOCTYPE html>
<head>
<meta name="Robots" content="index,follow">
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width; initial-scale=1.0">
<link rel="icon" href="/lib/icon.png" type="image/png">
<link rel="stylesheet" type="text/css" href="/lib/style.css">
<title>go-ii</title>
</head>
<body>
<div id="body">
<table id="header">
  <tr>
    <td class="title">
      <span class="logo"><a href="/">ii-go</a></span>
      <span class="info">II/IDEC networks {{ with .Echo }} :: {{.}}{{end}}
</span>
    </td>
    <td class="links">
      <span>
      {{ with .User }}
      {{.Name}}
      {{end}}
      {{ with .Echo }}
      :: <a href="/{{.}}/new">New topic</a>
      {{ end }}
      </span>
    </td>
</tr>
</table>
