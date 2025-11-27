package talos

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/storage"
	"github.com/siderolabs/image-factory/pkg/schematic"
	"github.com/stolos-cloud/stolos/backend/internal/helpers"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"google.golang.org/api/option"
)

// EnsureTalosGCPImages ensures Talos images are uploaded and registered in GCP
// This is called during provider initialization
func (s *TalosService) EnsureTalosGCPImages(ctx context.Context, gcpConfig *models.GCPConfig) error {
	log.Printf("Checking Talos images for GCP (version: %s)", gcpConfig.TalosVersion)

	// Check if AMD64 image already exists (we skip ARM64 for now..)
	if gcpConfig.TalosImageAMD64 != "" {
		log.Println("Talos images already configured")
		return nil
	}

	log.Println("Talos AMD64 image not found, starting upload process...")

	// Process AMD64 image if missing
	if gcpConfig.TalosImageAMD64 == "" {
		imageName, err := s.uploadAndRegisterTalosImage(ctx, gcpConfig, "amd64")
		if err != nil {
			return fmt.Errorf("failed to upload AMD64 image: %w", err)
		}
		gcpConfig.TalosImageAMD64 = imageName

		// Update database
		if err := s.db.Model(&models.GCPConfig{}).
			Where("id = ?", gcpConfig.ID).
			Update("talos_image_amd64", imageName).Error; err != nil {
			return fmt.Errorf("failed to update AMD64 image name in database: %w", err)
		}
		log.Printf("AMD64 image registered: %s", imageName)
	}

	// Process ARM64 image if missing
	// For now, skip ARM64 to save time during initialization
	log.Println("Skipping ARM64 image")

	log.Println("Talos GCP images configured successfully")
	return nil
}

// uploadAndRegisterTalosImage downloads, uploads, and registers a Talos image
func (s *TalosService) uploadAndRegisterTalosImage(ctx context.Context, gcpConfig *models.GCPConfig, arch string) (string, error) {
	version := gcpConfig.TalosVersion
	if version == "" {
		version = "v1.11.1"
	}

	log.Printf("Processing Talos %s image for %s...", version, arch)

	localPath, err := s.downloadTalosGCPImage(ctx, version, arch)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(localPath)) // Clean up temp directory

	log.Printf("Downloaded Talos image to: %s", localPath)

	// Upload to GCS bucket
	gcsPath := fmt.Sprintf("talos-images/talos-%s-%s.raw.tar.gz", version, arch)
	if err := s.uploadToGCS(ctx, gcpConfig, localPath, gcsPath); err != nil {
		return "", fmt.Errorf("failed to upload to GCS: %w", err)
	}

	log.Printf("Uploaded to GCS: gs://%s/%s", gcpConfig.BucketName, gcsPath)

	// Register as GCP compute image with cluster name prefix
	// Format: <cluster>-talos-<version>-<arch> (e.g., prod-talos-1-11-1-amd64)
	clusterPrefix := helpers.SanitizeResourceName(s.cfg.ClusterName)
	imageName := fmt.Sprintf("%s-talos-%s-%s", clusterPrefix, strings.TrimPrefix(version, "v"), arch)
	imageName = strings.ReplaceAll(imageName, ".", "-") // e.g., prod-talos-1-11-1-amd64

	if err := s.registerGCPImage(ctx, gcpConfig, imageName, gcsPath); err != nil {
		return "", fmt.Errorf("failed to register image: %w", err)
	}

	log.Printf("Registered GCP image: %s", imageName)

	return imageName, nil
}

// downloadTalosGCPImage downloads a Talos GCP image from Talos Image Factory
func (s *TalosService) downloadTalosGCPImage(ctx context.Context, version, arch string) (string, error) {
	// Create a schematic (using default for now. no customizations needed for GCP base image)
	schematicID, err := s.factoryClient.SchematicCreate(ctx, s.createDefaultSchematic())
	if err != nil {
		return "", fmt.Errorf("failed to create schematic: %w", err)
	}

	log.Printf("Created schematic: %s", schematicID)

	// Format: https://factory.talos.dev/image/{schematicID}/{version}/gcp-{arch}.raw.tar.gz
	url := fmt.Sprintf("https://factory.talos.dev/image/%s/%s/gcp-%s.raw.tar.gz", schematicID, version, arch)

	log.Printf("Downloading from Image Factory: %s", url)

	tempDir, err := os.MkdirTemp("", "talos-image-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	localPath := filepath.Join(tempDir, fmt.Sprintf("gcp-%s.raw.tar.gz", arch))

	// Download file
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Create local file
	out, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	// Copy data
	size, err := io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write image data: %w", err)
	}

	log.Printf("Downloaded %d bytes", size)

	return localPath, nil
}

// uploadToGCS uploads a file to Google Cloud Storage
func (s *TalosService) uploadToGCS(ctx context.Context, gcpConfig *models.GCPConfig, localPath, gcsPath string) error {
	// Create GCS client with service account credentials
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(gcpConfig.ServiceAccountKeyJSON)))
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Get file info for size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	log.Printf("Uploading %d bytes to GCS...", stat.Size())

	// Create object writer
	bucket := client.Bucket(gcpConfig.BucketName)
	obj := bucket.Object(gcsPath)
	writer := obj.NewWriter(ctx)

	// Copy file to GCS
	if _, err := io.Copy(writer, file); err != nil {
		writer.Close()
		return fmt.Errorf("failed to upload to GCS: %w", err)
	}

	// Close writer. this completes the upload
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize upload: %w", err)
	}

	log.Println("Upload to GCS completed")

	return nil
}

// createDefaultSchematic returns a minimal schematic for GCP base images
// GCP images don't need kernel args or overlays - just use the default Talos image
func (s *TalosService) createDefaultSchematic() schematic.Schematic {
	return schematic.Schematic{
		Customization: schematic.Customization{
			ExtraKernelArgs: []string{}, // Empty - no customizations needed for GCP base images
		},
	}
}

// registerGCPImage creates a GCP compute image from a GCS tarball
func (s *TalosService) registerGCPImage(ctx context.Context, gcpConfig *models.GCPConfig, imageName, gcsPath string) error {
	log.Printf("Registering GCP image: %s", imageName)

	// Create compute client with service account credentials
	client, err := compute.NewImagesRESTClient(ctx, option.WithCredentialsJSON([]byte(gcpConfig.ServiceAccountKeyJSON)))
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}
	defer client.Close()

	// Check if image already exists
	_, err = client.Get(ctx, &computepb.GetImageRequest{
		Project: gcpConfig.ProjectID,
		Image:   imageName,
	})
	if err == nil {
		log.Printf("Image %s already exists, skipping registration", imageName)
		return nil
	}

	// Construct GCS source URI
	sourceURI := fmt.Sprintf("https://storage.googleapis.com/%s/%s", gcpConfig.BucketName, gcsPath)

	// Create image
	req := &computepb.InsertImageRequest{
		Project: gcpConfig.ProjectID,
		ImageResource: &computepb.Image{
			Name: &imageName,
			RawDisk: &computepb.RawDisk{
				Source: &sourceURI,
			},
			GuestOsFeatures: []*computepb.GuestOsFeature{
				{
					Type: func() *string { s := "VIRTIO_SCSI_MULTIQUEUE"; return &s }(),
				},
			},
		},
	}

	log.Println("Creating GCP compute image (this may take several minutes)...")

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	// Wait for operation to complete
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("image creation failed: %w", err)
	}

	log.Printf("GCP image %s created successfully", imageName)

	return nil
}
