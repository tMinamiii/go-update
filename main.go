package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
)

const DefaultBase = "/usr/local/go"
const Repository = "github.com/tMinamiii/go-update"

type GoDownloadCandidate struct {
	Version string `json:"version"`
}

type GoDownloadCandidates []GoDownloadCandidate

func rebuildGoUpdate() error {
	gobin := DefaultBase + "/bin/go"
	repos := Repository + "@latest"
	_, err := exec.Command(gobin, "install", repos).Output()
	if err != nil {
		return err
	}
	return nil
}

func fetchLatestVersion() (*string, error) {
	resp, err := http.Get("https://go.dev/dl/?mode=json")
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()

	candidates := GoDownloadCandidates{}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}

	err = json.Unmarshal(b, &candidates)
	if err != nil {
		return nil, nil
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates")
	}

	return &candidates[0].Version, nil
}

func copyFileTgz(base string, from *tar.Reader, header *tar.Header) error {
	tok := []string{base}
	tok = append(tok, strings.Split(header.Name, "/")[1:]...)
	fullpath := filepath.Join(tok...)

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

func checkVersion(target, current string) {
	currentVersion, err := version.NewVersion(strings.Replace(current, "go", "", 1))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	targetVersion, err := version.NewVersion(strings.Replace(target, "go", "", 1))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if targetVersion.LessThanOrEqual(currentVersion) {
		fmt.Printf("Target version already installed. -- %s\n", current)
		os.Exit(0)
	}
}

func install(target, current string) error {
	fmt.Printf("Your version is %s and latest version is %s\n", current, target)
	fmt.Printf("Start the Installation %s\n", target)

	url := fmt.Sprintf("https://go.dev/dl/%s", packageName(target))
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	defer resp.Body.Close()

	err = os.RemoveAll(DefaultBase)
	if err != nil {
		log.Fatal(err)
	}

	err = extractTgz(DefaultBase, resp)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	return nil
}

func main() {
	target := flag.String("v", "", "version")

	flag.Parse()
	current := runtime.Version()
	if runtime.GOOS == "windows" {
		fmt.Println("windows is incompatible")
		os.Exit(0)
	}

	if *target == "" {
		var err error
		target, err = fetchLatestVersion()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

	}
	checkVersion(*target, runtime.Version())

	err := install(*target, current)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	rebuildGoUpdate()

	fmt.Printf("latest version %s installed\n", *target)
}
