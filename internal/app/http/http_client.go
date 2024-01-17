package http

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

type HttpClient struct {
	client *http.Client
}

func NewHttpClient() *HttpClient {
	return &HttpClient{
		client: &http.Client{},
	}
}

func (h *HttpClient) FetchData(d Download, fragment Fragment) error {
	slog.Debug("download", "Fragment", fragment)
	req, err := http.NewRequestWithContext(d.Context, "GET", d.Uri, nil)
	if err != nil {
		slog.Error("request", "error", err)
		return err
	}
	if d.Fragments > 1 && fragment.End > fragment.Start {
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
	written, err := io.CopyBuffer(fragment.Destination, resp.Body, buf)
	slog.Debug("write", "written", written, "from", fragment.End-fragment.Start)
	if err != nil {
		slog.Error("write", "error", err)
		return err
	}
	return nil
}
