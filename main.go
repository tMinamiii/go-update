package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
)

type GoDownloadCandidate struct {
	Version string `json:"version"`
}

type GoDownloadCandidates []GoDownloadCandidate

func fetchLatestVersion() (string, error) {
	resp, err := http.Get("https://go.dev/dl/?mode=json")
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()

	candidates := GoDownloadCandidates{}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	err = json.Unmarshal(b, &candidates)
	if err != nil {
		return "", nil
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no candidates")
	}

	return candidates[0].Version, nil
}

func copyFileTgz(base string, from *tar.Reader, header *tar.Header) error {
	tok := []string{base}
	tok = append(tok, strings.Split(header.Name, "/")[1:]...)
	fullpath := filepath.Join(tok...)

	// fmt.Println(fullpath)

	if header.FileInfo().Mode().IsDir() {
		return os.MkdirAll(fullpath, 0755)
	}

	newf, err := os.Create(fullpath)
	if err != nil {
		return err
	}
	_, err = io.Copy(newf, from)
	if err != nil {
		return err
	}
	newf.Close()
	return os.Chmod(newf.Name(), fs.FileMode(header.Mode))
}

func extractTgz(base string, resp *http.Response) error {
	r1, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	r := tar.NewReader(r1)
	for {
		cur, err := r.Next()
		if err != nil {
			return err
		}

		err = copyFileTgz(base, r, cur)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func packageName(version string) string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	ext := "tar.gz"
	pkgName := fmt.Sprintf("%s.%s-%s.%s", version, os, arch, ext)
	return pkgName
}

func IsLatest(latest, current string) (bool, error) {
	currentVersion, err := version.NewVersion(strings.Replace(current, "go", "", 1))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	latestVersion, err := version.NewVersion(strings.Replace(latest, "go", "", 1))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	return latestVersion.LessThanOrEqual(currentVersion), nil

}

func main() {
	latest, err := fetchLatestVersion()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	url := fmt.Sprintf("https://go.dev/dl/%s", packageName(latest))
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	defer resp.Body.Close()

	const base = "/usr/local/go"
	err = os.RemoveAll(base)
	if err != nil {
		log.Fatal(err)
	}

	err = extractTgz(base, resp)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	fmt.Printf("latest version %s installed\n", latest)
}
