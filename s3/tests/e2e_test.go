package tests

import (
	"encoding/json"
	"net/http"
	"testing"
)

// One test for each scenario:
// CreateBucket
// ListBuckets
// DeleteBucket
// ListObjects
// PutObject
// GetObject
// DeleteObject

const (
	baseUrl = "http://localhost:8080"
)

func fatal(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestE2E_CreateBucket(t *testing.T) {
	client := &http.Client{}
	var (
		req  *http.Request
		resp *http.Response
		err  error
	)

	// Create a Bucket
	req, _ = http.NewRequest("PUT", baseUrl+"/b1", nil)
	resp, err = client.Do(req)
	fatal(err, t)
	if resp.StatusCode != http.StatusCreated {
		t.Fatal("create bucket - status code: ", resp.StatusCode)
	}

	// list buckets
	req, _ = http.NewRequest("GET", baseUrl+"/", nil)
	resp, err = client.Do(req)
	fatal(err, t)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatal("list buckets - status code: ", resp.StatusCode)
	}

	var body []any
	json.NewDecoder(resp.Body).Decode(&body)
	bucketName := body[0].(map[string]any)["Key"]
	if bucketName != "b1" {
		t.Fatal("list buckets - bucket name: ", bucketName)
	}

	// delete bucket
	req, _ = http.NewRequest("DELETE", baseUrl+"/b1", nil)
	resp, err = client.Do(req)
	fatal(err, t)
	if resp.StatusCode != http.StatusOK {
		t.Fatal("delete bucket - status code: ", resp.StatusCode)
	}

	// list buckets - should be empty now
	req, _ = http.NewRequest("GET", baseUrl+"/", nil)
	resp, err = client.Do(req)
	fatal(err, t)
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body) > 0 {
		t.Fatal("delete bucket - still present: ", body[0].(map[string]any)["Key"])
	}
}

