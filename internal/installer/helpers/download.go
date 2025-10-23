package helpers

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadBinary downloads a binary from a URL to the destination path.
// Supports direct binaries and .tar.gz archives.
// For archives, use ExtractFromTarGz after downloading.
func DownloadBinary(ctx context.Context, url, destPath string) error {
	// Create temp file for download
	tmpFile, err := os.CreateTemp("", "runlite-download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("download: nil response")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Copy response to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("save download: %w", err)
	}

	tmpFile.Close()

	// If URL ends with .tar.gz, keep as archive for extraction
	// Otherwise, make executable and move to destination
	if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		// Rename temp file to keep extension
		archivePath := tmpFile.Name() + ".tar.gz"
		if err := os.Rename(tmpFile.Name(), archivePath); err != nil {
			return fmt.Errorf("rename archive: %w", err)
		}
		// Caller will handle extraction
		return nil
	}

	// Make executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	// Atomic move to destination
	if err := os.Rename(tmpFile.Name(), destPath); err != nil {
		return fmt.Errorf("move to destination: %w", err)
	}

	return nil
}

// ExtractFromTarGz extracts a specific file from a .tar.gz archive.
// fileName is the name of the file to extract (e.g., "caddy", "litestream").
// destPath is where to place the extracted file.
func ExtractFromTarGz(archivePath, fileName, destPath string) error {
	// Open archive
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	// Create gzip reader
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Find and extract the target file
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		// Check if this is the file we want
		if header.Typeflag == tar.TypeReg && strings.HasSuffix(header.Name, fileName) {
			// Create temp file
			tmpFile, err := os.CreateTemp("", "runlite-extract-*")
			if err != nil {
				return fmt.Errorf("create temp file: %w", err)
			}
			defer os.Remove(tmpFile.Name())

			// Copy file contents
			if _, err := io.Copy(tmpFile, tr); err != nil {
				tmpFile.Close()
				return fmt.Errorf("extract file: %w", err)
			}
			tmpFile.Close()

			// Make executable
			if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
				return fmt.Errorf("chmod: %w", err)
			}

			// Ensure destination directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("create dest dir: %w", err)
			}

			// Atomic move to destination
			if err := os.Rename(tmpFile.Name(), destPath); err != nil {
				return fmt.Errorf("move to destination: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("file %q not found in archive", fileName)
}
