package http

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

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

func (h *HttpClient) FetchData(d *Download, fragment *Fragment) error {
	slog.Debug("download", "Fragment", fragment)
	req, err := http.NewRequestWithContext(d.Context, "GET", d.Uri, nil)
	if err != nil {
		slog.Error("request", "error", err)
		return err
	}
	if d.MaxFragments > 1 && fragment.End > fragment.Start {
		rangeHeader := "bytes=" + strconv.FormatInt(fragment.Start, 10) + "-" + strconv.FormatInt(fragment.End, 10)
		slog.Debug("rangeHeader", "rangeHeader", rangeHeader)
		req.Header.Add("Range", rangeHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("download", "error", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		slog.Error("download", "error", err, "status", resp.StatusCode)
		return err
	}
	_, err = d.File.Seek(fragment.Start, 0)
	if err != nil {
		slog.Error("seek", "error", err)
		return err
	}
	buf := make([]byte, d.BufferSize)
	for {
		read, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			slog.Error("read", "error", err)
			return err
		}
		if read == 0 {
			break
		}
		_, err = fragment.Destination.Write(buf[:read])
		if err != nil {
			slog.Error("write", "error", err)
			return err
		}
		fragment.Progress += int64(read)
	}
	slog.Debug("write", "written", fragment.Progress, "from", fragment.End-fragment.Start)
	return nil
}
