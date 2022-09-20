package server

import (
	"Distributed-RPC-Framework/service"
	"fmt"
	"html/template"
	"net/http"
)

const debugText = `<html>
	<body>
	<title>Spartan RPC Services</title>
	{{range .}}
	<hr>
	Service {{.Name}}
	<hr>
		<table>
		<th align=center>Method</th><th align=center>Calls</th>
		{{range $name, $method := .Method}}
			<tr>
			<td align=left font=fixed>{{$name}}({{$method.InputType}}, {{$method.OutputType}}) error</td>
			<td align=center>{{$method.CallCounts}}</td>
			</tr>
		{{end}}
		</table>
	{{end}}
	</body>
	</html>`

var debug = template.Must(template.New("RPC - debug").Parse(debugText))

type HTTPDebug struct {
	*Server
}

type serviceDebug struct {
	Name   string
	Method map[string]*service.Method
}

// Runs at /debug/srpc
func (server HTTPDebug) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	// Build a sorted version of the data.
	var services []serviceDebug
	server.ServiceMap.Range(func(nameInterface, serviceInterface interface{}) bool {
		currentService := serviceInterface.(*service.Service)
		services = append(services, serviceDebug{
			Name:   nameInterface.(string),
			Method: currentService.ServiceMethod,
		})
		return true
	})
	err := debug.Execute(responseWriter, services)
	if err != nil {
		_, _ = fmt.Fprintln(responseWriter, "RPC: error when executing template:", err.Error())
	}
}
