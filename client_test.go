package reggie

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gopkg.in/resty.v1"
)

func TestClient(t *testing.T) {
	authTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		h := r.Header.Get("Authorization")
		if h != expectedAuthHeader {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token": "abc123"}`))
		}
	}))
	defer authTestServer.Close()

	registryTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "Bearer abc123" {
			w.WriteHeader(http.StatusOK)
		} else {
			wwwHeader := fmt.Sprintf("Bearer realm=\"%s/v2/auth\",service=\"testservice\",scope=\"testscope\"",
				authTestServer.URL)
			w.Header().Set("www-authenticate", wwwHeader)
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer registryTestServer.Close()

	client := &Client{}
	client.Client = resty.New()
	client.Config.Address = registryTestServer.URL
	client.SetName("testnamespace")
	client.Config.Auth.Basic.Username = "testuser"
	client.Config.Auth.Basic.Password = "testpass"

	req := client.NewRequest(resty.MethodGet, "/v2/:name/tags/list")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error executing request: %s", err)
	}

	if status := resp.StatusCode(); status != http.StatusOK {
		t.Fatalf("Expected response code 200 but was %d", status)
	}

	req = client.NewRequest(resty.MethodGet, "/v2/:name/tags/list")
	oldURL := req.URL
	param := "digest"
	value := "zwxyz"
	req.SetQueryParam(param, value)
	if req.URL != oldURL {
		t.Fatalf("Something is destroying the request url before Do.\n\tOriginal Url: %s\n\tUrl After Do: %s",
			oldURL, req.URL)
	}

	resp, err = client.Do(req)

	if req.URL != oldURL + fmt.Sprintf("?%s=%s", param, value) {
		t.Errorf("Do is destroying the request url.\n\tOriginal Url: %s\n\tUrl After Do: %s",
			oldURL, req.Request.URL)
	}
}
