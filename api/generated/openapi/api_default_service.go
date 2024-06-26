/*
 * Artefact Download Service API
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"context"
	"errors"
	"net/http"
)

// DefaultApiService is a service that implements the logic for the DefaultApiServicer
// This service should implement the business logic for every endpoint for the DefaultApi API.
// Include any external packages or services that will be required by this service.
type DefaultApiService struct {
}

// NewDefaultApiService creates a default api service
func NewDefaultApiService() DefaultApiServicer {
	return &DefaultApiService{}
}

// DownloadsDownloadIdGet - Get the current status of a download
func (s *DefaultApiService) DownloadsDownloadIdGet(ctx context.Context, downloadId string) (ImplResponse, error) {
	// TODO - update DownloadsDownloadIdGet with the required logic for this service method.
	// Add api_default_service.go to the .openapi-generator-ignore to avoid overwriting this service implementation when updating open api generation.

	//TODO: Uncomment the next line to return response Response(200, DownloadStatus{}) or use other options such as http.Ok ...
	//return Response(200, DownloadStatus{}), nil

	//TODO: Uncomment the next line to return response Response(404, Error{}) or use other options such as http.Ok ...
	//return Response(404, Error{}), nil

	//TODO: Uncomment the next line to return response Response(429, Error{}) or use other options such as http.Ok ...
	//return Response(429, Error{}), nil

	return Response(http.StatusNotImplemented, nil), errors.New("DownloadsDownloadIdGet method not implemented")
}

// DownloadsDownloadIdPatch - Update a download
func (s *DefaultApiService) DownloadsDownloadIdPatch(ctx context.Context, downloadId string, downloadUpdate DownloadUpdate) (ImplResponse, error) {
	// TODO - update DownloadsDownloadIdPatch with the required logic for this service method.
	// Add api_default_service.go to the .openapi-generator-ignore to avoid overwriting this service implementation when updating open api generation.

	//TODO: Uncomment the next line to return response Response(202, {}) or use other options such as http.Ok ...
	//return Response(202, nil),nil

	//TODO: Uncomment the next line to return response Response(400, Error{}) or use other options such as http.Ok ...
	//return Response(400, Error{}), nil

	//TODO: Uncomment the next line to return response Response(404, Error{}) or use other options such as http.Ok ...
	//return Response(404, Error{}), nil

	//TODO: Uncomment the next line to return response Response(429, Error{}) or use other options such as http.Ok ...
	//return Response(429, Error{}), nil

	return Response(http.StatusNotImplemented, nil), errors.New("DownloadsDownloadIdPatch method not implemented")
}

// DownloadsGet - List all ongoing downloads
func (s *DefaultApiService) DownloadsGet(ctx context.Context) (ImplResponse, error) {
	// TODO - update DownloadsGet with the required logic for this service method.
	// Add api_default_service.go to the .openapi-generator-ignore to avoid overwriting this service implementation when updating open api generation.

	//TODO: Uncomment the next line to return response Response(200, []DownloadStatus{}) or use other options such as http.Ok ...
	//return Response(200, []DownloadStatus{}), nil

	//TODO: Uncomment the next line to return response Response(429, Error{}) or use other options such as http.Ok ...
	//return Response(429, Error{}), nil

	return Response(http.StatusNotImplemented, nil), errors.New("DownloadsGet method not implemented")
}

// DownloadsPost - Request a new download
func (s *DefaultApiService) DownloadsPost(ctx context.Context, downloadRequest DownloadRequest) (ImplResponse, error) {
	// TODO - update DownloadsPost with the required logic for this service method.
	// Add api_default_service.go to the .openapi-generator-ignore to avoid overwriting this service implementation when updating open api generation.

	//TODO: Uncomment the next line to return response Response(202, DownloadResponse{}) or use other options such as http.Ok ...
	//return Response(202, DownloadResponse{}), nil

	//TODO: Uncomment the next line to return response Response(400, Error{}) or use other options such as http.Ok ...
	//return Response(400, Error{}), nil

	//TODO: Uncomment the next line to return response Response(429, Error{}) or use other options such as http.Ok ...
	//return Response(429, Error{}), nil

	return Response(http.StatusNotImplemented, nil), errors.New("DownloadsPost method not implemented")
}
