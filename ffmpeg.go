package ffmpeghelper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

func getUserBinDir() string {
	var dir string
	if home, err := os.UserHomeDir(); err == nil {
		var d string
		switch runtime.GOOS {
		case "windows":
			// windows
			d = filepath.Join(home, "AppData", "Local", "Programs")
		default:
			// unix-like and others
			d = filepath.Join(home, ".local", "bin")
		}
		dir, _ = filepath.Abs(d)
	} else {
		// fallback to the executable's dir on error
		exe, _ := os.Executable()
		dir, _ = filepath.Abs(exe)
	}

	return dir
}

func isValidFfmpegExe(path string) bool {
	// check if file exists and not a dir
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return false
	}
	// check if file executes
	cmd := exec.Command(path, "-version")
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}

func getFfmpegVariant() string {
	o := runtime.GOOS
	var a string
	switch runtime.GOARCH {
	case "amd64":
		a = "x86_64"
	case "386":
		a = "i686"
	case "arm":
		a = "armhf"
	case "arm64":
		a = "aarch64"
	case "loong64":
		a = "loongarch64"
	default:
		a = runtime.GOARCH
	}
	return o + "_" + a
}

func getFfmpegName(variant string) string {
	name := "ffmpeg"
	if variant != "" {
		name += "_" + variant
	}
	switch runtime.GOOS {
	case "windows":
		name += ".exe"
	}
	return name
}

// Get path of FFmpeg.
//
// Returns:
//
//	string: path of the executable
func GetFfmpegPath() string {
	name := getFfmpegName("")
	// find in user bin dir
	if path := filepath.Join(getUserBinDir(), name); isValidFfmpegExe(path) {
		return path
	}
	// find in os path
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return ""
}

const userAgent = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36"

var (
	httpClient             = &http.Client{}
	errVariantIncompatible = errors.New("compatible variant not found")
	errDownloadFailed      = errors.New("binary fetching failed")
	errFileCorrupted       = errors.New("binary sha256 mismatch")
)

type assetType struct {
	Name               string `json:"name"`
	Digest             string `json:"digest"`
	BrowserDownloadUrl string `json:"browser_download_url"`
}

func fetchRelease() (*assetType, error) {
	// get the latest release
	const url = "https://api.github.com/repos/StellarForager/FFmpeg/releases/latest"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		return nil, errors.New(string(body))
	}
	var data1 struct {
		Assets []assetType `json:"assets"` // emptiable
	}
	if err := json.NewDecoder(res.Body).Decode(&data1); err != nil {
		return nil, err
	}
	// return a matching variant's info
	vName := getFfmpegName(getFfmpegVariant())
	for _, a := range data1.Assets {
		if a.Name == vName {
			return &a, nil
		}
	}
	return nil, nil
}

func downloadFile(url, path string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return errDownloadFailed
	}
	// save to path without variant in name
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, res.Body)
	return err
}

func verifySha256(path, digest string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	sh := sha256.New()
	if _, err := io.Copy(sh, file); err != nil {
		return false, err
	}
	return strings.EqualFold(hex.EncodeToString(sh.Sum(nil)), digest), nil
}

func chmodExec(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	// u+x, g+x, o+x
	return os.Chmod(path, info.Mode()|0111)
}

var fetchFfmpegLock sync.Mutex

// Download FFmpeg to the user's bin directory.
//
// Returns:
//
//	string: path on success
//	error: error
func FetchFfmpeg() (string, error) {
	// get a matching variant from the latest release
	asset, err := fetchRelease()
	if err != nil {
		return "", err
	} else if asset == nil {
		return "", errVariantIncompatible
	}
	// create dir
	dir := getUserBinDir()
	if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
		return "", err
	}
	// download the binary
	fetchFfmpegLock.Lock()
	defer fetchFfmpegLock.Unlock()
	path := filepath.Join(dir, getFfmpegName(""))
	isDownloadFailed := true
	var dlErr error
	// try proxies first
	for _, proxy := range []string{
		"https://ghfast.top/",
		"https://gh-proxy.com/",
		"",
	} {
		if err := downloadFile(
			proxy+asset.BrowserDownloadUrl, path); err == nil {
			isDownloadFailed = false
			break
		} else {
			dlErr = err
			os.Remove(path)
		}
	}
	if isDownloadFailed {
		return "", dlErr
	}
	// verify hash, strip leading "sha256:" from the digest
	if eq, err := verifySha256(path, asset.Digest[7:]); err != nil {
		return "", err
	} else if !eq {
		os.Remove(path)
		return "", errFileCorrupted
	}
	if err := chmodExec(path); err != nil {
		return "", err
	}
	return path, nil
}

var (
	ffmpegPath        string
	errFfmpegNotFound = errors.New("cannot find executable ffmpeg")
)

// Get FFmpeg's path or download it if not yet.
//
// Returns:
//
//	string: path on success
//	error: error
func Ffmpeg() (string, error) {
	// return if cached
	if ffmpegPath != "" {
		return ffmpegPath, nil
	}
	// try if ffmpeg exists
	if path := GetFfmpegPath(); path != "" {
		ffmpegPath = path
		return path, nil
	}
	// download ffmpeg
	fmt.Println("downloading FFmpeg...")
	if _, err := FetchFfmpeg(); err != nil {
		fmt.Println("download failed")
		return "", err
	}
	// re-get the path to ensure the downloaded ffmpeg is ok
	if path := GetFfmpegPath(); path != "" {
		ffmpegPath = path
		return path, nil
	}
	return "", errFfmpegNotFound
}
