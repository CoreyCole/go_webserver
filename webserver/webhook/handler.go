package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

const cmdTimeout = 5 * time.Minute

// Handler handles GitHub webhook events and self-rebuilds the binary.
type Handler struct {
	buildMu sync.Mutex
	secret  string
	repoDir string
}

// New creates a webhook handler. If secret is empty, webhook verification is disabled.
func New(secret string) *Handler {
	return &Handler{
		secret:  secret,
		repoDir: "/home/ubuntu/go_webserver",
	}
}

// HandleGitHub processes GitHub webhook push events.
func (h *Handler) HandleGitHub(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Error().Err(err).Msg("webhook: error reading body")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "bad request"})
	}

	// Verify HMAC-SHA256 signature.
	if h.secret != "" {
		sigHeader := c.Request().Header.Get("X-Hub-Signature-256")
		if sigHeader == "" {
			log.Warn().Msg("webhook: missing X-Hub-Signature-256 header")
			return c.JSON(
				http.StatusForbidden,
				map[string]string{"error": "missing signature"},
			)
		}
		if !strings.HasPrefix(sigHeader, "sha256=") {
			log.Warn().Str("header", sigHeader).Msg("webhook: malformed signature header")
			return c.JSON(
				http.StatusForbidden,
				map[string]string{"error": "bad signature"},
			)
		}
		sigHex := strings.TrimPrefix(sigHeader, "sha256=")
		sig, err := hex.DecodeString(sigHex)
		if err != nil {
			log.Warn().Err(err).Msg("webhook: invalid hex in signature")
			return c.JSON(
				http.StatusForbidden,
				map[string]string{"error": "bad signature"},
			)
		}

		mac := hmac.New(sha256.New, []byte(h.secret))
		mac.Write(body)
		expected := mac.Sum(nil)
		if !hmac.Equal(sig, expected) {
			log.Warn().Msg("webhook: signature mismatch")
			return c.JSON(
				http.StatusForbidden,
				map[string]string{"error": "bad signature"},
			)
		}
	}

	// Handle ping event (GitHub sends on webhook creation).
	event := c.Request().Header.Get("X-GitHub-Event")
	if event == "ping" {
		log.Info().Msg("webhook: received ping event")
		return c.JSON(http.StatusOK, map[string]string{"status": "pong"})
	}

	if event != "push" {
		log.Info().Str("event", event).Msg("webhook: ignoring event")
		return c.JSON(
			http.StatusOK,
			map[string]string{"status": "ignored", "reason": "not a push event"},
		)
	}

	// Parse push payload.
	var payload struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error().Err(err).Msg("webhook: error parsing payload")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "bad payload"})
	}

	// Only deploy for pushes to main.
	if payload.Ref != "refs/heads/main" {
		log.Info().Str("ref", payload.Ref).Msg("webhook: ignoring push")
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ignored",
			"reason": "ref is " + payload.Ref,
		})
	}

	// Trigger rebuild asynchronously.
	log.Info().Msg("webhook: push to main detected — triggering rebuild")
	go h.rebuild()

	return c.JSON(http.StatusAccepted, map[string]string{"status": "rebuilding"})
}

// rebuild pulls latest code, rebuilds the binary, and SIGTERMs for systemd restart.
func (h *Handler) rebuild() {
	if !h.buildMu.TryLock() {
		log.Info().Msg("webhook: rebuild already in progress, skipping")
		return
	}
	defer h.buildMu.Unlock()

	start := time.Now()
	log.Info().Msg("webhook: starting rebuild...")

	// 1. git fetch + reset
	if err := h.run("git", "fetch", "origin", "main"); err != nil {
		return
	}
	if err := h.run("git", "reset", "--hard", "origin/main"); err != nil {
		return
	}
	log.Info().Msg("webhook: code updated")

	// 2. sqlc generate
	if err := h.run("sqlc", "generate"); err != nil {
		return
	}
	log.Info().Msg("webhook: sqlc generated")

	// 3. templ generate
	if err := h.run("templ", "generate", "-path", "webserver/view"); err != nil {
		return
	}
	log.Info().Msg("webhook: templ generated")

	// 4. Build tailwind CSS
	if err := h.run("tailwindcss",
		"-i", "webserver/view/css/index.css", "-o", "public/build.css",
		"--content", "./webserver/**/*.go", "--content", "./webserver/**/*.templ",
	); err != nil {
		return
	}
	log.Info().Msg("webhook: tailwind built")

	// 5. Build new binary
	if err := h.run(
		"go",
		"build",
		"-o",
		"/tmp/go_webserver-next",
		"main.go",
	); err != nil {
		return
	}

	// 6. Replace running binary: unlink old, then rename new into place.
	// os.Remove unlinks the directory entry while the kernel keeps the inode
	// alive for the running process, freeing the path for the rename.
	binPath := "/home/ubuntu/go_webserver/bin/go_webserver"
	if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("webhook: remove old binary failed")
		return
	}
	if err := os.Rename("/tmp/go_webserver-next", binPath); err != nil {
		log.Error().Err(err).Msg("webhook: rename failed")
		return
	}

	log.Info().
		Dur("duration", time.Since(start)).
		Msg("webhook: build succeeded, restarting")

	// 7. SIGTERM self — systemd restarts with new binary
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(syscall.SIGTERM)
}

// run executes a command in the repo directory and returns any error.
func (h *Handler) run(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = h.repoDir
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().
			Str("cmd", fmt.Sprintf("%s %s", name, strings.Join(args, " "))).
			Str("output", string(output)).
			Err(err).
			Msg("webhook: command failed")
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
