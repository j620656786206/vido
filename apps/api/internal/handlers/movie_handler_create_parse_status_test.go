package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// parse_status on create (E2E seed of a pending row — see CreateMovieRequest).
func TestMovieHandler_Create_ParseStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newRouter := func(svc *MockMovieService) *gin.Engine {
		r := gin.New()
		NewMovieHandler(svc).RegisterRoutes(r.Group("/api/v1"))
		return r
	}
	post := func(r *gin.Engine, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("pending is stored on the row", func(t *testing.T) {
		svc := new(MockMovieService)
		var created *models.Movie
		svc.On("Create", mock.Anything, mock.AnythingOfType("*models.Movie")).
			Run(func(args mock.Arguments) { created = args.Get(1).(*models.Movie) }).
			Return(nil).Once()

		w := post(newRouter(svc), `{"title":"Pending","release_date":"2020-01-01","parse_status":"pending"}`)

		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		require.NotNil(t, created)
		assert.Equal(t, models.ParseStatusPending, created.ParseStatus)
		var resp APIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "pending", resp.Data.(map[string]interface{})["parse_status"])
	})

	t.Run("omitted keeps the row stateless, as before", func(t *testing.T) {
		svc := new(MockMovieService)
		var created *models.Movie
		svc.On("Create", mock.Anything, mock.AnythingOfType("*models.Movie")).
			Run(func(args mock.Arguments) { created = args.Get(1).(*models.Movie) }).
			Return(nil).Once()

		w := post(newRouter(svc), `{"title":"Plain","release_date":"2020-01-01"}`)

		require.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, models.ParseStatus(""), created.ParseStatus)
	})

	t.Run("an unknown value is a 400, nothing created", func(t *testing.T) {
		svc := new(MockMovieService)
		w := post(newRouter(svc), `{"title":"Bad","release_date":"2020-01-01","parse_status":"parsing-ish"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		svc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}
