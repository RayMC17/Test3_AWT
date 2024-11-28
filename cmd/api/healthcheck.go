package main

import (
	"net/http"
)

// healthCheckHandler responds with basic information about the application status
func (a *applicationDependencies) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Define the data to be returned in the response
	data := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": a.config.environment,
			"version":     appVersion,
		},
	}

	// Write JSON response with 200 OK status
	err := a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}
