package hlsdl

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// share one HTTP client
var httpClient *http.Client = &http.Client{}

func (client *Client) doRequest(url string) (res *http.Response, err error) {
	logger.Println("GET", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	for key, val := range client.Headers {
		req.Header.Add(key, val)
	}
	res, err = httpClient.Do(req)
	if err != nil {
		return
	} else if res.StatusCode != 200 {
		err = errors.New(res.Status)
	}
	return
}

func absURL(baseURL, relURL string) (newURL string, err error) {
	parsed, err := url.Parse(relURL)
	if err == nil && strings.HasPrefix(parsed.Scheme, "http") {
		return relURL, nil
	}
	parsed, err = url.Parse(baseURL)
	if err != nil {
		return
	} else if !strings.HasPrefix(parsed.Scheme, "http") {
		err = fmt.Errorf("URL '%s' is not HTTP", baseURL)
		return
	}
	dir := path.Dir(parsed.Path)
	if dir == "/" {
		dir = ""
	}
	newURL = fmt.Sprintf("%s://%s%s/%s", parsed.Scheme, parsed.Host, dir, relURL)
	return
}
