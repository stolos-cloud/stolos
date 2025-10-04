package gcp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	gcpconfig "github.com/stolos-cloud/stolos-bootstrap/pkg/gcp"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

type Client struct {
	config        *gcpconfig.GCPConfig
	computeClient *compute.Service
	storageClient *storage.Service
}

func NewClientFromEnv() (*Client, error) {
	config, err := configFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load config from env: %w", err)
	}

	return NewClientFromConfig(config)
}

func NewClientFromConfig(config *gcpconfig.GCPConfig) (*Client, error) {
	ctx := context.Background()
	credentials, err := google.CredentialsFromJSON(ctx, []byte(config.ServiceAccountJSON), compute.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP credentials: %w", err)
	}

	computeService, err := compute.NewService(ctx, option.WithCredentials(credentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	storageService, err := storage.NewService(ctx, option.WithCredentials(credentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}

	if config.ServiceAccountEmail == "" {
		config.ServiceAccountEmail = extractServiceAccountEmail(config.ServiceAccountJSON)
	}

	return &Client{
		config:        config,
		computeClient: computeService,
		storageClient: storageService,
	}, nil
}

func configFromEnv() (*gcpconfig.GCPConfig, error) {
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
		serviceAccountEmail = extractServiceAccountEmail(serviceAccountJSON)
	}

	return gcpconfig.NewConfig(projectID, region, serviceAccountJSON, serviceAccountEmail)
}

func (c *Client) GetProjectInfo() (projectID, region string) {
	return c.config.ProjectID, c.config.Region
}

func (c *Client) GetComputeService() *compute.Service {
	return c.computeClient
}

func (c *Client) GetConfig() *gcpconfig.GCPConfig {
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

func (c *Client) CreateTerraformBucket(ctx context.Context) (string, error) {
	bucketName := fmt.Sprintf("stolos-tf-state-%s", generateRandomString(8))

	bucket := &storage.Bucket{
		Name:     bucketName,
		Location: c.config.Region,
		Versioning: &storage.BucketVersioning{
			Enabled: true,
		},
		IamConfiguration: &storage.BucketIamConfiguration{
			UniformBucketLevelAccess: &storage.BucketIamConfigurationUniformBucketLevelAccess{
				Enabled: true,
			},
		},
	}

	_, err := c.storageClient.Buckets.Insert(c.config.ProjectID, bucket).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create storage bucket: %w", err)
	}

	return bucketName, nil
}

func extractServiceAccountEmail(serviceAccountJSON string) string {
	var sa struct {
		ClientEmail string `json:"client_email"`
	}
	json.Unmarshal([]byte(serviceAccountJSON), &sa)
	return sa.ClientEmail
}

func ExtractProjectIDFromServiceAccount(serviceAccountJSON []byte) (string, error) {
	var sa struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(serviceAccountJSON, &sa); err != nil {
		return "", err
	}
	if sa.ProjectID == "" {
		return "", fmt.Errorf("project_id not found in service account JSON")
	}
	return sa.ProjectID, nil
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
