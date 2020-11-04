package lcu

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type client struct {
	httpClient *http.Client
	baseUrl    *url.URL
}

type authTransport struct {
	token     string
	transport *http.RoundTripper
}

func newTransport(token string) authTransport {

	transport := http.DefaultTransport
	transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return authTransport{
		token,
		&transport,
	}

}

func (t authTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	request.SetBasicAuth("riot", t.token)
	return http.DefaultTransport.RoundTrip(request)

}

func (c client) Post(path string, body interface{}) (*http.Response, error) {
	reqUrl, err := c.baseUrl.Parse(path)
	if err != nil {
		return nil, err
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Post(reqUrl.String(),
		"application/json", bytes.NewBuffer(bodyBytes))
}

func (c client) Get(path string) (*http.Response, error) {
	reqUrl, err := c.baseUrl.Parse(path)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Get(reqUrl.String())
}

func New(protocol, token string, port int) (*client, error) {
	baseUrl := url.URL{Scheme: protocol, Host: fmt.Sprintf("127.0.0.1:%d", port)}

	client := &client{
		httpClient: &http.Client{
			Transport: newTransport(token),
		},
		baseUrl: &baseUrl,
	}
	response, err := client.Get("/lol-summoner/v1/current-summoner")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to verify the connection to the League Client")
	}

	if response.StatusCode == 403 {
		return nil, errors.New("Could not authorize to the League Client")
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read from Response")
	}

	var parsedResponse map[string]interface{}
	err = json.Unmarshal(responseBytes, &parsedResponse)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse response from League Client")
	}
	if parsedResponse["accountId"] == 0 {
		return nil, errors.New("Failed to verify response data from League Client (missing accountId)")
	}

	return client, nil
}
