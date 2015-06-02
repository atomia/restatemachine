package main

import (
	"encoding/base64"
	"github.com/emicklei/go-restful"
	"io"
	"io/ioutil"
	"strings"
)

func initApi() {

	ws := new(restful.WebService)
	filter := noAuthentication
	if globalConfig.Username != "" && globalConfig.Password != "" {
		filter = basicAuthenticate
	}

	ws.Consumes(restful.MIME_OCTET).Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/").Filter(filter).To(apiUsage))
	ws.Route(ws.GET("/machines").Filter(filter).To(apiListMachines))
	ws.Route(ws.GET("/machines/{name}").Filter(filter).To(apiGetMachine))
	ws.Route(ws.GET("/runs").Filter(filter).To(apiListCurrentRuns))
	ws.Route(ws.GET("/runs/{id}").Filter(filter).To(apiGetRun))
	ws.Route(ws.POST("/runs/{machine}").Filter(filter).To(apiRunMachine))
	ws.Route(ws.DELETE("/runs/{id}").Filter(filter).To(apiDeleteRun))
	restful.Add(ws)
}

func basicAuthenticate(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	encoded := req.Request.Header.Get("Authorization")

	unAuthorized := func() {
		resp.AddHeader("WWW-Authenticate", "Basic realm=Protected Area")
		errorResponse(401, "Not authorized", resp)
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

func errorResponse(code int, message string, resp *restful.Response) {
	resp.WriteServiceError(code, restful.NewError(code, message))
}

func apiUsage(req *restful.Request, resp *restful.Response) {
	io.WriteString(resp, "Usage goes here\n")
}

func apiListMachines(req *restful.Request, resp *restful.Response) {
	resp.WriteEntity(globalStateMachines)
}

func apiGetMachine(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")

	if machine := machineGet(name); machine != nil {
		resp.WriteEntity(machine)
	} else {
		errorResponse(404, "State machine not found", resp)
	}
}

func apiListCurrentRuns(req *restful.Request, resp *restful.Response) {
	resp.WriteEntity(globalScheduler.GetRunningMachines())
}

func apiGetRun(req *restful.Request, resp *restful.Response) {
	id := req.PathParameter("id")
	machine, err := globalScheduler.GetMachineRun(id)
	if err != nil {
		errorResponse(500, "Error retrieving information about state machine run: "+err.Error(), resp)
	} else {
		resp.WriteEntity(machine)
	}
}

func apiRunMachine(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("machine")

	buffer, err := ioutil.ReadAll(req.Request.Body)
	if err != nil {
		errorResponse(500, "Error reading request body", resp)
		return
	}

	errCode, errMessage, executeResponse := machineExecute(name, string(buffer))
	if errMessage != "" {
		errorResponse(errCode, errMessage, resp)
	} else {
		resp.WriteEntity(executeResponse)
	}
}

func apiDeleteRun(req *restful.Request, resp *restful.Response) {
	type DeleteResponse struct {
		Message string
	}

	id := req.PathParameter("id")
	err := globalScheduler.CancelMachineRun(id)
	if err != nil {
		errorResponse(500, "Error cancelling state machine run", resp)
	} else {
		responseStruct := DeleteResponse{Message: "State machine run cancelled successfully"}
		resp.WriteEntity(responseStruct)
	}
}
