package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/smallnest/goclaw/internal/logger"
	"go.uber.org/zap"
)

// TokenStore manages token persistence
type TokenStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewTokenStore creates a new token store
func NewTokenStore() (*TokenStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".goclaw", "weixin", "accounts")
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create token directory: %w", err)
	}

	return &TokenStore{baseDir: baseDir}, nil
}

// Save saves token info for an account
func (s *TokenStore) Save(accountID string, info *TokenInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.tokenPath(accountID)
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token info: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	logger.Info("Token saved", zap.String("account_id", accountID))
	return nil
}

// Load loads token info for an account
func (s *TokenStore) Load(accountID string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.tokenPath(accountID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No token file exists
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var info TokenInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token info: %w", err)
	}

	return &info, nil
}

// Delete removes token info for an account
func (s *TokenStore) Delete(accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.tokenPath(accountID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	return nil
}

// tokenPath returns the file path for an account's token
func (s *TokenStore) tokenPath(accountID string) string {
	return filepath.Join(s.baseDir, accountID+".json")
}

// WeixinAuth handles Weixin authentication
type WeixinAuth struct {
	apiClient  *WeixinAPIClient
	tokenStore *TokenStore
}

// NewWeixinAuth creates a new auth handler
func NewWeixinAuth(apiClient *WeixinAPIClient) (*WeixinAuth, error) {
	tokenStore, err := NewTokenStore()
	if err != nil {
		return nil, err
	}

	return &WeixinAuth{
		apiClient:  apiClient,
		tokenStore: tokenStore,
	}, nil
}

// LoadToken loads the stored token for an account
func (a *WeixinAuth) LoadToken(accountID string) (*TokenInfo, error) {
	return a.tokenStore.Load(accountID)
}

// SaveToken saves the token for an account
func (a *WeixinAuth) SaveToken(accountID string, info *TokenInfo) error {
	return a.tokenStore.Save(accountID, info)
}

// DeleteToken removes the stored token for an account
func (a *WeixinAuth) DeleteToken(accountID string) error {
	return a.tokenStore.Delete(accountID)
}

// StartQRLogin initiates QR code login flow
func (a *WeixinAuth) StartQRLogin(ctx context.Context) (*GetBotQRCodeResp, error) {
	resp, err := a.apiClient.GetBotQRCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get QR code: %w", err)
	}

	logger.Info("QR code generated",
		zap.String("qrcode_url", resp.QRCodeURL),
		zap.String("session_key", resp.SessionKey))

	return resp, nil
}

// WaitForLogin waits for the user to scan the QR code
func (a *WeixinAuth) WaitForLogin(ctx context.Context, sessionKey string, onStatus func(status int)) (*TokenInfo, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := a.apiClient.GetQRCodeStatus(ctx, sessionKey)
			if err != nil {
				logger.Warn("Failed to check QR code status", zap.Error(err))
				continue
			}

			if onStatus != nil {
				onStatus(status.Status)
			}

			switch status.Status {
			case QRCodeStatusWaiting:
				// Continue waiting
				logger.Debug("Waiting for QR code scan")
			case QRCodeStatusScanned:
				logger.Info("QR code scanned, waiting for confirmation")
			case QRCodeStatusConfirmed:
				logger.Info("Login confirmed",
					zap.String("user_id", status.UserID),
					zap.String("nick_name", status.NickName))

				return &TokenInfo{
					Token:     status.Token,
					UserID:    status.UserID,
					NickName:  status.NickName,
					ExpiresAt: time.Now().Add(time.Duration(status.ExpiresIn) * time.Second).Unix(),
				}, nil
			case QRCodeStatusExpired:
				return nil, fmt.Errorf("QR code expired")
			}
		}
	}
}

// LoginWithQRCode performs the complete QR code login flow
func (a *WeixinAuth) LoginWithQRCode(ctx context.Context, accountID string, displayQR func(url string) error) (*TokenInfo, error) {
	// Get QR code
	qrResp, err := a.StartQRLogin(ctx)
	if err != nil {
		return nil, err
	}

	// Display QR code
	if displayQR != nil {
		if err := displayQR(qrResp.QRCodeURL); err != nil {
			return nil, fmt.Errorf("failed to display QR code: %w", err)
		}
	}

	// Wait for login
	tokenInfo, err := a.WaitForLogin(ctx, qrResp.SessionKey, nil)
	if err != nil {
		return nil, err
	}

	// Save token
	if err := a.SaveToken(accountID, tokenInfo); err != nil {
		logger.Warn("Failed to save token", zap.Error(err))
	}

	return tokenInfo, nil
}

// IsTokenValid checks if a token is still valid
func (a *WeixinAuth) IsTokenValid(info *TokenInfo) bool {
	if info == nil || info.Token == "" {
		return false
	}

	// Check expiration with 5 minute buffer
	if info.ExpiresAt > 0 && time.Now().Unix() > info.ExpiresAt-300 {
		return false
	}

	return true
}

// Logout logs out and removes the stored token
func (a *WeixinAuth) Logout(accountID string) error {
	return a.DeleteToken(accountID)
}
