// Package skillgild is the official Go client for SkillGild (https://skillgild.dev),
// the marketplace for hosted AI agent skills. It searches the catalog, runs hosted
// skills and drives hybrid tool sessions over SkillGild's REST API.
//
//	client, err := skillgild.NewClient("", skillgild.WithAPIKey(os.Getenv("SKILLGILD_API_KEY")))
//	run, err := client.RunSkill(ctx, "dashboard-review", map[string]any{"prompt": "Review this brief"})
package skillgild

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.skillgild.dev/v1"

// ErrAPIKeyRequired is returned by signed-in methods on a client built without WithAPIKey.
var ErrAPIKeyRequired = errors.New("skillgild: an API key is required for this request")

// maxResponseBytes bounds a response body; a hosted run's output is capped well below it.
const maxResponseBytes = 8 << 20

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

type Option func(*Client)

func WithAPIKey(apiKey string) Option { return func(client *Client) { client.apiKey = apiKey } }
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.http = client
		}
	}
}

type Skill struct {
	ID               string          `json:"id"`
	Slug             string          `json:"slug"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	DistributionMode string          `json:"distribution_mode"`
	AccessTier       string          `json:"access_tier"`
	PriceAmountMinor int64           `json:"price_amount_minor"`
	PriceCurrency    string          `json:"price_currency"`
	IncludedInPro    bool            `json:"included_in_pro"`
	FreeRunsPerMonth int             `json:"free_runs_per_month"`
	CurrentVersion   string          `json:"current_version"`
	InputSchema      json.RawMessage `json:"input_schema"`
	// RuntimeType is "prompt_pipeline" (runs on SkillGild) or "hybrid_tools"
	// (runs in your agent; use StartSession and CallTool).
	RuntimeType string `json:"runtime_type"`
	// Tools is the public interface of a hybrid skill's server tools (GetSkill only).
	Tools        []Tool `json:"tools,omitempty"`
	Visibility   string `json:"visibility,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
	Category     string `json:"category,omitempty"`
	CategorySlug string `json:"category_slug,omitempty"`
	CreatorName  string `json:"creator_name,omitempty"`
	// Provenance for skills packaged from an upstream repository.
	SourceURL   string `json:"source_url,omitempty"`
	License     string `json:"license,omitempty"`
	Attribution string `json:"attribution,omitempty"`
	// Cover is the card media (catalog listings only).
	Cover *Media `json:"cover,omitempty"`
	// Media is the full ordered gallery (GetSkill only).
	Media []Media `json:"media,omitempty"`
	// Page is the long-form page copy (GetSkill only).
	Page *PageContent `json:"page,omitempty"`
}

// Media is a skill image or video.
type Media struct {
	Kind       string `json:"kind"`
	URL        string `json:"url"`
	PosterURL  string `json:"poster_url,omitempty"`
	PreviewURL string `json:"preview_url,omitempty"`
	Alt        string `json:"alt"`
	Credit     string `json:"credit,omitempty"`
	Width      *int   `json:"width,omitempty"`
	Height     *int   `json:"height,omitempty"`
}

// PageContent is a skill's long-form page copy.
type PageContent struct {
	WhatYouGet   []string `json:"what_you_get,omitempty"`
	GoodFit      []string `json:"good_fit,omitempty"`
	NotFor       []string `json:"not_for,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
	FAQs         []FAQ    `json:"faqs,omitempty"`
}

type FAQ struct {
	Q string `json:"q"`
	A string `json:"a"`
}

// SkillSummary is the shorter skill shape used by featured skills and collections.
type SkillSummary struct {
	ID               string `json:"id"`
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Category         string `json:"category"`
	CreatorName      string `json:"creator_name"`
	Visibility       string `json:"visibility"`
	AccessTier       string `json:"access_tier"`
	PriceAmountMinor int64  `json:"price_amount_minor"`
	PriceCurrency    string `json:"price_currency"`
	FreeRunsPerMonth int    `json:"free_runs_per_month"`
	UpdatedAt        string `json:"updated_at"`
	Cover            *Media `json:"cover,omitempty"`
}

type Category struct {
	ID               string  `json:"id"`
	ParentID         *string `json:"parent_id"`
	Slug             string  `json:"slug"`
	Name             string  `json:"name"`
	ShortDescription string  `json:"short_description"`
	SortOrder        int     `json:"sort_order"`
}

type Tag struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	SkillCount int64  `json:"skill_count"`
}

type Collection struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	SkillCount  int64  `json:"skill_count"`
	UpdatedAt   string `json:"updated_at"`
}

type CollectionDetail struct {
	ID          string         `json:"id"`
	Slug        string         `json:"slug"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	ImageURL    string         `json:"image_url"`
	Skills      []SkillSummary `json:"skills"`
}

// DeviceAuthorization is a pending device login. Show UserCode and VerificationURI to the user.
type DeviceAuthorization struct {
	DeviceCode      string    `json:"device_code"`
	UserCode        string    `json:"user_code"`
	VerificationURI string    `json:"verification_uri"`
	ExpiresAt       time.Time `json:"expires_at"`
	IntervalSeconds int       `json:"interval_seconds"`
}

// DeviceToken is the state of a device login: Status is "authorization_pending"
// or "authorized", and AccessToken holds the issued API key once authorized.
type DeviceToken struct {
	Status      string `json:"status"`
	AccessToken string `json:"access_token,omitempty"`
}

// SkillPage is one page of the catalog. NextCursor is empty on the last page.
type SkillPage struct {
	Items      []Skill `json:"items"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

// ListOptions selects a catalog page. Limit is 1 to 100 (the API default is 50).
type ListOptions struct {
	Query  string
	Limit  int
	Cursor string
}

type Tool struct {
	Name         string          `json:"name"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"input_schema"`
	ExampleInput json.RawMessage `json:"example_input,omitempty"`
}

// Usage is the caller's allowance after a run or a new session.
type Usage struct {
	Used       int       `json:"used"`
	Limit      *int      `json:"limit"`
	ResetAt    time.Time `json:"reset_at"`
	AccessTier string    `json:"access_tier"`
}

type AgentSession struct {
	SessionID     string    `json:"session_id"`
	SkillID       string    `json:"skill_id"`
	SkillSlug     string    `json:"skill_slug"`
	SkillName     string    `json:"skill_name"`
	Version       string    `json:"version"`
	ExpiresAt     time.Time `json:"expires_at"`
	MaxToolCalls  int       `json:"max_tool_calls"`
	ToolCallsUsed int       `json:"tool_calls_used"`
	// Resumed is true when an open session was returned instead of a new one.
	Resumed bool `json:"resumed"`
	// Guide holds the instructions for the agent to follow during this session.
	Guide string `json:"guide"`
	Tools []Tool `json:"tools"`
	// Usage is set when this call started the session and so used one run.
	Usage *Usage `json:"usage,omitempty"`
}

type ToolCallResult struct {
	SessionID          string          `json:"session_id"`
	Tool               string          `json:"tool"`
	Result             json.RawMessage `json:"result"`
	ToolCallsUsed      int             `json:"tool_calls_used"`
	ToolCallsRemaining int             `json:"tool_calls_remaining"`
	ExpiresAt          time.Time       `json:"expires_at"`
}

type RunResult struct {
	ExecutionID string `json:"execution_id"`
	SkillID     string `json:"skill_id"`
	SkillSlug   string `json:"skill_slug"`
	Version     string `json:"version"`
	Output      string `json:"output"`
	Usage       Usage  `json:"usage"`
}

// Account is the owner of the configured API key.
type Account struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	AuthType string `json:"auth_type"`
}

type APIError struct {
	Status  int
	Code    string
	Message string
	// RetryAfter is the delay the API asked for (429 and 503 responses), or zero.
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("SkillGild API request failed (HTTP %d)", e.Status)
	}
	return fmt.Sprintf("SkillGild API (%d, %s): %s", e.Status, e.Code, e.Message)
}

func NewClient(baseURL string, options ...Option) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return nil, fmt.Errorf("base URL must be an absolute HTTP(S) URL without user information")
	}
	client := &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 100 * time.Second}}
	for _, option := range options {
		option(client)
	}
	return client, nil
}

// ListSkills returns one page of the public catalog.
func (c *Client) ListSkills(ctx context.Context, options ListOptions) (SkillPage, error) {
	limit := options.Limit
	if limit <= 0 {
		limit = 100
	}
	params := url.Values{"limit": []string{strconv.Itoa(limit)}}
	if query := strings.TrimSpace(options.Query); query != "" {
		params.Set("q", query)
	}
	if options.Cursor != "" {
		params.Set("cursor", options.Cursor)
	}
	var page SkillPage
	err := c.request(ctx, http.MethodGet, "/skills?"+params.Encode(), nil, false, "", &page)
	return page, err
}

// SearchSkills returns the first 100 matching skills. Use ListSkills to page further.
func (c *Client) SearchSkills(ctx context.Context, query string) ([]Skill, error) {
	page, err := c.ListSkills(ctx, ListOptions{Query: query, Limit: 100})
	return page.Items, err
}

func (c *Client) GetSkill(ctx context.Context, idOrSlug string) (Skill, error) {
	var result Skill
	err := c.request(ctx, http.MethodGet, "/skills/"+url.PathEscape(idOrSlug), nil, false, "", &result)
	return result, err
}

func (c *Client) ListCategories(ctx context.Context) ([]Category, error) {
	var result []Category
	err := c.request(ctx, http.MethodGet, "/categories", nil, false, "", &result)
	return result, err
}

func (c *Client) ListTags(ctx context.Context) ([]Tag, error) {
	var result []Tag
	err := c.request(ctx, http.MethodGet, "/tags", nil, false, "", &result)
	return result, err
}

func (c *Client) ListCollections(ctx context.Context) ([]Collection, error) {
	var result []Collection
	err := c.request(ctx, http.MethodGet, "/collections", nil, false, "", &result)
	return result, err
}

func (c *Client) GetCollection(ctx context.Context, slug string) (CollectionDetail, error) {
	var result CollectionDetail
	err := c.request(ctx, http.MethodGet, "/collections/"+url.PathEscape(slug), nil, false, "", &result)
	return result, err
}

// FeaturedSkills returns the skills featured in a placement slot ("home" when empty).
func (c *Client) FeaturedSkills(ctx context.Context, slot string) ([]SkillSummary, error) {
	if slot == "" {
		slot = "home"
	}
	var result []SkillSummary
	err := c.request(ctx, http.MethodGet, "/featured-skills?"+url.Values{"slot": []string{slot}}.Encode(), nil, false, "", &result)
	return result, err
}

// RevokeCurrentKey revokes the API key this client authenticates with.
func (c *Client) RevokeCurrentKey(ctx context.Context) error {
	return c.request(ctx, http.MethodDelete, "/me/api-keys/current", nil, true, "", nil)
}

// StartDeviceAuthorization starts a device login. Show the user the code and URL,
// then call PollDeviceAuthorization every IntervalSeconds until it is authorized.
// clientType labels the device in the user's account ("skillgild-sdk" when empty).
func (c *Client) StartDeviceAuthorization(ctx context.Context, deviceName, clientType string) (DeviceAuthorization, error) {
	if clientType == "" {
		clientType = "skillgild-sdk"
	}
	var result DeviceAuthorization
	err := c.request(ctx, http.MethodPost, "/device-authorizations", map[string]string{"device_name": deviceName, "client_type": clientType}, false, "", &result)
	return result, err
}

func (c *Client) PollDeviceAuthorization(ctx context.Context, deviceCode string) (DeviceToken, error) {
	var result DeviceToken
	err := c.request(ctx, http.MethodPost, "/device-authorizations/token", map[string]string{"device_code": deviceCode}, false, "", &result)
	return result, err
}

// Me returns the account the API key belongs to.
func (c *Client) Me(ctx context.Context) (Account, error) {
	var result Account
	err := c.request(ctx, http.MethodGet, "/me", nil, true, "", &result)
	return result, err
}

// RunOption configures a single RunSkill call.
type RunOption func(*runOptions)

type runOptions struct{ idempotencyKey string }

// WithIdempotencyKey makes a run safe to retry: the API replays the stored result
// for 24 hours. Reuse a key only for the same request.
func WithIdempotencyKey(key string) RunOption {
	return func(o *runOptions) { o.idempotencyKey = key }
}

// RunSkill runs a hosted (prompt_pipeline) skill and uses one run from the caller's allowance.
func (c *Client) RunSkill(ctx context.Context, idOrSlug string, input map[string]any, options ...RunOption) (RunResult, error) {
	var opts runOptions
	for _, option := range options {
		option(&opts)
	}
	var result RunResult
	err := c.request(ctx, http.MethodPost, "/skills/"+url.PathEscape(idOrSlug)+"/run", map[string]any{"input": input}, true, opts.idempotencyKey, &result)
	return result, err
}

// StartSession starts (or resumes) a session for a hybrid_tools skill. A new
// session uses one run from the caller's allowance.
func (c *Client) StartSession(ctx context.Context, idOrSlug string) (AgentSession, error) {
	var result AgentSession
	err := c.request(ctx, http.MethodPost, "/skills/"+url.PathEscape(idOrSlug)+"/sessions", map[string]any{}, true, "", &result)
	return result, err
}

// CallTool calls a server tool in an open session. Each call uses one of the
// session's tool calls.
func (c *Client) CallTool(ctx context.Context, sessionID, tool string, input map[string]any) (ToolCallResult, error) {
	var result ToolCallResult
	err := c.request(ctx, http.MethodPost, "/skill-sessions/"+url.PathEscape(sessionID)+"/tools/"+url.PathEscape(tool), map[string]any{"input": input}, true, "", &result)
	return result, err
}

func (c *Client) EndSession(ctx context.Context, sessionID string) error {
	return c.request(ctx, http.MethodDelete, "/skill-sessions/"+url.PathEscape(sessionID), nil, true, "", nil)
}

func (c *Client) request(ctx context.Context, method, path string, body any, authenticated bool, idempotencyKey string, result any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode SkillGild request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create SkillGild request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "skillgild-go")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		if c.apiKey == "" {
			return ErrAPIKeyRequired
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("SkillGild API unavailable: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("read SkillGild response (HTTP %d): %w", response.StatusCode, err)
	}
	var envelope struct {
		Data  json.RawMessage `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	ok := response.StatusCode >= 200 && response.StatusCode < 300
	var retryAfter time.Duration
	if seconds, err := strconv.Atoi(response.Header.Get("Retry-After")); err == nil && seconds > 0 {
		retryAfter = time.Duration(seconds) * time.Second
	}
	if len(bytes.TrimSpace(raw)) == 0 || json.Unmarshal(raw, &envelope) != nil {
		if !ok {
			// A proxy answered instead of the API; the status is still meaningful.
			return &APIError{Status: response.StatusCode, RetryAfter: retryAfter}
		}
		return fmt.Errorf("SkillGild API returned an unexpected response (HTTP %d)", response.StatusCode)
	}
	if !ok {
		return &APIError{Status: response.StatusCode, Code: envelope.Error.Code, Message: envelope.Error.Message, RetryAfter: retryAfter}
	}
	if result != nil {
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return fmt.Errorf("decode SkillGild response: %w", err)
		}
	}
	return nil
}
