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

func (g *GoDownloadCandidates) versions() GoVersions {
	versions := make(GoVersions, 0, len(*g))
	for _, v := range *g {
		versions = append(versions, v.Version)
	}
	return versions
}

type GoVersions []string

func (g *GoVersions) isAvailable(version string) bool {
	for _, v := range *g {
		if v == version {
			return true
		}
	}
	return false
}

func (g *GoVersions) printAvailable() {
	fmt.Println("Available versions:")
	for _, v := range *g {
		fmt.Println(v)
	}
}

func rebuildGoUpdate() error {
	gobin := DefaultBase + "/bin/go"
	repos := Repository + "@latest"
	_, err := exec.Command(gobin, "install", repos).Output()
	if err != nil {
		return err
	}
	return nil
}

func getCurrentVersion() (*string, error) {
	gobin := DefaultBase + "/bin/go"
	goVersionBytes, err := exec.Command(gobin, "version").Output()
	if err != nil {
		return nil, err
	}
	goVersionStr := string(goVersionBytes)
	goVersion := strings.Split(goVersionStr, " ")
	if len(goVersion) < 2 {
		return nil, fmt.Errorf("failed to exec go version. result = %+v", goVersion)
	}
	return &goVersion[2], nil
}

func fetchAvailableVersion() (GoVersions, error) {
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

	return candidates.versions(), nil
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
	fmt.Printf("Your version is %s and target version is %s\n", current, target)
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
	current, err := getCurrentVersion()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if runtime.GOOS == "windows" {
		fmt.Println("Windows is incompatible.")
		os.Exit(0)
	}

	versions, err := fetchAvailableVersion()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if *target == "" {
		target = &versions[0]
		checkVersion(*target, *current)
	} else if !versions.isAvailable(*target) {
		versions.printAvailable()
		os.Exit(0)
	}

	err = install(*target, *current)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	err = rebuildGoUpdate()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Printf("Latest version %s installed.\n", *target)
}
