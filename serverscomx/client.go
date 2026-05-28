package serverscomx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is a minimal REST client for the Servers.com Public API v1, scoped to
// PTR record CRUD on dedicated servers. The API surface is much larger; only
// the endpoints this provider needs are wrapped here.
type Client struct {
	Token      string
	BaseURL    string
	HTTPClient *RateLimitedHTTPClient
}

// PtrRecord mirrors the JSON shape returned by
// /hosts/dedicated_servers/{host_id}/ptr_records.
type PtrRecord struct {
	ID       string `json:"id"`
	IP       string `json:"ip"`
	Domain   string `json:"domain"`
	Priority int64  `json:"priority"`
	TTL      int64  `json:"ttl"`
}

// PtrCreateRequest is the payload accepted by POST .../ptr_records.
// Priority and TTL are optional — omitted (zero-value) lets the API pick
// defaults (priority=0, ttl=60).
type PtrCreateRequest struct {
	IP       string `json:"ip"`
	Domain   string `json:"domain"`
	Priority *int64 `json:"priority,omitempty"`
	TTL      *int64 `json:"ttl,omitempty"`
}

func NewClient(token, endpoint string, requestIntervalSeconds int64) *Client {
	return &Client{
		Token:      token,
		BaseURL:    endpoint,
		HTTPClient: NewRateLimitedHTTPClient(&http.Client{}, requestIntervalSeconds),
	}
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		buf = bytes.NewBuffer(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// CreatePtrRecord POSTs a new PTR record on the given dedicated server host.
func (c *Client) CreatePtrRecord(hostID string, in PtrCreateRequest) (*PtrRecord, error) {
	if hostID == "" {
		return nil, fmt.Errorf("host ID cannot be empty")
	}
	if in.IP == "" || in.Domain == "" {
		return nil, fmt.Errorf("ip and domain are required")
	}

	req, err := c.newRequest("POST", fmt.Sprintf("/hosts/dedicated_servers/%s/ptr_records", hostID), in)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var out PtrRecord
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &out, nil
}

// GetPtrRecord fetches a single PTR record by host + id. The Servers.com API
// has no single-record GET endpoint — we list the collection and filter by id.
// 404 from the collection is returned as `status 404` for callers to match on.
func (c *Client) GetPtrRecord(hostID, ptrID string) (*PtrRecord, error) {
	if hostID == "" {
		return nil, fmt.Errorf("host ID cannot be empty")
	}
	if ptrID == "" {
		return nil, fmt.Errorf("ptr ID cannot be empty")
	}

	records, err := c.ListPtrRecords(hostID)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == ptrID {
			return &records[i], nil
		}
	}
	return nil, fmt.Errorf("API request failed with status 404: ptr record %s not found on host %s", ptrID, hostID)
}

// ListPtrRecords paginates through all PTR records on a host.
// Pagination shape: ?per_page=100&page=N; stop when a page returns <100.
func (c *Client) ListPtrRecords(hostID string) ([]PtrRecord, error) {
	if hostID == "" {
		return nil, fmt.Errorf("host ID cannot be empty")
	}

	var out []PtrRecord
	page := 1
	for {
		req, err := c.newRequest("GET", fmt.Sprintf("/hosts/dedicated_servers/%s/ptr_records?per_page=100&page=%d", hostID, page), nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}
		var batch []PtrRecord
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		out = append(out, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}
	return out, nil
}

// DeletePtrRecord removes a PTR record. 404 is treated as success (idempotent).
func (c *Client) DeletePtrRecord(hostID, ptrID string) error {
	if hostID == "" {
		return fmt.Errorf("host ID cannot be empty")
	}
	if ptrID == "" {
		return fmt.Errorf("ptr ID cannot be empty")
	}

	req, err := c.newRequest("DELETE", fmt.Sprintf("/hosts/dedicated_servers/%s/ptr_records/%s", hostID, ptrID), nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
}
