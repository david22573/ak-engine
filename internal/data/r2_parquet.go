package data

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type S3Downloader interface {
	Download(ctx context.Context, key string, destPath string) error
}

type AWSS3Downloader struct {
	client *s3.Client
	bucket string
}

func (d *AWSS3Downloader) Download(ctx context.Context, key string, destPath string) error {
	result, err := d.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer result.Body.Close()

	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directories for %s: %w", destPath, err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, result.Body); err != nil {
		return fmt.Errorf("failed to write object content to %s: %w", destPath, err)
	}

	return nil
}

type R2ParquetSource struct {
	downloader S3Downloader
	cacheDir   string
	cache      *Cache
	filter     *ObjectFilter
}

func NewR2ParquetSource() *R2ParquetSource {
	return &R2ParquetSource{
		filter: NewObjectFilter(),
	}
}

func (s *R2ParquetSource) Name() string {
	return "r2"
}

func (s *R2ParquetSource) initClient(ctx context.Context) (S3Downloader, *Cache, error) {
	if s.cache == nil {
		if s.cacheDir != "" {
			s.cache = NewCache(s.cacheDir)
		}
	}

	if s.downloader != nil && s.cache != nil {
		return s.downloader, s.cache, nil
	}

	cfg, err := LoadR2Config()
	if err != nil {
		return nil, nil, err
	}

	client, err := NewS3Client(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	if s.downloader == nil {
		s.downloader = &AWSS3Downloader{
			client: client,
			bucket: cfg.BucketName,
		}
	}
	if s.cache == nil {
		s.cache = NewCache(cfg.CacheDir)
	}

	return s.downloader, s.cache, nil
}

func (s *R2ParquetSource) FilterObjects(objects []ObjectStats, req CandleRequest) ([]ObjectStats, error) {
	if s.filter == nil {
		s.filter = NewObjectFilter()
	}
	return s.filter.Filter(objects, req)
}

func (s *R2ParquetSource) LoadCandles(ctx context.Context, req CandleRequest) ([]protocol.Candle, error) {
	if req.Market == "" || req.Symbol == "" || req.Interval == "" || req.From.IsZero() || req.To.IsZero() {
		return nil, fmt.Errorf("missing required candle request fields")
	}

	downloader, cache, err := s.initClient(ctx)
	if err != nil {
		return nil, err
	}

	manifestKey := fmt.Sprintf("manifests/%s/%s/symbol=%s/manifest.json", req.Market, req.Interval, req.Symbol)

	// Resolve path using Cache to prevent path traversal
	manifestPath, err := cache.ResolvePath(manifestKey)
	if err != nil {
		return nil, err
	}

	var manifestBytes []byte
	if _, err := os.Stat(manifestPath); err == nil {
		manifestBytes, err = os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read cached manifest: %w", err)
		}
	} else {
		// Ensure parent dir exists before downloading
		if err := cache.EnsureDir(manifestPath); err != nil {
			return nil, err
		}
		if err := downloader.Download(ctx, manifestKey, manifestPath); err != nil {
			return nil, fmt.Errorf("manifest not found: %w", err)
		}
		manifestBytes, err = os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read downloaded manifest: %w", err)
		}
	}

	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return nil, err
	}

	if err := manifest.ValidateForRequest(req); err != nil {
		return nil, err
	}

	matchedObjects, err := s.FilterObjects(manifest.Objects, req)
	if err != nil {
		return nil, err
	}

	if len(matchedObjects) == 0 {
		return nil, fmt.Errorf("no matching objects found for the requested range")
	}

	var allCandles []protocol.Candle
	for _, obj := range matchedObjects {
		localPath, err := cache.ResolvePath(obj.Key)
		if err != nil {
			return nil, err
		}

		if _, err := os.Stat(localPath); err != nil {
			if err := cache.EnsureDir(localPath); err != nil {
				return nil, err
			}
			if err := downloader.Download(ctx, obj.Key, localPath); err != nil {
				return nil, fmt.Errorf("object download failure for %s: %w", obj.Key, err)
			}
		}

		candles, err := readParquetFile(localPath, req)
		if err != nil {
			return nil, fmt.Errorf("invalid parquet schema or unreadable parquet file %s: %w", obj.Key, err)
		}
		allCandles = append(allCandles, candles...)
	}

	if len(allCandles) == 0 {
		return nil, fmt.Errorf("empty candle result")
	}

	// Sort candles by OpenTimeMS before validating
	sort.Slice(allCandles, func(i, j int) bool {
		return allCandles[i].OpenTimeMS < allCandles[j].OpenTimeMS
	})

	// Filter candles inside the files by exact requested From/To
	var filtered []protocol.Candle
	for _, c := range allCandles {
		if !req.From.IsZero() && c.OpenTimeMS < req.From.UnixMilli() {
			continue
		}
		if !req.To.IsZero() && c.OpenTimeMS > req.To.UnixMilli() {
			continue
		}
		filtered = append(filtered, c)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("empty candle result")
	}

	// Validate using existing ValidateCandles
	if err := ValidateCandles(req.Interval, filtered); err != nil {
		return nil, err
	}

	return filtered, nil
}
