package http

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/codejago/polypully/downloader/internal/app/model"
)

var _ model.CommunicationClient = (*HttpClient)(nil)

type HttpClient struct {
	client *http.Client
}

type HttpClientConfig struct {
	Timeout   time.Duration
	Redirects int
}

func NewHttpClient(h *HttpClientConfig) *HttpClient {
	return &HttpClient{
		client: &http.Client{
			Timeout: h.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= h.Redirects {
					return fmt.Errorf("stopped after %d redirects", h.Redirects)
				}
				lastResponse := via[len(via)-1]
				switch lastResponse.Response.StatusCode {
				case http.StatusMultipleChoices, http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusNotModified, http.StatusUseProxy, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
					return nil
				default:
					return http.ErrUseLastResponse
				}
			},
		},
	}
}

// FetchData fetches a fragment of data from the resource
// and writes it to the destination fragment file. The
// context is used to enable cancellation of the fetch.
func (h *HttpClient) FetchData(context context.Context, d *model.Resource, fragment *model.Fragment) error {
	slog.Debug("download", "Fragment", fragment)
	req, err := http.NewRequestWithContext(context, "GET", d.Uri, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	if len(d.Fragments) > 1 && fragment.End > fragment.Start {
		rangeHeader := "bytes=" + strconv.FormatInt(int64(fragment.Start), 10) + "-" +
			strconv.FormatInt(int64(fragment.End), 10)
		req.Header.Add("Range", rangeHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error downloading: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("error downloading: %v", err)
	}
	buf := make([]byte, d.BufferSize)
	for {
		read, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading: %v", err)
		}
		if read == 0 {
			break
		}
		_, err = fragment.Destination.Write(buf[:read])
		if err != nil {
			return fmt.Errorf("error writing: %v", err)
		}
		fragment.Progress += read
	}
	slog.Debug("write", "wrote", fragment.Progress, "from", fragment.End-fragment.Start)
	return nil
}
