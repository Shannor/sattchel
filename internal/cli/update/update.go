package update

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/minio/selfupdate"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	repoOwner = "Shannor"
	repoName  = "test-cli"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update the CLI to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate()
		},
	}
}

func runUpdate() error {
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	fmt.Printf("Latest version: %s\n", release.TagName)

	assetName := buildAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s", assetName)
	}

	fmt.Printf("Downloading %s...\n", assetName)
	if err := doUpdate(downloadURL); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println("Successfully updated!")
	return nil
}

func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func buildAssetName() string {
	os := cases.Title(language.English).String(runtime.GOOS)
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}

	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("%s_%s_%s.%s", repoName, os, arch, ext)
}

func doUpdate(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	var reader io.Reader

	// tar.gz archives need to be extracted to get the binary
	if strings.HasSuffix(url, ".tar.gz") {
		r, err := extractBinaryFromTarGz(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to extract binary: %w", err)
		}
		reader = r
	} else {
		reader = resp.Body
	}

	if err := selfupdate.Apply(reader, selfupdate.Options{}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return fmt.Errorf("update failed and rollback also failed: %v (original: %w)", rerr, err)
		}
		return err
	}
	return nil
}

func extractBinaryFromTarGz(r io.Reader) (io.Reader, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// The binary name matches the repo name (or repo name + .exe on Windows)
		name := header.Name
		if name == repoName || name == repoName+".exe" {
			return tr, nil
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive", repoName)
}
