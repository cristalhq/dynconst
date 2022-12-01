package dynconst

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSet(t *testing.T) {
	batchSize := NewInt(1000, "TestSet-batch")
	if val := batchSize.Value(); val != 1000 {
		t.Fatal(val)
	}

	s := httptest.NewServer(http.HandlerFunc(SetHandler))
	defer s.Close()

	url := s.URL + "?name=TestSet-batch&value=123"
	req, err := http.NewRequest(http.MethodPost, url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.StatusCode)
	}
	if val := batchSize.Value(); val != 123 {
		t.Fatal(val)
	}
}

func TestViewJSON(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(ViewHandler))
	defer s.Close()

	req, err := http.NewRequest(http.MethodPost, s.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	t.Log(string(body))
}

func TestViewText(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(ViewHandler))
	defer s.Close()

	req, err := http.NewRequest(http.MethodPost, s.URL+"?format=text", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	t.Log(string(body))
}
