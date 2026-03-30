package metadata

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateBucketSingleName(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name": "my-first-bucket"}`))
	rr := httptest.NewRecorder()

	hldr := http.HandlerFunc(CreateBucketHandler)
	hldr.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusCreated {
		t.Fatalf("\n[EXP] %d\n[GOT] %d", http.StatusCreated, rr.Result().StatusCode)
	}
}

func TestCreateBucketDuplicateName(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name": "my-first-bucket"}`))
	rr := httptest.NewRecorder()

	hldr := http.HandlerFunc(CreateBucketHandler)
	hldr.ServeHTTP(rr, req)

	// This one should fail with duplicate bucket name error
	req = httptest.NewRequest("POST", "/", strings.NewReader(`{"name": "my-first-bucket"}`))
	rr = httptest.NewRecorder()
	hldr.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusConflict {
		t.Fatalf("\n[EXP] %d\n[GOT] %d", http.StatusConflict, rr.Result().StatusCode)
	}

	if rr.Body.String() != "Duplicate bucket name" {
		t.Fatalf("\n[EXP] %s\n[GOT] %s", "Duplicate Bucket name", rr.Body.String())
	}
}

func TestCreateBucketMalformedJson(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"not-name": "not-name"}`))
	rr := httptest.NewRecorder()

	hldr := http.HandlerFunc(CreateBucketHandler)
	hldr.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("\n[EXP] %d\n[GOT] %d", http.StatusBadRequest, rr.Result().StatusCode)
	}

	if rr.Body.String() != "Bucket name is required" {
		t.Fatalf("\n[EXP] %s\n[GOT] %s", "Bucket name is required", rr.Body.String())
	}
}
