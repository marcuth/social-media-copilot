// Package firecrawl is a small client for the Firecrawl v2 search and scrape
// endpoints (https://docs.firecrawl.dev/api-reference/introduction).
package firecrawl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.firecrawl.dev"

type Client struct {
	BaseURL string
	// APIKey is optional: self-hosted instances usually run without auth, and
	// an empty bearer token is rejected, so no header is sent when it's empty.
	APIKey string
	HTTP   *http.Client
}

type Result struct {
	URL         string
	Title       string
	Description string
	Markdown    string
}

type Page struct {
	URL      string
	Title    string
	Markdown string
}

type SearchOptions struct {
	Limit int
	// Country is an ISO code such as "BR"; empty uses Firecrawl's default.
	Country string
	// TBS filters web results by time, e.g. "qdr:w" for the past week.
	TBS string
	// WithContent also scrapes each result as Markdown (costs extra credits).
	WithContent bool
}

// Error is a non-2xx answer from Firecrawl.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	switch e.Status {
	case http.StatusUnauthorized:
		return "o Firecrawl recusou a chave de API"
	case http.StatusPaymentRequired:
		return "a conta do Firecrawl está sem créditos"
	case http.StatusTooManyRequests:
		return "limite de requisições do Firecrawl atingido; tente de novo em instantes"
	case http.StatusRequestTimeout:
		return "o Firecrawl demorou demais para responder"
	}
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	return fmt.Sprintf("erro do Firecrawl (%d): %s", e.Status, msg)
}

func (c *Client) endpoint(path string) string {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		base = DefaultBaseURL
	}
	base = strings.TrimSuffix(base, "/v2")
	return base + "/v2/" + path
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(path), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(c.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("não foi possível conectar ao Firecrawl: %w", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		var e struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		_ = json.Unmarshal(data, &e)
		return &Error{Status: res.StatusCode, Code: e.Code, Message: e.Error}
	}
	return json.Unmarshal(data, out)
}

type apiResult struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Snippet     string `json:"snippet"`
	Markdown    string `json:"markdown"`
	Metadata    struct {
		Title     string `json:"title"`
		SourceURL string `json:"sourceURL"`
	} `json:"metadata"`
}

func (r apiResult) toResult() Result {
	out := Result{URL: r.URL, Title: r.Title, Description: r.Description, Markdown: r.Markdown}
	if out.URL == "" {
		out.URL = r.Metadata.SourceURL
	}
	if out.Title == "" {
		out.Title = r.Metadata.Title
	}
	if out.Description == "" {
		out.Description = r.Snippet
	}
	return out
}

func (c *Client) Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("pesquisa vazia")
	}
	if len([]rune(query)) > 500 {
		query = string([]rune(query)[:500])
	}
	body := map[string]any{"query": query}
	if opts.Limit > 0 {
		body["limit"] = opts.Limit
	}
	if opts.Country != "" {
		body["country"] = opts.Country
	}
	if opts.TBS != "" {
		body["tbs"] = opts.TBS
	}
	if opts.WithContent {
		body["scrapeOptions"] = map[string]any{"formats": []string{"markdown"}, "onlyMainContent": true}
	}
	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := c.post(ctx, "search", body, &resp); err != nil {
		return nil, err
	}
	// v2 answers {"data": {"web": [...], "news": [...]}}; some docs examples
	// still show the v1 shape {"data": [...]}, so both are accepted.
	var items []apiResult
	var grouped struct {
		Web  []apiResult `json:"web"`
		News []apiResult `json:"news"`
	}
	if err := json.Unmarshal(resp.Data, &grouped); err == nil {
		items = append(grouped.Web, grouped.News...)
	} else if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, fmt.Errorf("resposta inesperada do Firecrawl: %w", err)
	}
	out := make([]Result, 0, len(items))
	for _, it := range items {
		if r := it.toResult(); r.URL != "" {
			out = append(out, r)
		}
	}
	return out, nil
}

func (c *Client) Scrape(ctx context.Context, url string) (Page, error) {
	var resp struct {
		Success bool      `json:"success"`
		Data    apiResult `json:"data"`
	}
	err := c.post(ctx, "scrape", map[string]any{
		"url":             strings.TrimSpace(url),
		"formats":         []string{"markdown"},
		"onlyMainContent": true,
		"timeout":         60000,
	}, &resp)
	if err != nil {
		return Page{}, err
	}
	r := resp.Data.toResult()
	if r.URL == "" {
		r.URL = url
	}
	return Page{URL: r.URL, Title: r.Title, Markdown: r.Markdown}, nil
}
