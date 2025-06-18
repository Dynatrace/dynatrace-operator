package dynatrace

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

type LatestImageInfo struct {
	Source string `json:"source"`
	Tag    string `json:"tag"`
}

func (image LatestImageInfo) String() string {
	return image.Source + ":" + image.Tag
}

func (dtc *dynatraceClient) GetLatestOneAgentImage(ctx context.Context) (*LatestImageInfo, error) {
	latestImageInfo, err := dtc.processLatestImageRequest(ctx, dtc.getLatestOneAgentImageURL())
	if err != nil {
		log.Info("failed to process latest image response")

		return nil, err
	}

	return latestImageInfo, nil
}

func (dtc *dynatraceClient) GetLatestCodeModulesImage(ctx context.Context) (*LatestImageInfo, error) {
	latestImageInfo, err := dtc.processLatestImageRequest(ctx, dtc.getLatestCodeModulesImageURL())
	if err != nil {
		log.Info("failed to process latest image response")

		return nil, err
	}

	return latestImageInfo, nil
}

func (dtc *dynatraceClient) GetLatestActiveGateImage(ctx context.Context) (*LatestImageInfo, error) {
	latestImageInfo, err := dtc.processLatestImageRequest(ctx, dtc.getLatestActiveGateImageURL())
	if err != nil {
		log.Info("failed to process latest image response")

		return nil, err
	}

	return latestImageInfo, nil
}

func (dtc *dynatraceClient) processLatestImageRequest(ctx context.Context, url string) (*LatestImageInfo, error) {
	request, err := dtc.createLatestImageRequest(ctx, url)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	response, err := dtc.httpClient.Do(request)
	if err != nil {
		log.Info("failed to retrieve latest image")

		return nil, err
	}

	defer utils.CloseBodyAfterRequest(response)

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
		return nil, dtc.handleErrorResponseFromAPI(data, response.StatusCode, response.Header)
	}

	latestImageInfo, err := dtc.readResponseForLatestImage(data)
	if err != nil {
		return nil, err
	}

	return latestImageInfo, err
}

func (dtc *dynatraceClient) createLatestImageRequest(ctx context.Context, url string) (*http.Request, error) {
	body := &LatestImageInfo{}

	bodyData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	request, err := createBaseRequest(
		ctx,
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
