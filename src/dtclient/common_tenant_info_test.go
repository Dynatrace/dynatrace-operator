package dtclient

import (
	"encoding/json"
	"net/http"
)

func tenantMalformedJson(url string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == url {
			writer.Write([]byte("this is not json"))
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
	}
}

func tenantInternalServerError(url string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == url {
			rawData, err := json.Marshal(serverErrorResponse{
				ErrorMessage: ServerError{
					Code:    http.StatusInternalServerError,
					Message: "error retrieving tenant info",
				}})
			writer.WriteHeader(http.StatusInternalServerError)

			if err == nil {
				_, _ = writer.Write(rawData)
			}
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
	}
}

func tenantServerHandler(url string, response interface{}) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == url {
			rawData, err := json.Marshal(response)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
			} else {
				writer.Header().Add("Content-Type", "application/json")
				_, _ = writer.Write(rawData)
			}
		} else {
			writer.WriteHeader(http.StatusBadRequest)
		}
	}
}
