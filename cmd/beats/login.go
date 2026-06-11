package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/palarix/beats/internal/auth"
	"github.com/palarix/beats/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var loginCmd = &cobra.Command{
	Use:   "login <server-url>",
	Short: "Authenticate against a remote beats server",
	Long: `Authenticate against a remote beats server using SSH key authentication.

The command discovers available SSH keys, signs a challenge nonce from the
server, and exchanges it for a JWT token that is cached locally.

Examples:
  beats login https://beats.example.com
  beats login http://localhost:8080`,
	Args: cobra.ExactArgs(1),
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	serverURL := strings.TrimRight(args[0], "/")

	// Discover SSH keys
	keys, err := auth.DiscoverSSHKeys()
	if err != nil {
		return fmt.Errorf("failed to discover SSH keys: %w", err)
	}
	if len(keys) == 0 {
		return fmt.Errorf("no SSH keys found in ~/.ssh/ or ssh-agent")
	}

	// Select key (prefer ed25519, fall back to first available)
	var selected *auth.SSHKey
	if len(keys) == 1 {
		selected = &keys[0]
	} else {
		selected = auth.PreferEd25519(keys)
		fmt.Printf("Using key: %s\n", selected.Label())
	}

	client := &http.Client{Timeout: 15 * time.Second}

	// 1. Request challenge nonce
	resp, err := client.Post(serverURL+"/auth/challenge", "application/json", nil)
	if err != nil {
		return fmt.Errorf("server unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("challenge failed (%d): %s", resp.StatusCode, string(body))
	}

	var challengeResp struct {
		Nonce string `json:"nonce"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&challengeResp); err != nil {
		return fmt.Errorf("invalid challenge response: %w", err)
	}

	// 2. Sign the nonce
	sig, err := selected.Signer.Sign(rand.Reader, []byte(challengeResp.Nonce))
	if err != nil {
		return fmt.Errorf("failed to sign challenge: %w", err)
	}

	// 3. Send verification
	verifyBody, _ := json.Marshal(map[string]string{
		"public_key": base64.StdEncoding.EncodeToString(selected.Signer.PublicKey().Marshal()),
		"signature":  base64.StdEncoding.EncodeToString(ssh.Marshal(sig)),
		"nonce":      challengeResp.Nonce,
	})

	resp, err = client.Post(serverURL+"/auth/verify", "application/json", bytes.NewReader(verifyBody))
	if err != nil {
		return fmt.Errorf("verification request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed (%d): %s", resp.StatusCode, string(body))
	}

	var verifyResp struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return fmt.Errorf("invalid verify response: %w", err)
	}

	// 4. Save token
	rc := config.RemoteConfig{
		URL:   serverURL,
		Token: verifyResp.Token,
	}
	if err := config.SaveRemoteConfig(rc); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	// Decode identity from JWT claims for display
	identity := "authenticated user"
	parts := strings.SplitN(verifyResp.Token, ".", 3)
	if len(parts) == 3 {
		if payload, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			var claims struct {
				Sub string `json:"sub"`
			}
			if json.Unmarshal(payload, &claims) == nil && claims.Sub != "" {
				identity = claims.Sub
			}
		}
	}

	fmt.Printf("Logged in to %s as %s\n", serverURL, identity)
	if verifyResp.ExpiresAt != "" {
		fmt.Printf("Token expires: %s\n", verifyResp.ExpiresAt)
	}
	return nil
}
