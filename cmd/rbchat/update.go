package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoOwner = "Esteban-Bermudez"
	repoName  = "rbchat"
)

var supportedPlatform = map[string]bool{
	"darwin/amd64":  true,
	"darwin/arm64":  true,
	"linux/amd64":   true,
	"windows/amd64": true,
	"windows/arm64": true,
}

func cmdUpdate() {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if !supportedPlatform[platform] {
		fmt.Fprintf(os.Stderr, "Unsupported platform: %s\n", platform)
		os.Exit(1)
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Checking for updates...")

	latest, err := fetchLatestVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching latest version: %v\n", err)
		os.Exit(1)
	}

	if version != "dev" && "v"+version == latest {
		fmt.Printf("Already up to date (%s).\n", latest)
		return
	}

	if version != "dev" {
		fmt.Printf("Updating v%s -> %s\n", version, latest)
	} else {
		fmt.Printf("Updating to %s\n", latest)
	}

	tmpDir, err := os.MkdirTemp("", "rbchat-update-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	archiveVersion := strings.TrimPrefix(latest, "v")
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	archiveName := fmt.Sprintf("%s_%s_%s_%s.%s", repoName, archiveVersion, runtime.GOOS, runtime.GOARCH, ext)
	archiveURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", repoOwner, repoName, latest, archiveName)

	fmt.Println("Downloading...")
	archivePath := filepath.Join(tmpDir, archiveName)
	if err := downloadFile(archivePath, archiveURL); err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}

	checksumURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/checksums.txt", repoOwner, repoName, latest)
	if err := verifyChecksum(archivePath, archiveName, checksumURL); err != nil {
		fmt.Fprintf(os.Stderr, "Checksum verification failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Installing...")
	binaryName := repoName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tmpDir, binaryName)
	if err := extractBinary(binaryPath, archivePath); err != nil {
		fmt.Fprintf(os.Stderr, "Extraction failed: %v\n", err)
		os.Exit(1)
	}

	if err := replaceBinary(binaryPath, exe); err != nil {
		fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated to %s.\n", latest)
	fmt.Println("Run 'rbchat' to start the new version.")
}

func fetchLatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("rate limited by GitHub API — try again later")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("could not determine latest version")
	}
	return release.TagName, nil
}

func downloadFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func verifyChecksum(archivePath, archiveName, checksumURL string) error {
	resp, err := http.Get(checksumURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == archiveName {
			expected = parts[0]
			break
		}
	}
	if expected == "" {
		return nil
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil
	}
	computed := fmt.Sprintf("%x", h.Sum(nil))
	if computed != expected {
		return fmt.Errorf("expected %s, got %s", expected, computed)
	}
	return nil
}

func extractBinary(dst, archive string) error {
	if strings.HasSuffix(archive, ".zip") {
		return unzipBinary(dst, archive)
	}
	return untarBinary(dst, archive)
}

func untarBinary(dst, archive string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	binaryName := repoName
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Name == binaryName {
			out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, tr)
			return err
		}
	}
	return fmt.Errorf("binary %s not found in archive", binaryName)
}

func unzipBinary(dst, archive string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()

	binaryName := repoName + ".exe"
	for _, f := range r.File {
		if f.Name == binaryName {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, f.Mode())
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, rc)
			return err
		}
	}
	return fmt.Errorf("binary %s not found in archive", binaryName)
}

func replaceBinary(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Write to a temp file in the same directory for atomic rename
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, "rbchat-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	tmp.Close()

	if err := os.Rename(tmpName, dst); err != nil {
		os.Remove(tmpName)
		return err
	}

	return nil
}
