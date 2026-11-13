package upstream

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// ValidatorFunc runs offline dry-run pre-flight validation on the staged file before atomic rename.
type ValidatorFunc func(ctx context.Context, stageFilePath string) error

// PostReloadFunc triggers core hot-reload/switch after rename; returning error causes immediate 0ms rollback to .bak.
type PostReloadFunc func(ctx context.Context) error

// copyBufPool pools 32KB buffers to eliminate memory allocation spikes during large file streaming.
var copyBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

// AtomicInstaller manages streaming download, SHA256 verification, POSIX atomic rename, and failure rollback.
type AtomicInstaller struct {
	httpClient   *http.Client
	maxAssetSize int64
}

// NewAtomicInstaller initializes an AtomicInstaller with production timeout limits.
func NewAtomicInstaller(timeout time.Duration) *AtomicInstaller {
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	tr := &http.Transport{
		MaxIdleConns:        4,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     15 * time.Second,
	}
	return &AtomicInstaller{
		httpClient:   &http.Client{Transport: tr, Timeout: timeout},
		maxAssetSize: 128 * 1024 * 1024, // 128MB ceiling prevents container disk exhaustion
	}
}

// InstallFile executes: [Same-Directory Staging -> SHA256 Stream -> Validator -> .bak Backup -> Atomic Rename -> PostReload / Rollback]
func (a *AtomicInstaller) InstallFile(
	ctx context.Context,
	downloadURL string,
	targetPath string,
	expectedSHA256 string,
	fileMode os.FileMode,
	validator ValidatorFunc,
	postReload PostReloadFunc,
) error {
	defer func() {
		runtime.GC()
		debug.FreeOSMemory()
	}()

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target dir failed: %w", err)
	}

	// Step 1: Stage file in the exact same directory (same filesystem inode, avoids EXDEV on rename)
	stagePath := filepath.Join(targetDir, fmt.Sprintf(".cubex-stage-%s.%d.tmp", filepath.Base(targetPath), time.Now().UnixNano()))
	defer func() {
		_ = os.Remove(stagePath)
	}()

	// Step 2: Stream download with on-the-fly SHA256 calculation
	actualHash, err := a.streamDownloadToStage(ctx, downloadURL, stagePath, fileMode)
	if err != nil {
		return fmt.Errorf("stream download failed: %w", err)
	}

	if expectedSHA256 != "" && !strings.EqualFold(actualHash, expectedSHA256) {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", expectedSHA256, actualHash)
	}

	// Step 3: Offline pre-flight validation
	if validator != nil {
		if err := validator(ctx, stagePath); err != nil {
			return fmt.Errorf("pre-install validation failed: %w", err)
		}
	}

	// Step 4: Backup existing file to .bak and perform atomic rename (avoids ETXTBSY on running binaries)
	bakPath := targetPath + ".bak"
	hasBackup := false
	if _, err := os.Stat(targetPath); err == nil {
		_ = os.Remove(bakPath)
		if err := os.Rename(targetPath, bakPath); err != nil {
			return fmt.Errorf("backup existing file to .bak failed: %w", err)
		}
		hasBackup = true
	}

	if err := os.Rename(stagePath, targetPath); err != nil {
		if hasBackup {
			_ = os.Rename(bakPath, targetPath)
		}
		return fmt.Errorf("atomic rename stage to target failed: %w", err)
	}

	// Step 5: Post-reload hook; if hot reload fails, trigger immediate rollback
	if postReload != nil {
		if err := postReload(ctx); err != nil {
			if hasBackup {
				_ = os.Rename(bakPath, targetPath)
				_ = postReload(ctx) // restore previous state
			}
			return fmt.Errorf("post-reload failed, rolled back automatically: %w", err)
		}
	}

	if hasBackup {
		_ = os.Remove(bakPath)
	}
	return nil
}

func (a *AtomicInstaller) streamDownloadToStage(ctx context.Context, url, stagePath string, mode os.FileMode) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected http status %d", resp.StatusCode)
	}

	f, err := os.OpenFile(stagePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	limitedReader := io.LimitReader(resp.Body, a.maxAssetSize)
	teeReader := io.TeeReader(limitedReader, hasher)

	bufPtr := copyBufPool.Get().(*[]byte)
	defer copyBufPool.Put(bufPtr)

	if _, err := io.CopyBuffer(f, teeReader, *bufPtr); err != nil {
		_ = f.Close()
		return "", err
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// VerifyELFBinary checks if the file begins with Linux ELF magic bytes.
func VerifyELFBinary(_ context.Context, stageFilePath string) error {
	f, err := os.Open(stageFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return fmt.Errorf("read elf header failed: %w", err)
	}
	if !bytes.Equal(magic, []byte{0x7f, 'E', 'L', 'F'}) {
		return errors.New("invalid binary format: missing ELF magic header")
	}
	return nil
}
