package flymachines

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.machines.dev/v1"

// Client is a Fly Machines API client.
type Client struct {
	apiToken string
	appName  string // e.g. "regctl-sites"
	region   string // e.g. "nrt"
	http     *http.Client
}

// NewClient creates a new Fly Machines API client.
func NewClient(apiToken, appName, region string) *Client {
	return &Client{
		apiToken: apiToken,
		appName:  appName,
		region:   region,
		http:     &http.Client{Timeout: 60 * time.Second},
	}
}

// Machine represents a Fly Machine.
type Machine struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Region string `json:"region"`
}

type machineConfig struct {
	Image    string            `json:"image"`
	Env      map[string]string `json:"env,omitempty"`
	Guest    guest             `json:"guest"`
	Services []service         `json:"services"`
}

type guest struct {
	CPUKind  string `json:"cpu_kind"`
	CPUs     int    `json:"cpus"`
	MemoryMB int    `json:"memory_mb"`
}

type service struct {
	Ports        []port `json:"ports"`
	Protocol     string `json:"protocol"`
	InternalPort int    `json:"internal_port"`
}

type port struct {
	Port     int      `json:"port"`
	Handlers []string `json:"handlers"`
}

type createMachineReq struct {
	Name   string        `json:"name"`
	Region string        `json:"region"`
	Config machineConfig `json:"config"`
}

type updateMachineReq struct {
	Config machineConfig `json:"config"`
}

func (c *Client) do(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := fmt.Sprintf("%s/apps/%s%s", baseURL, c.appName, path)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fly api request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("fly api error %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// CreateMachine creates a new Fly Machine for a site.
func (c *Client) CreateMachine(name, domain string, env map[string]string) (string, error) {
	if env == nil {
		env = make(map[string]string)
	}
	env["SITE_DOMAIN"] = domain

	reqBody := createMachineReq{
		Name:   name,
		Region: c.region,
		Config: machineConfig{
			Image: "caddy:2-alpine",
			Env:   env,
			Guest: guest{
				CPUKind:  "shared",
				CPUs:     1,
				MemoryMB: 256,
			},
			Services: []service{
				{
					InternalPort: 80,
					Protocol:     "tcp",
					Ports: []port{
						{Port: 80, Handlers: []string{"http"}},
						{Port: 443, Handlers: []string{"tls", "http"}},
					},
				},
			},
		},
	}

	respBody, err := c.do("POST", "/machines", reqBody)
	if err != nil {
		return "", fmt.Errorf("create machine: %w", err)
	}

	var m Machine
	if err := json.Unmarshal(respBody, &m); err != nil {
		return "", fmt.Errorf("parse machine response: %w", err)
	}
	return m.ID, nil
}

// UpdateMachine updates a machine's environment variables.
func (c *Client) UpdateMachine(machineID string, env map[string]string) error {
	reqBody := updateMachineReq{
		Config: machineConfig{
			Image: "caddy:2-alpine",
			Env:   env,
			Guest: guest{
				CPUKind:  "shared",
				CPUs:     1,
				MemoryMB: 256,
			},
			Services: []service{
				{
					InternalPort: 80,
					Protocol:     "tcp",
					Ports: []port{
						{Port: 80, Handlers: []string{"http"}},
						{Port: 443, Handlers: []string{"tls", "http"}},
					},
				},
			},
		},
	}

	_, err := c.do("POST", "/machines/"+machineID, reqBody)
	if err != nil {
		return fmt.Errorf("update machine: %w", err)
	}
	return nil
}

// StopMachine stops a running machine.
func (c *Client) StopMachine(machineID string) error {
	_, err := c.do("POST", "/machines/"+machineID+"/stop", nil)
	if err != nil {
		return fmt.Errorf("stop machine: %w", err)
	}
	return nil
}

// StartMachine starts a stopped machine.
func (c *Client) StartMachine(machineID string) error {
	_, err := c.do("POST", "/machines/"+machineID+"/start", nil)
	if err != nil {
		return fmt.Errorf("start machine: %w", err)
	}
	return nil
}

// DeleteMachine permanently destroys a machine.
func (c *Client) DeleteMachine(machineID string) error {
	_, err := c.do("DELETE", "/machines/"+machineID+"?force=true", nil)
	if err != nil {
		return fmt.Errorf("delete machine: %w", err)
	}
	return nil
}

// AddCertificate adds a TLS certificate for a custom hostname.
// Uses the Fly platform API (not machines API).
func (c *Client) AddCertificate(appName, hostname string) error {
	body := map[string]string{"hostname": hostname}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.fly.io/v1/apps/%s/certificates", appName)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("fly cert request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("fly cert error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
