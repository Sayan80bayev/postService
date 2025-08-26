package integration

import (
	"bytes"
	"encoding/json"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"postService/tests/testutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPostLifecycle(t *testing.T) {
	userID := uuid.New()
	token := testutil.GenerateMockToken(userID.String())
	logger := logging.GetLogger()
	// --- Step 2: Run consumer in goroutine ---
	go container.Consumer.Start()

	// Always close the consumer after the test ends
	t.Cleanup(func() {
		logger.Info("Shutting down consumer...")
		container.Consumer.Close()
	})
	// -------- CREATE POST --------
	var createdID string
	{
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// text field
		require.NoError(t, writer.WriteField("content", "Hello world from integration test"))

		// attach image file
		imgPath := filepath.Join("../assets", "img.png")
		imgFile, err := os.Open(imgPath)
		require.NoError(t, err)
		defer imgFile.Close()
		part, err := writer.CreateFormFile("media", filepath.Base(imgPath))
		require.NoError(t, err)
		_, err = io.Copy(part, imgFile)
		require.NoError(t, err)

		// attach pdf file
		pdfPath := filepath.Join("../assets", "test_1.pdf")
		pdfFile, err := os.Open(pdfPath)
		require.NoError(t, err)
		defer pdfFile.Close()
		part, err = writer.CreateFormFile("files", filepath.Base(pdfPath))
		require.NoError(t, err)
		_, err = io.Copy(part, pdfFile)
		require.NoError(t, err)

		require.NoError(t, writer.Close())

		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		testApp.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		// fetch all posts to get ID
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
		w2 := httptest.NewRecorder()
		testApp.ServeHTTP(w2, req2)
		require.Equal(t, http.StatusOK, w2.Code)

		var posts []map[string]interface{}
		require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &posts))
		require.NotEmpty(t, posts)

		createdID = posts[0]["id"].(string)
		require.NotEmpty(t, createdID)
	}

	// -------- READ POST BY ID --------
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/"+createdID, nil)
		w := httptest.NewRecorder()
		testApp.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var post map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &post))
		require.Equal(t, createdID, post["id"])
		require.Equal(t, userID.String(), post["user_id"])
	}

	// -------- UPDATE POST --------
	{
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		require.NoError(t, writer.WriteField("content", "Updated content!"))
		require.NoError(t, writer.Close())

		req := httptest.NewRequest(http.MethodPut, "/api/v1/posts/"+createdID, body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		testApp.ServeHTTP(w, req)
		time.Sleep(5 * time.Second)

		require.Equal(t, http.StatusOK, w.Code)
	}

	// -------- DELETE POST --------
	{
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/posts/"+createdID, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		testApp.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		time.Sleep(5 * time.Second)
	}

	// -------- CONFIRM DELETE --------
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/"+createdID, nil)
		w := httptest.NewRecorder()
		testApp.ServeHTTP(w, req)
		require.Equal(t, http.StatusNotFound, w.Code)
	}
}
