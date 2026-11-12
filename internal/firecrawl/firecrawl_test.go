package firecrawl

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func server(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, body map[string]any)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		handler(w, r, body)
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL + "/v2/"}
}

func TestSearchV2Shape(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		if r.URL.Path != "/v2/search" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("no key must mean no Authorization header")
		}
		if body["query"] != "tofu" || body["country"] != "BR" || body["tbs"] != "qdr:w" {
			t.Errorf("body %v", body)
		}
		if _, ok := body["scrapeOptions"]; !ok {
			t.Error("WithContent should request markdown")
		}
		w.Write([]byte(`{"success":true,"data":{"web":[{"url":"https://a.com","title":"A","description":"d","markdown":"# A"}],
			"news":[{"url":"https://n.com","title":"N","snippet":"s"}]}}`))
	})
	got, err := c.Search(context.Background(), "tofu", SearchOptions{Limit: 3, Country: "BR", TBS: "qdr:w", WithContent: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Markdown != "# A" || got[1].Description != "s" {
		t.Fatalf("results %+v", got)
	}
}

func TestSearchV1ShapeAndAuth(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request, _ map[string]any) {
		if r.Header.Get("Authorization") != "Bearer fc-1" {
			t.Errorf("auth %q", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"success":true,"data":[{"markdown":"x","metadata":{"title":"T","sourceURL":"https://b.com"}}]}`))
	})
	c.APIKey = "fc-1"
	got, err := c.Search(context.Background(), "q", SearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].URL != "https://b.com" || got[0].Title != "T" {
		t.Fatalf("results %+v", got)
	}
}

func TestScrape(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		if r.URL.Path != "/v2/scrape" || body["url"] != "https://c.com/p" {
			t.Errorf("request %s %v", r.URL.Path, body)
		}
		w.Write([]byte(`{"success":true,"data":{"markdown":"# Página","metadata":{"title":"Página","sourceURL":"https://c.com/p","statusCode":200}}}`))
	})
	page, err := c.Scrape(context.Background(), "https://c.com/p")
	if err != nil {
		t.Fatal(err)
	}
	if page.Title != "Página" || page.Markdown != "# Página" || page.URL != "https://c.com/p" {
		t.Fatalf("page %+v", page)
	}
}

func TestErrors(t *testing.T) {
	c := server(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte(`{"success":false,"error":"Payment required to access this resource.","code":"insufficient_credits"}`))
	})
	_, err := c.Scrape(context.Background(), "https://x.com")
	var fe *Error
	if !errors.As(err, &fe) || fe.Status != 402 || fe.Code != "insufficient_credits" {
		t.Fatalf("err %v", err)
	}
	if fe.Error() != "a conta do Firecrawl está sem créditos" {
		t.Fatalf("message %q", fe.Error())
	}
}
