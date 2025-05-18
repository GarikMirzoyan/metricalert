package utils

import (
	"html/template"
	"log"
)

func InitTemplate() *template.Template {
	const tmpl = `
	<!DOCTYPE html>
	<html>
	<head><title>Metrics</title></head>
	<body>
		<h1>Metrics</h1>
		<ul>
			{{range $key, $metric := .Gauges}}
				<li>{{$key}}: {{$metric.Value}}</li>
			{{end}}
			{{range $key, $metric := .Counters}}
				<li>{{$key}}: {{$metric.Value}}</li>
			{{end}}
		</ul>
	</body>
	</html>
	`
	t, err := template.New("metrics").Parse(tmpl)
	if err != nil {
		log.Fatalf("template parsing failed: %v", err)
	}
	return t
}
