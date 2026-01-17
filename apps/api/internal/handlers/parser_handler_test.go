package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/services"
)

func setupParserHandler() (*ParserHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	service := services.NewParserService()
	handler := NewParserHandler(service)

	router.POST("/api/v1/parser/parse", handler.Parse)
	router.POST("/api/v1/parser/parse-batch", handler.ParseBatch)

	return handler, router
}

func TestParserHandler_Parse_Movie(t *testing.T) {
	_, router := setupParserHandler()

	body := ParseRequest{Filename: "The.Matrix.1999.1080p.BluRay.mkv"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/parser/parse", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Check data
	dataBytes, _ := json.Marshal(resp.Data)
	var result parser.ParseResult
	err = json.Unmarshal(dataBytes, &result)
	require.NoError(t, err)

	assert.Equal(t, parser.ParseStatusSuccess, result.Status)
	assert.Equal(t, parser.MediaTypeMovie, result.MediaType)
	assert.Equal(t, "The Matrix", result.Title)
	assert.Equal(t, 1999, result.Year)
}

func TestParserHandler_Parse_TVShow(t *testing.T) {
	_, router := setupParserHandler()

	body := ParseRequest{Filename: "Breaking.Bad.S01E05.720p.BluRay.mkv"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/parser/parse", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataBytes, _ := json.Marshal(resp.Data)
	var result parser.ParseResult
	err = json.Unmarshal(dataBytes, &result)
	require.NoError(t, err)

	assert.Equal(t, parser.ParseStatusSuccess, result.Status)
	assert.Equal(t, parser.MediaTypeTVShow, result.MediaType)
	assert.Equal(t, "Breaking Bad", result.Title)
	assert.Equal(t, 1, result.Season)
	assert.Equal(t, 5, result.Episode)
}

func TestParserHandler_Parse_NeedsAI(t *testing.T) {
	_, router := setupParserHandler()

	body := ParseRequest{Filename: "[Leopard-Raws] Kimetsu no Yaiba - 26.mkv"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/parser/parse", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataBytes, _ := json.Marshal(resp.Data)
	var result parser.ParseResult
	err = json.Unmarshal(dataBytes, &result)
	require.NoError(t, err)

	assert.Equal(t, parser.ParseStatusNeedsAI, result.Status)
	assert.Equal(t, parser.MediaTypeUnknown, result.MediaType)
}

func TestParserHandler_Parse_MissingFilename(t *testing.T) {
	_, router := setupParserHandler()

	// Empty filename will fail the binding validation
	body := ParseRequest{Filename: ""}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/parser/parse", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	// The binding validation catches empty required fields
	assert.Equal(t, "VALIDATION_INVALID_FORMAT", resp.Error.Code)
}

func TestParserHandler_Parse_InvalidJSON(t *testing.T) {
	_, router := setupParserHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/parser/parse", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestParserHandler_ParseBatch(t *testing.T) {
	_, router := setupParserHandler()

	body := ParseBatchRequest{
		Filenames: []string{
			"The.Matrix.1999.1080p.BluRay.mkv",
			"Breaking.Bad.S01E05.720p.BluRay.mkv",
			"[Group] Anime - 01.mkv",
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/parser/parse-batch", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Check that we got an array of results
	dataBytes, _ := json.Marshal(resp.Data)
	var results []*parser.ParseResult
	err = json.Unmarshal(dataBytes, &results)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	assert.Equal(t, parser.ParseStatusSuccess, results[0].Status)
	assert.Equal(t, parser.MediaTypeMovie, results[0].MediaType)

	assert.Equal(t, parser.ParseStatusSuccess, results[1].Status)
	assert.Equal(t, parser.MediaTypeTVShow, results[1].MediaType)

	assert.Equal(t, parser.ParseStatusNeedsAI, results[2].Status)
}

func TestParserHandler_ParseBatch_EmptyArray(t *testing.T) {
	_, router := setupParserHandler()

	body := ParseBatchRequest{Filenames: []string{}}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/parser/parse-batch", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}
