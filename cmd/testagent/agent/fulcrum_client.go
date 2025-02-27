package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

// FulcrumClient handles communication with the Fulcrum Core API
type FulcrumClient struct {
	baseURL    string
	httpClient *http.Client
	token      string // Agent authentication token
}

// NewFulcrumClient creates a new Fulcrum API client
func NewFulcrumClient(baseURL string, token string) *FulcrumClient {
	return &FulcrumClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UpdateAgentStatus updates the agent's status in Fulcrum Core
func (c *FulcrumClient) UpdateAgentStatus(status string) error {
	reqBody, err := json.Marshal(map[string]interface{}{
		"state": status,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal status update request: %w", err)
	}

	resp, err := c.patch("/api/v1/agents/me/status", reqBody)
	if err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update agent status, status: %d", resp.StatusCode)
	}

	return nil
}

// GetAgentInfo retrieves the agent's information from Fulcrum Core
func (c *FulcrumClient) GetAgentInfo() (map[string]interface{}, error) {
	resp, err := c.get("/api/v1/agents/me")
	if err != nil {
		return nil, fmt.Errorf("failed to get agent info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get agent info, status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode agent info response: %w", err)
	}

	return result, nil
}

// JobType represents the type of job
type JobType string

const (
	JobTypeServiceCreate JobType = "ServiceCreate"
	JobTypeServiceUpdate JobType = "ServiceUpdate"
	JobTypeServiceDelete JobType = "ServiceDelete"
)

// JobState represents the state of a job
type JobState string

const (
	JobStatePending    JobState = "Pending"
	JobStateProcessing JobState = "Processing"
	JobStateCompleted  JobState = "Completed"
	JobStateFailed     JobState = "Failed"
)

// Job represents a job from the Fulcrum Core job queue
type Job struct {
	ID          string          `json:"id"`
	Type        JobType         `json:"type"`
	State       JobState        `json:"state"`
	AgentID     string          `json:"agentId"`
	ServiceID   string          `json:"serviceId"`
	Priority    int             `json:"priority"`
	RequestData json.RawMessage `json:"requestData"`
}

// GetPendingJobs retrieves pending jobs for this agent
func (c *FulcrumClient) GetPendingJobs() ([]Job, error) {
	resp, err := c.get("/api/v1/jobs/pending")
	if err != nil {
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get pending jobs, status: %d", resp.StatusCode)
	}

	var result struct {
		Jobs []Job `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode jobs response: %w", err)
	}

	return result.Jobs, nil
}

// ClaimJob claims a job for processing
func (c *FulcrumClient) ClaimJob(jobID string) error {
	resp, err := c.post(fmt.Sprintf("/api/v1/jobs/%s/claim", jobID), nil, "")
	if err != nil {
		return fmt.Errorf("failed to claim job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to claim job, status: %d", resp.StatusCode)
	}

	return nil
}

// CompleteJob marks a job as completed with results
func (c *FulcrumClient) CompleteJob(jobID string, resultData map[string]interface{}) error {
	reqBody, err := json.Marshal(map[string]interface{}{
		"resultData": resultData,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal job completion request: %w", err)
	}

	resp, err := c.post(fmt.Sprintf("/api/v1/jobs/%s/complete", jobID), reqBody, "")
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to complete job, status: %d", resp.StatusCode)
	}

	return nil
}

// FailJob marks a job as failed with an error message
func (c *FulcrumClient) FailJob(jobID string, errorMessage string) error {
	reqBody, err := json.Marshal(map[string]interface{}{
		"errorMessage": errorMessage,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal job failure request: %w", err)
	}

	resp, err := c.post(fmt.Sprintf("/api/v1/jobs/%s/fail", jobID), reqBody, "")
	if err != nil {
		return fmt.Errorf("failed to mark job as failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to mark job as failed, status: %d", resp.StatusCode)
	}

	return nil
}

// ReportMetrics sends collected metrics to Fulcrum Core
func (c *FulcrumClient) ReportMetrics(metrics []MetricEntry) error {
	reqBody, err := json.Marshal(map[string]interface{}{
		"metrics": metrics,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal metrics request: %w", err)
	}

	resp, err := c.post("/api/v1/metrics", reqBody, "")
	if err != nil {
		return fmt.Errorf("failed to report metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to report metrics, status: %d", resp.StatusCode)
	}

	return nil
}

// Helper methods for HTTP requests
func (c *FulcrumClient) get(endpoint string) (*http.Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, endpoint)

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *FulcrumClient) post(endpoint string, body []byte, contentType string) (*http.Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, endpoint)

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)

	return c.httpClient.Do(req)
}

func (c *FulcrumClient) patch(endpoint string, body []byte) (*http.Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, endpoint)

	req, err := http.NewRequest(http.MethodPatch, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *FulcrumClient) delete(endpoint string) (*http.Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, endpoint)

	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}
