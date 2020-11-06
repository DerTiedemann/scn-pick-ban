package lcu

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type HttpClient struct {
	httpClient *http.Client
	baseUrl    url.URL
	token      string
}

type authTransport struct {
	token     string
	transport http.RoundTripper
}

func newAuthTransport(token string, transport http.RoundTripper) authTransport {

	transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return authTransport{
		token,
		transport,
	}

}

func (t authTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	request.SetBasicAuth("riot", t.token)
	return t.transport.RoundTrip(request)
}

func (c *HttpClient) Post(path string, body interface{}) (*http.Response, error) {
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

func (c *HttpClient) Get(path string) (*http.Response, error) {
	reqUrl, err := c.baseUrl.Parse(path)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Get(reqUrl.String())
}

func (c *HttpClient) validateConnection() error {

	response, err := c.Get("/lol-summoner/v1/current-summoner")
	if err != nil {
		return errors.Wrap(err, "failed to connect")
	}

	if response.StatusCode == 403 {
		return errors.New("invalid authorization")
	}

	var parsedResponse map[string]interface{}
	d := json.NewDecoder(response.Body)
	d.UseNumber()
	err = d.Decode(&parsedResponse)
	if err != nil {
		return errors.Wrap(err, "failed to parse response")
	}
	accountId, err := parsedResponse["accountId"].(json.Number).Int64()
	if err != nil {
		return errors.Wrap(err, "could not parse accountId")
	}
	if accountId == 0 {
		return errors.New("invalid accountId")
	}

	return nil
}

func NewHttpClient(options Options) (*HttpClient, error) {

	if err := options.validate(); err != nil {
		return nil, errors.Wrap(err, "invalid options")
	}

	client := &HttpClient{
		httpClient: &http.Client{
			Transport: newAuthTransport(options.Token, http.DefaultTransport),
		},
		baseUrl: url.URL{
			Scheme: options.Protocol,
			Host:   fmt.Sprintf("127.0.0.1:%d", options.Port),
		},
		token: options.Token,
	}

	err := client.validateConnection()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate connection to league client")
	}

	return client, nil
}
