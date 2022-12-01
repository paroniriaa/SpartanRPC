package server

import (
	"Distributed-RPC-Framework/service"
	"fmt"
	"html/template"
	"net/http"
)

const serverInfoText = `<html>
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
	</html>
`

var serverInfo = template.Must(template.New("Spartan RPC - Server Info").Parse(serverInfoText))

type ServerInfoHTTP struct {
	*Server
	ServerConnectURL string
	ServerInfoURL    string
}

type serviceInfo struct {
	Name   string
	Method map[string]*service.Method
}

// ServeHTTP is the server HTTP endpoint that runs at DefaultServerInfoPath
func (server ServerInfoHTTP) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	// Build a sorted version of the data.
	var services []serviceInfo
	server.ServiceMap.Range(func(nameInterface, serviceInterface interface{}) bool {
		currentService := serviceInterface.(*service.Service)
		services = append(services, serviceInfo{
			Name:   nameInterface.(string),
			Method: currentService.ServiceMethod,
		})
		return true
	})
	err := serverInfo.Execute(responseWriter, services)
	if err != nil {
		_, _ = fmt.Fprintln(responseWriter, "RPC server info -> ServeHTTP: error when executing template:", err.Error())
	}
}
