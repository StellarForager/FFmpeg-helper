package ffmpeghelper

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
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
		switch o {
		case "android":
			a = "armv7a"
		default:
			a = "armhf"
		}
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

func getExecDir() string {
	if ex, err := os.Executable(); err == nil {
		return filepath.Dir(ex)
	}
	return "."
}

// Get path of FFmpeg.
//
// Returns:
//
//	string: path of the executable
func GetFfmpegPath() string {
	names := []string{getFfmpegName("")}
	if runtime.GOOS == "android" {
		names = append(names, "libffmpeg.so")
	}
	for _, name := range names {
		if path := filepath.Join(getExecDir(), name); isValidFfmpegExe(path) {
			// find in the same dir
			return path
		} else if path := filepath.Join(
			getUserBinDir(), name); isValidFfmpegExe(path) {
			// find in user bin dir
			return path
		} else if path, err := exec.LookPath(
			name); err == nil && isValidFfmpegExe(path) {
			// find in os path
			return path
		}
	}
	return ""
}

const userAgent = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36"

var (
	httpClient        = &http.Client{Timeout: time.Minute * 15}
	errDownloadFailed = errors.New("binary fetching failed")
	errFileCorrupted  = errors.New("binary sha256 mismatch")
)

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
	if _, err := io.Copy(file, res.Body); err != nil {
		return err
	}
	// verify hash
	if v, ok := res.Header["X-Ms-Blob-Content-Md5"]; ok {
		if sum, err := base64.StdEncoding.DecodeString(v[0]); err == nil {
			if eq, err := verifyMd5(path, sum); eq {
				return nil
			} else {
				return err
			}
		} else {
			return err
		}
	}
	return errFileCorrupted
}

func verifyMd5(path string, sum []byte) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false, err
	}
	fsum := hasher.Sum(nil)
	return bytes.Equal(sum, fsum), nil
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
	url :=
		"https://github.com/StellarForager/FFmpeg/releases/latest/download/" +
			getFfmpegName(getFfmpegVariant())
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
			proxy+url, path); err == nil {
			isDownloadFailed = false
			break
		} else {
			dlErr = err
		}
	}
	if isDownloadFailed {
		os.Remove(path)
		return "", dlErr
	}
	// chmod +x
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
	os.Stdout.WriteString("FFmpeg downloading...\n")
	if _, err := FetchFfmpeg(); err != nil {
		os.Stderr.WriteString("FFmpeg download faild\n")
		return "", err
	}
	// re-get the path to ensure the downloaded ffmpeg is ok
	if path := GetFfmpegPath(); path != "" {
		ffmpegPath = path
		return path, nil
	}
	return "", errFfmpegNotFound
}
