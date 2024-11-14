package main

import (
	"testing"

	"github.com/numkem/traffikey"
	"github.com/stretchr/testify/assert"
)

func TestHTTPTarget(t *testing.T) {
	aliveUrls := testHTTPTarget(&traffikey.Target{
		Name: "google.com",
		Type: "http",
		ServerURLs: []string{
			"https://google.com",
		},
	})

	assert.NotEmpty(t, aliveUrls, "testHTTPTarget should return one url")
}

func TestHTTPTargetFailed(t *testing.T) {
	aliveUrls := testHTTPTarget(&traffikey.Target{
		Name: "failed.com",
		Type: "http",
		ServerURLs: []string{
			"http://falksjdfalskjfaldkjf.com",
		},
	})

	assert.Empty(t, aliveUrls, "testHTTPTarget should not return any url")
}
