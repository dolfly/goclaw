package channels

import (
	"bytes"
	"context"
	"crypto/aes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// WeixinMediaHandler handles media file operations
type WeixinMediaHandler struct {
	apiClient  *WeixinAPIClient
	cdnBaseURL string
	httpClient *http.Client
}

// NewWeixinMediaHandler creates a new media handler
func NewWeixinMediaHandler(apiClient *WeixinAPIClient, cdnBaseURL string) *WeixinMediaHandler {
	if cdnBaseURL == "" {
		cdnBaseURL = DefaultWeixinCDNURL
	}
	return &WeixinMediaHandler{
		apiClient:  apiClient,
		cdnBaseURL: cdnBaseURL,
		httpClient: &http.Client{},
	}
}

// PKCS7Pad applies PKCS7 padding to data
func PKCS7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// PKCS7Unpad removes PKCS7 padding from data
func PKCS7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padding := int(data[length-1])
	if padding > length {
		return nil, fmt.Errorf("invalid padding")
	}
	return data[:length-padding], nil
}

// AESEncrypt encrypts data using AES-128-ECB with PKCS7 padding
func AESEncrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) > 16 {
		key = key[:16]
	} else if len(key) < 16 {
		paddedKey := make([]byte, 16)
		copy(paddedKey, key)
		key = paddedKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Apply PKCS7 padding
	paddedData := PKCS7Pad(plaintext, aes.BlockSize)

	// Encrypt in ECB mode
	ciphertext := make([]byte, len(paddedData))
	for i := 0; i < len(paddedData); i += aes.BlockSize {
		block.Encrypt(ciphertext[i:i+aes.BlockSize], paddedData[i:i+aes.BlockSize])
	}

	return ciphertext, nil
}

// AESDecrypt decrypts data using AES-128-ECB and removes PKCS7 padding
func AESDecrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) > 16 {
		key = key[:16]
	} else if len(key) < 16 {
		paddedKey := make([]byte, 16)
		copy(paddedKey, key)
		key = paddedKey
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Decrypt in ECB mode
	plaintext := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += aes.BlockSize {
		block.Decrypt(plaintext[i:i+aes.BlockSize], ciphertext[i:i+aes.BlockSize])
	}

	// Remove PKCS7 padding
	return PKCS7Unpad(plaintext)
}

// UploadFile uploads a file to Weixin CDN
func (h *WeixinMediaHandler) UploadFile(ctx context.Context, filePath, toUserID string) (*CDNMedia, error) {
	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Get file info
	fileName := filepath.Base(filePath)
	fileExt := strings.TrimPrefix(filepath.Ext(filePath), ".")
	if fileExt == "" {
		fileExt = "bin"
	}

	// Determine MIME type
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Get upload URL
	uploadReq := &GetUploadUrlReq{
		ToUserID:      toUserID,
		FileName:      fileName,
		FileExtension: fileExt,
		FileSize:      int64(len(fileData)),
		MimeType:      mimeType,
	}

	uploadResp, err := h.apiClient.GetUploadURL(ctx, uploadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload URL: %w", err)
	}

	// Decode AES key
	aesKey, err := base64.StdEncoding.DecodeString(uploadResp.AESKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	// Encrypt file data
	encryptedData, err := AESEncrypt(fileData, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt file: %w", err)
	}

	// Upload to CDN
	if err := h.apiClient.UploadToCDN(ctx, uploadResp.UploadURL, encryptedData); err != nil {
		return nil, fmt.Errorf("failed to upload to CDN: %w", err)
	}

	return &CDNMedia{
		CDNMediaID:    uploadResp.CDNMediaID,
		AESKey:        uploadResp.AESKey,
		EncryptedSize: int64(len(encryptedData)),
		OriginalSize:  int64(len(fileData)),
		FileName:      fileName,
		FileExtension: fileExt,
		MimeType:      mimeType,
	}, nil
}

// UploadData uploads data to Weixin CDN
func (h *WeixinMediaHandler) UploadData(ctx context.Context, data []byte, fileName, toUserID string, itemType int) (*CDNMedia, error) {
	fileExt := filepath.Ext(fileName)
	if strings.HasPrefix(fileExt, ".") {
		fileExt = fileExt[1:]
	}

	mimeType := mime.TypeByExtension(filepath.Ext(fileName))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	uploadReq := &GetUploadUrlReq{
		ToUserID:      toUserID,
		FileName:      fileName,
		FileExtension: fileExt,
		FileSize:      int64(len(data)),
		MimeType:      mimeType,
	}

	uploadResp, err := h.apiClient.GetUploadURL(ctx, uploadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload URL: %w", err)
	}

	// Decode AES key
	aesKey, err := base64.StdEncoding.DecodeString(uploadResp.AESKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	// Encrypt data
	encryptedData, err := AESEncrypt(data, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Upload to CDN
	if err := h.apiClient.UploadToCDN(ctx, uploadResp.UploadURL, encryptedData); err != nil {
		return nil, fmt.Errorf("failed to upload to CDN: %w", err)
	}

	return &CDNMedia{
		CDNMediaID:    uploadResp.CDNMediaID,
		AESKey:        uploadResp.AESKey,
		EncryptedSize: int64(len(encryptedData)),
		OriginalSize:  int64(len(data)),
		FileName:      fileName,
		FileExtension: fileExt,
		MimeType:      mimeType,
	}, nil
}

// DownloadFile downloads and decrypts a file from CDN
func (h *WeixinMediaHandler) DownloadFile(ctx context.Context, cdnMedia *CDNMedia) ([]byte, error) {
	// Download encrypted data
	encryptedData, err := h.apiClient.DownloadFromCDN(ctx, cdnMedia)
	if err != nil {
		return nil, fmt.Errorf("failed to download from CDN: %w", err)
	}

	// Decode AES key
	aesKey, err := base64.StdEncoding.DecodeString(cdnMedia.AESKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	// Decrypt data
	decryptedData, err := AESDecrypt(encryptedData, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return decryptedData, nil
}

// DownloadToBase64 downloads a file and returns as base64 string
func (h *WeixinMediaHandler) DownloadToBase64(ctx context.Context, cdnMedia *CDNMedia) (string, error) {
	data, err := h.DownloadFile(ctx, cdnMedia)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// MultipartUpload handles multipart upload for large files
func (h *WeixinMediaHandler) MultipartUpload(ctx context.Context, uploadURL string, data []byte, contentType string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "file")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
