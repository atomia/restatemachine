package main

import (
	"github.com/emicklei/go-restful"
	"encoding/base64"
	"io"
	"strings"
)

func initApi() {

	ws := new(restful.WebService)
	filter := noAuthentication
	if globalConfig.Username != "" && globalConfig.Password != "" {
		filter = basicAuthenticate
	}

	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/").Filter(filter).To(apiUsage))
	ws.Route(ws.GET("/machines").Filter(filter).To(apiListMachines))
	ws.Route(ws.GET("/machines/{name}").Filter(filter).To(apiGetMachine))
	restful.Add(ws)
}

func basicAuthenticate(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	encoded := req.Request.Header.Get("Authorization")

	unAuthorized := func() {
		resp.AddHeader("WWW-Authenticate", "Basic realm=Protected Area")
		resp.WriteErrorString(401, "401: Not Authorized")
	}

	auth := strings.SplitN(encoded, " ", 2)
	if len(auth) != 2 || auth[0] != "Basic" {
		unAuthorized()
		return
        }
 
	decoded_auth, _ := base64.StdEncoding.DecodeString(auth[1])
	credentials := strings.SplitN(string(decoded_auth), ":", 2)

	if len(credentials) != 2 || credentials[0] != globalConfig.Username || credentials[1] != globalConfig.Password {
		unAuthorized()
		return
	}
	chain.ProcessFilter(req, resp)
}

func noAuthentication(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	chain.ProcessFilter(req, resp)
}

func apiUsage(req *restful.Request, resp *restful.Response) {
	io.WriteString(resp, "Usage goes here\n")
}

func apiListMachines(req *restful.Request, resp *restful.Response) {
	resp.WriteEntity(globalStateMachines)
}

func apiGetMachine(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	for _, machine := range globalStateMachines {
		if machine.Name == name {
			resp.WriteEntity(machine)
			return
		}
	}

	resp.WriteErrorString(404, "404: State machine not found")
}
