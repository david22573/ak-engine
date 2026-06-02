package data

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Endpoint        string
	CacheDir        string
}

func LoadR2Config() (R2Config, error) {
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("R2_BUCKET_NAME")

	var missing []string
	if accountID == "" {
		missing = append(missing, "R2_ACCOUNT_ID")
	}
	if accessKeyID == "" {
		missing = append(missing, "R2_ACCESS_KEY_ID")
	}
	if secretAccessKey == "" {
		missing = append(missing, "R2_SECRET_ACCESS_KEY")
	}
	if bucketName == "" {
		missing = append(missing, "R2_BUCKET_NAME")
	}

	if len(missing) > 0 {
		return R2Config{}, fmt.Errorf("missing required environment variables: %v", missing)
	}

	endpoint := os.Getenv("R2_ENDPOINT")
	cacheDir := os.Getenv("AK_ENGINE_R2_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = ".ak-engine/cache/r2"
	}

	return R2Config{
		AccountID:       accountID,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		BucketName:      bucketName,
		Endpoint:        endpoint,
		CacheDir:        cacheDir,
	}, nil
}

func NewS3Client(ctx context.Context, cfg R2Config) (*s3.Client, error) {
	var endpoint string
	if cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	} else {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           endpoint,
			SigningRegion: "auto",
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return client, nil
}
