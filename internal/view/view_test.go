package view

import (
	"strings"
	"testing"
	"testing/fstest"
)

const styleURL = "/static/style.css?v="

func TestAssetURLChangesWithContent(t *testing.T) {
	t.Parallel()

	first := styleURLFor(t, "body{}")
	second := styleURLFor(t, "body{color:red}")

	if first == second {
		t.Errorf("expected changed content to yield a new URL, got %q twice", first)
	}
	if !strings.HasPrefix(first, styleURL) {
		t.Fatalf("unexpected URL shape: %q", first)
	}
	if got := len(strings.TrimPrefix(first, styleURL)); got != 8 {
		t.Errorf("expected an 8 character hash, got %d in %q", got, first)
	}
}

func TestAssetURLRejectsUnknownAsset(t *testing.T) {
	t.Parallel()

	asset, err := assetURL(fstest.MapFS{"static/style.css": {Data: []byte("body{}")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := asset("missing.js"); err == nil {
		t.Error("expected an error for an asset that is not embedded")
	}
}

func styleURLFor(t *testing.T, content string) string {
	t.Helper()

	asset, err := assetURL(fstest.MapFS{"static/style.css": {Data: []byte(content)}})
	if err != nil {
		t.Fatal(err)
	}
	u, err := asset("style.css")
	if err != nil {
		t.Fatal(err)
	}
	return u
}
