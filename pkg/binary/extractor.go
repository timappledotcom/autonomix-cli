package binary

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractBinary extracts or copies binary from downloaded asset
func ExtractBinary(assetPath, expectedName string) (string, error) {
	lower := strings.ToLower(assetPath)

	if !isArchive(lower) {
		tmpDir := os.TempDir()
		destPath := filepath.Join(tmpDir, expectedName)
		
		if err := copyFile(assetPath, destPath); err != nil {
			return "", err
		}
		
		if err := os.Chmod(destPath, 0755); err != nil {
			return "", err
		}
		
		return destPath, nil
	}

	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return extractFromTarGz(assetPath, expectedName)
	}

	if strings.HasSuffix(lower, ".zip") {
		return extractFromZip(assetPath, expectedName)
	}

	return "", fmt.Errorf("unsupported format: %s", assetPath)
}

func extractFromTarGz(archivePath, expectedName string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	tmpDir := os.TempDir()

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		if !isExecutable(header.FileInfo().Mode()) {
			continue
		}

		baseName := filepath.Base(header.Name)
		if baseName == expectedName || strings.HasPrefix(baseName, expectedName) {
			destPath := filepath.Join(tmpDir, expectedName)
			
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return "", err
			}
			
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", err
			}
			out.Close()
			
			return destPath, nil
		}
	}

	return "", fmt.Errorf("binary %s not found in release archive", expectedName)
}

func extractFromZip(archivePath, expectedName string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	tmpDir := os.TempDir()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		if !isExecutable(f.Mode()) {
			continue
		}

		baseName := filepath.Base(f.Name)
		if baseName == expectedName || strings.HasPrefix(baseName, expectedName) {
			destPath := filepath.Join(tmpDir, expectedName)
			
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				rc.Close()
				return "", err
			}
			
			if _, err := io.Copy(out, rc); err != nil {
				out.Close()
				rc.Close()
				return "", err
			}
			out.Close()
			rc.Close()
			
			return destPath, nil
		}
	}

	return "", fmt.Errorf("binary %s not found in release archive", expectedName)
}

func isExecutable(mode os.FileMode) bool {
	return mode&0111 != 0
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
