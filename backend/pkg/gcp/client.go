package gcp

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

type Config struct {
	ProjectID           string `json:"project_id"`
	Region              string `json:"region"`
	ServiceAccountJSON  string `json:"service_account_json"`
	ServiceAccountEmail string `json:"service_account_email"`
}

type Client struct {
	config        *Config
	computeClient *compute.Service
}

func NewClientFromEnv() (*Client, error) {
	config, err := configFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load config from env: %w", err)
	}

	ctx := context.Background()
	credentials, err := google.CredentialsFromJSON(ctx, []byte(config.ServiceAccountJSON), compute.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP credentials: %w", err)
	}

	computeService, err := compute.NewService(ctx, option.WithCredentials(credentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	return &Client{
		config:        config,
		computeClient: computeService,
	}, nil
}

func configFromEnv() (*Config, error) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("GCP_PROJECT_ID environment variable is required")
	}

	region := os.Getenv("GCP_REGION")
	if region == "" {
		return nil, fmt.Errorf("GCP_REGION environment variable is required")
	}

	serviceAccountJSON := os.Getenv("GCP_SERVICE_ACCOUNT_JSON")
	if serviceAccountJSON == "" {
		return nil, fmt.Errorf("GCP_SERVICE_ACCOUNT_JSON environment variable is required")
	}

	serviceAccountEmail := os.Getenv("GCP_SERVICE_ACCOUNT_EMAIL")
	if serviceAccountEmail == "" {
		return nil, fmt.Errorf("GCP_SERVICE_ACCOUNT_EMAIL environment variable is required")
	}

	return &Config{
		ProjectID:           projectID,
		Region:              region,
		ServiceAccountJSON:  serviceAccountJSON,
		ServiceAccountEmail: serviceAccountEmail,
	}, nil
}

func (c *Client) GetProjectInfo() (projectID, region string) {
	return c.config.ProjectID, c.config.Region
}

func (c *Client) GetComputeService() *compute.Service {
	return c.computeClient
}

func (c *Client) GetConfig() *Config {
	return c.config
}

func (c *Client) GetServiceAccountEmail() string {
	return c.config.ServiceAccountEmail
}

func (c *Client) ListInstancesInZone(ctx context.Context, zone string) ([]*compute.Instance, error) {
	resp, err := c.computeClient.Instances.List(c.config.ProjectID, zone).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	return resp.Items, nil
}

func (c *Client) ListAllInstances(ctx context.Context) (map[string][]*compute.Instance, error) {
	zones, err := c.getZonesInRegion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get zones: %w", err)
	}

	result := make(map[string][]*compute.Instance)
	for _, zone := range zones {
		instances, err := c.ListInstancesInZone(ctx, zone.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to list instances in zone %s: %w", zone.Name, err)
		}
		result[zone.Name] = instances
	}

	return result, nil
}

func (c *Client) getZonesInRegion(ctx context.Context) ([]*compute.Zone, error) {
	resp, err := c.computeClient.Zones.List(c.config.ProjectID).Context(ctx).Filter(fmt.Sprintf("region eq https://www.googleapis.com/compute/v1/projects/%s/regions/%s", c.config.ProjectID, c.config.Region)).Do()
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}
