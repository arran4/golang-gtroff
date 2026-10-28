package manlib

const libraryTemplate = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>{{.Title}}</title>
<style>body{margin:0;background:#ffffff;color:#202124;font:18px/1.5 sans-serif}body>header,body>main{max-width:1100px;margin:auto;padding:1rem}a{color:#174ea6}input,button{font:inherit}input{max-width:65vw}nav ul{padding-left:1.5em}.subsection{margin-left:1em}.notice{border-left:3px solid #8c3700;padding-left:1em}.results li{margin:.5em 0}.source{font-size:.85em;color:#555}.man-page{border-top:1px solid #ccc}</style>
</head><body><header><a href="/">Man library</a><form action="/search" method="get"><label>Search <input type="search" name="q" value="{{.Query}}" maxlength="256"></label> <button type="submit">Search</button></form><p>Read-only source snapshot. Unsupported syntax is shown with conversion notes.</p></header>
<main>{{range .Notices}}<p class="notice">{{.}}</p>{{end}}{{if .Collection}}<details><summary>Collection notices</summary><ul>{{range .Collection}}<li>{{.Root}}:{{.Path}} — {{.Message}}</li>{{end}}</ul></details>{{end}}
{{if .Rendered}}{{if .Headings}}<nav aria-label="Page sections"><ul>{{range .Headings}}<li{{if .Subsection}} class="subsection"{{end}}><a href="{{.URL}}">{{.Text}}</a></li>{{end}}</ul></nav>{{end}}{{.Rendered}}{{else}}
<h1>Man library</h1><p>{{.Summary}}</p><ul class="results">{{range .Hits}}<li>{{if .URL}}<a href="{{.URL}}">{{.Name}}({{.Section}})</a>{{else}}{{if .Name}}{{.Name}}({{.Section}}){{else}}System result{{end}}{{end}}{{if .Description}} — {{.Description}}{{end}} <span class="source">[{{.Root}}{{if .External}}, system search{{end}}]</span></li>{{end}}</ul>
<nav aria-label="Result pages">{{if .Previous}}<a href="{{.Previous}}">Previous</a>{{end}} {{if .Next}}<a href="{{.Next}}">Next</a>{{end}}</nav>{{end}}</main></body></html>
`
