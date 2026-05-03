package config

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"test-cli/internal/printer"

	"github.com/minio/selfupdate"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	repoOwner = "Shannor"
	repoName  = "test-cli"
)

// Version is set at build time via ldflags (e.g. -ldflags "-X test-cli/internal/config.Version=v1.0.0").
var Version = "dev"

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type UpdateInformation struct {
	CurrentVersion string
	NewVersion     string
	NeedToUpdate   bool
}
type Updater interface {
	CheckForUpdate() <-chan UpdateInformation
	RunUpdate() error
}
type updater struct {
	Writer printer.Writer
}

func NewUpdater(writer printer.Writer) Updater {
	return &updater{
		Writer: writer,
	}
}

// CheckForUpdate checks GitHub for a newer release in the background.
// Returns a channel that will receive a message string if an update is available,
// or be closed with no value if the current version is up-to-date or the check fails.
func (u *updater) CheckForUpdate() <-chan UpdateInformation {
	ch := make(chan UpdateInformation, 1)
	go func() {
		defer close(ch)
		update := UpdateInformation{
			NeedToUpdate:   false,
			CurrentVersion: Version,
		}
		if Version == "dev" {
			update.NewVersion = Version
			ch <- update
			return
		}
		release, err := fetchLatestRelease()
		if err != nil {
			ch <- update
			return
		}
		update.NewVersion = release.TagName
		update.NeedToUpdate = needsUpdate(release)
		ch <- update
	}()
	return ch
}

func needsUpdate(release *githubRelease) bool {
	currentVersion := Version
	if strings.HasPrefix(currentVersion, "v") {
		currentVersion = currentVersion[1:]
	}
	latestVersion := release.TagName
	if strings.HasPrefix(latestVersion, "v") {
		latestVersion = latestVersion[1:]
	}
	return latestVersion != currentVersion
}

// RunUpdate fetches the latest release and applies the update.
func (u *updater) RunUpdate() error {
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	if !needsUpdate(release) {
		u.Writer.Info("Already up to date!")
		return nil
	}

	fmt.Printf("Current Version: v%s, Latest version: %s\n", Version, release.TagName)

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

		name := header.Name
		if name == repoName || name == repoName+".exe" {
			return tr, nil
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive", repoName)
}
