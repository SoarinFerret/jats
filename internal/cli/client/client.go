package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/soarinferret/jats/internal/cli/config"
	"github.com/soarinferret/jats/internal/models"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	apiToken   string
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
	Session struct {
		ID        uint      `json:"id"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"session"`
}

type APIKeyRequest struct {
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type APIKeyResponse struct {
	APIKey struct {
		ID          uint      `json:"id"`
		Name        string    `json:"name"`
		Permissions []string  `json:"permissions"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	} `json:"api_key"`
	Key string `json:"key"`
}

type CreateTaskRequest struct {
	Name     string   `json:"name"`
	Priority string   `json:"priority,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Date     string   `json:"date,omitempty"`
}

type LogTimeRequest struct {
	Duration    int    `json:"duration"` // minutes
	Description string `json:"description,omitempty"`
	Date        string `json:"date,omitempty"`
}

func New() *Client {
	cfg := config.GetCurrent()
	if cfg == nil {
		cfg = &config.Config{ServerURL: "http://localhost:8081"}
	}

	// Create cookie jar for session-based auth during login
	jar, _ := cookiejar.New(nil)

	return &Client{
		baseURL:  strings.TrimSuffix(cfg.ServerURL, "/"),
		apiToken: cfg.Token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}
}

func (c *Client) Login(username, password string) (*LoginResponse, error) {
	// Step 1: Login with session to get authenticated
	req := LoginRequest{
		Username: username,
		Password: password,
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    LoginResponse `json:"data"`
		Message string `json:"message"`
	}
	
	err := c.post("/api/v1/auth/login", req, &loginResp)
	if err != nil {
		return nil, err
	}

	if !loginResp.Success {
		return nil, fmt.Errorf("login failed: %s", loginResp.Message)
	}

	// Extract session token from cookies and save it
	baseURL, _ := url.Parse(c.baseURL)
	cookies := c.httpClient.Jar.Cookies(baseURL)
	var sessionToken string
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			sessionToken = cookie.Value
			break
		}
	}

	if sessionToken == "" {
		return nil, fmt.Errorf("login succeeded but no session token found")
	}

	// Update config with session token and username
	cfg := config.GetCurrent()
	if cfg != nil {
		cfg.Username = username
		cfg.Token = sessionToken
	}

	// Update client token for immediate use
	c.apiToken = sessionToken

	return &loginResp.Data, nil
}

func (c *Client) CreateTask(req *CreateTaskRequest) (*models.Task, error) {
	var apiResp struct {
		Success bool `json:"success"`
		Data    models.Task `json:"data"`
		Message string `json:"message"`
	}
	
	err := c.post("/api/v1/tasks", req, &apiResp)
	if err != nil {
		return nil, err
	}
	
	if !apiResp.Success {
		return nil, fmt.Errorf("create task failed: %s", apiResp.Message)
	}
	
	return &apiResp.Data, nil
}

type TaskFilters struct {
	Status   []string `json:"status,omitempty"`
	Priority []string `json:"priority,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Search   string   `json:"search,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Offset   int      `json:"offset,omitempty"`
}

type Task struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Status      models.TaskStatus `json:"status"`
	Priority    models.TaskPriority `json:"priority"`
	Tags        []string          `json:"tags"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	TimeEntries []TimeEntry       `json:"time_entries"`
	Comments    []Comment         `json:"comments"`
}

type TimeEntry struct {
	ID          uint      `json:"id"`
	Description string    `json:"description"`
	Duration    int       `json:"duration"`
	CreatedAt   time.Time `json:"created_at"`
}

type Comment struct {
	ID        uint      `json:"id"`
	Content   string    `json:"content"`
	IsPrivate bool      `json:"is_private"`
	CreatedAt time.Time `json:"created_at"`
}

type SavedQuery struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	IncludedTags []string  `json:"included_tags"`
	ExcludedTags []string  `json:"excluded_tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (c *Client) GetTasks(filters *TaskFilters) ([]Task, error) {
	query := url.Values{}
	
	if filters != nil {
		if len(filters.Status) > 0 {
			query.Add("status", strings.Join(filters.Status, ","))
		}
		if len(filters.Priority) > 0 {
			query.Add("priority", strings.Join(filters.Priority, ","))
		}
		if len(filters.Tags) > 0 {
			query.Add("tags", strings.Join(filters.Tags, ","))
		}
		if filters.Search != "" {
			query.Add("search", filters.Search)
		}
		if filters.Limit > 0 {
			query.Add("limit", strconv.Itoa(filters.Limit))
		}
		if filters.Offset > 0 {
			query.Add("offset", strconv.Itoa(filters.Offset))
		}
	}

	endpoint := "/api/v1/tasks"
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var apiResp struct {
		Success bool `json:"success"`
		Data    struct {
			Items      []Task `json:"items"`
			Pagination struct {
				Total  int `json:"total"`
				Limit  int `json:"limit"`
				Offset int `json:"offset"`
				Pages  int `json:"pages"`
			} `json:"pagination"`
		} `json:"data"`
		Message string `json:"message"`
	}
	
	err := c.get(endpoint, &apiResp)
	if err != nil {
		return nil, err
	}
	
	if !apiResp.Success {
		return nil, fmt.Errorf("get tasks failed: %s", apiResp.Message)
	}
	
	return apiResp.Data.Items, nil
}

func (c *Client) GetTask(id uint) (*models.Task, error) {
	var apiResp struct {
		Success bool `json:"success"`
		Data    models.Task `json:"data"`
		Message string `json:"message"`
	}
	
	err := c.get(fmt.Sprintf("/api/v1/tasks/%d", id), &apiResp)
	if err != nil {
		return nil, err
	}
	
	if !apiResp.Success {
		return nil, fmt.Errorf("get task failed: %s", apiResp.Message)
	}
	
	return &apiResp.Data, nil
}

func (c *Client) UpdateTaskStatus(id uint, status string) (*models.Task, error) {
	req := map[string]interface{}{
		"status": status,
	}
	
	var apiResp struct {
		Success bool `json:"success"`
		Data    models.Task `json:"data"`
		Message string `json:"message"`
	}
	
	err := c.patch(fmt.Sprintf("/api/v1/tasks/%d", id), req, &apiResp)
	if err != nil {
		return nil, err
	}
	
	if !apiResp.Success {
		return nil, fmt.Errorf("update task failed: %s", apiResp.Message)
	}
	
	return &apiResp.Data, nil
}

func (c *Client) LogTime(taskID uint, req *LogTimeRequest) error {
	var apiResp struct {
		Success bool `json:"success"`
		Message string `json:"message"`
	}
	
	err := c.post(fmt.Sprintf("/api/v1/tasks/%d/time", taskID), req, &apiResp)
	if err != nil {
		return err
	}
	
	if !apiResp.Success {
		return fmt.Errorf("log time failed: %s", apiResp.Message)
	}
	
	return nil
}

func (c *Client) GetSavedQueries() ([]SavedQuery, error) {
	var apiResp struct {
		Success bool `json:"success"`
		Data    []SavedQuery `json:"data"`
		Message string `json:"message"`
	}
	
	err := c.get("/api/v1/saved-queries", &apiResp)
	if err != nil {
		return nil, err
	}
	
	if !apiResp.Success {
		return nil, fmt.Errorf("get saved queries failed: %s", apiResp.Message)
	}
	
	return apiResp.Data, nil
}

type CreateSavedQueryRequest struct {
	Name         string   `json:"name"`
	IncludedTags []string `json:"included_tags,omitempty"`
	ExcludedTags []string `json:"excluded_tags,omitempty"`
}

func (c *Client) CreateSavedQuery(req *CreateSavedQueryRequest) (*SavedQuery, error) {
	var apiResp struct {
		Success bool `json:"success"`
		Data    SavedQuery `json:"data"`
		Message string `json:"message"`
	}
	
	err := c.post("/api/v1/saved-queries", req, &apiResp)
	if err != nil {
		return nil, err
	}
	
	if !apiResp.Success {
		return nil, fmt.Errorf("create saved query failed: %s", apiResp.Message)
	}
	
	return &apiResp.Data, nil
}

type AddCommentRequest struct {
	Content   string `json:"content"`
	IsPrivate bool   `json:"is_private,omitempty"`
}

func (c *Client) AddComment(taskID uint, req *AddCommentRequest) error {
	var apiResp struct {
		Success bool `json:"success"`
		Message string `json:"message"`
	}
	
	err := c.post(fmt.Sprintf("/api/v1/tasks/%d/comments", taskID), req, &apiResp)
	if err != nil {
		return err
	}
	
	if !apiResp.Success {
		return fmt.Errorf("add comment failed: %s", apiResp.Message)
	}
	
	return nil
}

type UpdateTaskRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type TaskSummaryResponse struct {
	OpenTasks             int `json:"open_tasks"`
	InProgressTasks       int `json:"in_progress_tasks"`
	RecentlyAddedTasks    int `json:"recently_added_tasks"`
	RecentlyResolvedTasks int `json:"recently_resolved_tasks"`
}

func (c *Client) UpdateTask(taskID uint, req *UpdateTaskRequest) (*Task, error) {
	var apiResp struct {
		Success bool `json:"success"`
		Data    Task `json:"data"`
		Message string `json:"message"`
	}
	
	err := c.patch(fmt.Sprintf("/api/v1/tasks/%d", taskID), req, &apiResp)
	if err != nil {
		return nil, err
	}
	
	if !apiResp.Success {
		return nil, fmt.Errorf("update task failed: %s", apiResp.Message)
	}
	
	return &apiResp.Data, nil
}

func (c *Client) GetTaskSummary(savedQueryID *uint) (*TaskSummaryResponse, error) {
	endpoint := "/api/v1/summary/tasks"
	if savedQueryID != nil {
		endpoint += fmt.Sprintf("?saved_query_id=%d", *savedQueryID)
	}

	var apiResp struct {
		Success bool                 `json:"success"`
		Data    TaskSummaryResponse  `json:"data"`
		Message string               `json:"message"`
	}

	err := c.get(endpoint, &apiResp)
	if err != nil {
		return nil, err
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("get task summary failed: %s", apiResp.Message)
	}

	return &apiResp.Data, nil
}

func (c *Client) get(endpoint string, response interface{}) error {
	return c.request("GET", endpoint, nil, response)
}

func (c *Client) post(endpoint string, body interface{}, response interface{}) error {
	return c.request("POST", endpoint, body, response)
}

func (c *Client) put(endpoint string, body interface{}, response interface{}) error {
	return c.request("PUT", endpoint, body, response)
}

func (c *Client) patch(endpoint string, body interface{}, response interface{}) error {
	return c.request("PATCH", endpoint, body, response)
}

func (c *Client) delete(endpoint string) error {
	return c.request("DELETE", endpoint, nil, nil)
}

// Convenience methods for admin operations
func (c *Client) Get(endpoint string) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.get(endpoint, &response)
	return response, err
}

func (c *Client) Post(endpoint string, body interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.post(endpoint, body, &response)
	return response, err
}

func (c *Client) Put(endpoint string, body interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.put(endpoint, body, &response)
	return response, err
}

func (c *Client) Delete(endpoint string) error {
	return c.delete(endpoint)
}

func (c *Client) request(method, endpoint string, body interface{}, response interface{}) error {
	var reqBody io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Use session token if available (will be treated as Bearer token)
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// ParseDuration parses duration strings like "30m", "1h", "2h30m" and returns minutes
func ParseDuration(duration string) (int, error) {
	// Try parsing as Go duration first
	if d, err := time.ParseDuration(duration); err == nil {
		return int(d.Minutes()), nil
	}

	// Handle formats like "1.5h" or just numbers (assume minutes)
	if strings.Contains(duration, ".") {
		// Try parsing as decimal hours
		if strings.HasSuffix(duration, "h") {
			hourStr := strings.TrimSuffix(duration, "h")
			if hours, err := strconv.ParseFloat(hourStr, 64); err == nil {
				return int(hours * 60), nil
			}
		}
	}

	// If just a number, assume minutes
	if num, err := strconv.Atoi(duration); err == nil {
		return num, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s (examples: 30m, 1h, 2h30m, 1.5h, or just 30 for minutes)", duration)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}