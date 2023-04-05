package pushclient

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

type HttpFileWriter struct {
	http.Client

	Filename string
}

func (c *HttpFileWriter) Do(req *http.Request) (*http.Response, error) {
	metric, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(c.Filename, metric, 0600)
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("Done"))}, nil
}
