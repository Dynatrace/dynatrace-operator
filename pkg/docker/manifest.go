package docker

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Config struct {
	Digest string
}

type FsLayer struct {
	BlobSum string
}

type ManifestV2 struct {
	Config Config
}

type ManifestV1 struct {
	FsLayers []FsLayer
}

func (registry *Registry) GetManifest(digest string) (*ManifestV2, error) {
	request, err := registry.prepareRequest(registry.buildUrl(digest))
	if err != nil {
		return nil, err
	}
	return registry.getManifest(request, digest)
}

func (registry *Registry) GetLatestManifest() (*ManifestV2, error) {
	return registry.GetManifest(Latest)
}

func (registry *Registry) getManifest(request *http.Request, digest string) (*ManifestV2, error) {
	if registry.Server == DockerHubApiServer ||
		registry.Server == GcrApiServer ||
		registry.Server == QuayApiServer {
		request.Header.Add("Accept", ContentTypeManifestListV2)
	} else {
		request.Header.Add("Accept", ContentTypeManifestV2)
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == 404 {
		// If no v2 manifest list exists, try OCI v1 for DockerHub or V2 for anything else
		request, err = registry.prepareRequest(registry.buildUrl(digest))
		if err != nil {
			return nil, err
		}

		if registry.Server == DockerHubApiServer {
			request.Header.Add("Accept", ContentTypeManifestOCIV1)
		} else {
			request.Header.Add("Accept", ContentTypeManifestV2)
		}

		response, err = client.Do(request)
		if err != nil {
			return nil, err
		}
	}

	defer func() {
		//Ignore error because there is nothing one could do here
		_ = response.Body.Close()
	}()

	switch response.StatusCode {
	case 200:
		return registry.parseManifest(response, digest)
	case 201:
		return registry.parseManifest(response, digest)
	case 404:
		return nil, fmt.Errorf("could not find image: code: %d, status: %s", response.StatusCode, response.Status)
	case 401:
		return nil, fmt.Errorf("authorization failed: code: %d, status: %s", response.StatusCode, response.Status)
	default:
		return nil, fmt.Errorf("unexpected response: code: %d, status: %s", response.StatusCode, response.Status)
	}
}

func (registry *Registry) parseManifest(response *http.Response, digest string) (*ManifestV2, error) {
	// GCR and DockerHub send the image digests in the header
	if registry.Server == GcrApiServer || registry.Server == DockerHubApiServer {
		value := response.Header.Get(DockerContentDigest)
		return &ManifestV2{Config: Config{Digest: value}}, nil
	}

	// Quay sends manifest digest in header, which appears to differs for each tag
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	contentType := response.Header.Get(ContentType)
	if contentType == ContentTypeManifestListV2 {
		// Special handling for special DockerHub
		return handleV2ManifestList(body, digest)
	} else if contentType == ContentTypeManifestV1 || contentType == ContentTypeManifestV1Pretty {
		// If repository does not support v2 Manifest parse v1 and hash layers to one hash
		return handleV1Manifest(body)
	} else if contentType == ContentTypeManifestV2 {
		return handleV2Manifest(body)
	} else if contentType == ContentTypeManifestOCIV1 {
		type schemaVersion struct {
			SchemaVersion int32
		}
		var schema schemaVersion
		err := json.Unmarshal(body, &schema)
		if err != nil {
			return nil, err
		}

		if schema.SchemaVersion == 1 {
			return handleV1Manifest(body)
		} else if schema.SchemaVersion == 2 {
			return handleV2Manifest(body)
		}
	}

	return nil, fmt.Errorf("encountered unknown content-type while parsing manifest: " + contentType)
}

func handleV2ManifestList(body []byte, digest string) (*ManifestV2, error) {
	type manifestList struct {
		Manifests []Config
	}
	var manifests manifestList
	err := json.Unmarshal(body, &manifests)
	if err != nil {
		return nil, err
	}
	// If a list of manifests is returned, it exists and is valid
	return &ManifestV2{Config: Config{Digest: digest}}, nil
}

func handleV2Manifest(body []byte) (*ManifestV2, error) {
	// Repository supports v2 Manifest, act normal
	var manifest ManifestV2
	err := json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, err
}

func handleV1Manifest(body []byte) (*ManifestV2, error) {
	var manifest ManifestV1
	var result ManifestV2
	err := json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, err
	}

	// Add hashes of layers and create a hash from there
	hashSum := ""
	for _, layer := range manifest.FsLayers {
		hashSum += layer.BlobSum
	}

	hash := sha256.Sum256([]byte(hashSum))
	prinableHash := base64.StdEncoding.EncodeToString(hash[:])
	result = ManifestV2{Config: struct{ Digest string }{Digest: string(prinableHash)}}
	return &result, nil
}

func (registry *Registry) buildUrl(digest string) string {
	image := registry.Image
	if registry.Server == DockerHubApiServer && !strings.Contains(image, "/") {
		//Special handling for DockerHub
		return fmt.Sprintf(UrlTemplate, registry.Server, "library/"+image, digest)
	}
	return fmt.Sprintf(UrlTemplate, registry.Server, image, digest)
}

const (
	ContentType                 = "content-type"
	ContentTypeManifestV1       = "application/vnd.docker.distribution.manifest.v1+json"
	ContentTypeManifestOCIV1    = "application/vnd.oci.image.manifest.v1+json"
	ContentTypeManifestV1Pretty = "application/vnd.docker.distribution.manifest.v1+prettyjws"
	ContentTypeManifestV2       = "application/vnd.docker.distribution.manifest.v2+json"
	ContentTypeManifestListV2   = "application/vnd.docker.distribution.manifest.list.v2+json"

	DockerContentDigest = "Docker-Content-Digest"
)
