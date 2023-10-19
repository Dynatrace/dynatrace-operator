package utils

import "net/http"

func CloseBodyAfterRequest(response *http.Response) {
	if response != nil && response.Body != nil {
		response.Body.Close()
	}
}
