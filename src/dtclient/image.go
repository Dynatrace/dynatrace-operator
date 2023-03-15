package dtclient

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

type LatestImageInfo struct {
	Source string `json:"source"`
	Tag    string `json:"tag"`
}

func (dtc *dynatraceClient) GetLatestOneAgentImage() (*LatestImageInfo, error) {
	latestImageInfo, err := dtc.processLatestImageRequest(dtc.getLatestOneAgentImageUrl())

	if err != nil {
		log.Info("failed to process latest image response")
		return nil, err
	}

	return latestImageInfo, nil
}

func (dtc *dynatraceClient) GetLatestCodeModulesImage() (*LatestImageInfo, error) {
	latestImageInfo, err := dtc.processLatestImageRequest(dtc.getLatestCodeModulesImageUrl())

	if err != nil {
		log.Info("failed to process latest image response")
		return nil, err
	}

	return latestImageInfo, nil
}

func (dtc *dynatraceClient) GetLatestActiveGateImage() (*LatestImageInfo, error) {
	latestImageInfo, err := dtc.processLatestImageRequest(dtc.getLatestActiveGateImageUrl())

	if err != nil {
		log.Info("failed to process latest image response")
		return nil, err
	}

	return latestImageInfo, nil
}

func (dtc *dynatraceClient) processLatestImageRequest(url string) (*LatestImageInfo, error) {
	request, err := dtc.createLatestImageRequest(url)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	response, err := dtc.httpClient.Do(request)
	if err != nil {
		log.Info("failed to retrieve latest image")
		return nil, err
	}

	defer CloseBodyAfterRequest(response)

	latestImageInfo, err := dtc.handleLatestImageResponse(response)
	if err != nil {
		log.Info("failed to handle latest image response")
		return nil, err
	}

	return latestImageInfo, nil
}

func (dtc *dynatraceClient) handleLatestImageResponse(response *http.Response) (*LatestImageInfo, error) {
	defer func() {
		err := response.Body.Close()
		if err != nil {
			log.Error(err, err.Error())
		}
	}()

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		return nil, dtc.handleErrorResponseFromAPI(data, response.StatusCode)
	}

	latestImageInfo, err := dtc.readResponseForLatestImage(data)
	if err != nil {
		return nil, err
	}
	return latestImageInfo, err
}

func (dtc *dynatraceClient) createLatestImageRequest(url string) (*retryablehttp.Request, error) {
	body := &LatestImageInfo{}

	bodyData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	request, err := createBaseRequest(
		url,
		http.MethodGet,
		dtc.apiToken,
		bytes.NewReader(bodyData),
	)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return request, nil
}

func (dtc *dynatraceClient) readResponseForLatestImage(response []byte) (*LatestImageInfo, error) {
	latestImageInfo := &LatestImageInfo{}
	err := json.Unmarshal(response, latestImageInfo)
	if err != nil {
		log.Error(err, "error unmarshalling LatestImageInfo", "response", string(response))
		return nil, err
	}

	return latestImageInfo, nil
}
