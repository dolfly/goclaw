package channels

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smallnest/goclaw/internal/logger"
	"go.uber.org/zap"
)

// Default Weixin API endpoints
const (
	DefaultWeixinBaseURL = "https://ilinkai.weixin.qq.com"
	DefaultWeixinCDNURL  = "https://novac2c.cdn.weixin.qq.com/c2c"
)

// WeixinAPIClient is the HTTP client for Weixin API
type WeixinAPIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewWeixinAPIClient creates a new Weixin API client
func NewWeixinAPIClient(baseURL, token string) *WeixinAPIClient {
	if baseURL == "" {
		baseURL = DefaultWeixinBaseURL
	}
	return &WeixinAPIClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// SetToken sets the authentication token
func (c *WeixinAPIClient) SetToken(token string) {
	c.token = token
}

// GetToken returns the current token
func (c *WeixinAPIClient) GetToken() string {
	return c.token
}

// generateWechatUIN generates a random X-WECHAT-UIN header value
func generateWechatUIN() string {
	b := make([]byte, 4)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// doRequest performs an HTTP request to the Weixin API
func (c *WeixinAPIClient) doRequest(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	var bodyReader io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	url := c.baseURL + "/" + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set authentication headers if token is available
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("AuthorizationType", "ilink_bot_token")
		req.Header.Set("X-WECHAT-UIN", generateWechatUIN())
	}

	logger.Debug("Weixin API request",
		zap.String("method", method),
		zap.String("url", url),
		zap.String("path", path))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	logger.Debug("Weixin API response",
		zap.Int("status_code", resp.StatusCode),
		zap.Int("response_size", len(respData)))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respData))
	}

	if respBody != nil {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// GetUpdates fetches new messages using long polling
func (c *WeixinAPIClient) GetUpdates(ctx context.Context, req *GetUpdatesReq) (*GetUpdatesResp, error) {
	resp := &GetUpdatesResp{}
	if err := c.doRequest(ctx, http.MethodPost, "ilink/bot/getupdates", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SendMessage sends a message to a user
func (c *WeixinAPIClient) SendMessage(ctx context.Context, req *SendMessageReq) (*SendMessageResp, error) {
	resp := &SendMessageResp{}
	if err := c.doRequest(ctx, http.MethodPost, "ilink/bot/sendmessage", req, resp); err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("send message failed: %s (code: %d)", resp.ErrMsg, resp.ErrCode)
	}
	return resp, nil
}

// GetUploadURL gets a pre-signed URL for CDN upload
func (c *WeixinAPIClient) GetUploadURL(ctx context.Context, req *GetUploadUrlReq) (*GetUploadUrlResp, error) {
	resp := &GetUploadUrlResp{}
	if err := c.doRequest(ctx, http.MethodPost, "ilink/bot/getuploadurl", req, resp); err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get upload URL failed: %s (code: %d)", resp.ErrMsg, resp.ErrCode)
	}
	return resp, nil
}

// GetConfig gets account configuration including typing ticket
func (c *WeixinAPIClient) GetConfig(ctx context.Context) (*GetConfigResp, error) {
	resp := &GetConfigResp{}
	if err := c.doRequest(ctx, http.MethodPost, "ilink/bot/getconfig", nil, resp); err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get config failed: %s (code: %d)", resp.ErrMsg, resp.ErrCode)
	}
	return resp, nil
}

// SendTyping sends typing indicator
func (c *WeixinAPIClient) SendTyping(ctx context.Context, req *SendTypingReq) error {
	resp := &SendTypingResp{}
	if err := c.doRequest(ctx, http.MethodPost, "ilink/bot/sendtyping", req, resp); err != nil {
		return err
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("send typing failed: %s (code: %d)", resp.ErrMsg, resp.ErrCode)
	}
	return nil
}

// GetBotQRCode gets the QR code for bot login
func (c *WeixinAPIClient) GetBotQRCode(ctx context.Context) (*GetBotQRCodeResp, error) {
	resp := &GetBotQRCodeResp{}
	if err := c.doRequest(ctx, http.MethodPost, "ilink/bot/get_bot_qrcode", nil, resp); err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get bot QR code failed: %s (code: %d)", resp.ErrMsg, resp.ErrCode)
	}
	return resp, nil
}

// GetQRCodeStatus checks the QR code scan status
func (c *WeixinAPIClient) GetQRCodeStatus(ctx context.Context, sessionKey string) (*GetQRCodeStatusResp, error) {
	req := &GetQRCodeStatusReq{SessionKey: sessionKey}
	resp := &GetQRCodeStatusResp{}
	if err := c.doRequest(ctx, http.MethodPost, "ilink/bot/get_qrcode_status", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// UploadToCDN uploads encrypted data to CDN
func (c *WeixinAPIClient) UploadToCDN(ctx context.Context, uploadURL string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload to CDN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CDN upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DownloadFromCDN downloads encrypted data from CDN
func (c *WeixinAPIClient) DownloadFromCDN(ctx context.Context, cdnMedia *CDNMedia) ([]byte, error) {
	if cdnMedia.DownloadURL == "" && cdnMedia.DownloadParam == "" {
		return nil, fmt.Errorf("no download URL available")
	}

	downloadURL := cdnMedia.DownloadURL
	if cdnMedia.DownloadParam != "" {
		// Use download param with CDN base URL
		downloadURL = DefaultWeixinCDNURL + "/" + cdnMedia.DownloadParam
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download from CDN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("CDN download failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CDN response: %w", err)
	}

	return data, nil
}
