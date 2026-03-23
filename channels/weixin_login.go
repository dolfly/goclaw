package channels

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/disintegration/imaging"
	"github.com/mdp/qrterminal"
	"github.com/smallnest/goclaw/internal/logger"
	"go.uber.org/zap"
)

// LoginResult represents the result of a login attempt
type LoginResult struct {
	Success  bool
	UserID   string
	NickName string
	Message  string
}

// WeixinLogin handles CLI-based Weixin login
type WeixinLogin struct {
	auth      *WeixinAuth
	accountID string
}

// NewWeixinLogin creates a new login handler
func NewWeixinLogin(accountID string) (*WeixinLogin, error) {
	auth, err := NewWeixinAuth(NewWeixinAPIClient("", ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create auth handler: %w", err)
	}

	return &WeixinLogin{
		auth:      auth,
		accountID: accountID,
	}, nil
}

// Login performs the login flow
func (l *WeixinLogin) Login(ctx context.Context) (*LoginResult, error) {
	return l.LoginWithDisplay(ctx, l.displayQRCode)
}

// LoginWithDisplay performs the login flow with a custom QR code display function
func (l *WeixinLogin) LoginWithDisplay(ctx context.Context, displayQR func(url string) error) (*LoginResult, error) {
	// Start QR code login
	qrResp, err := l.auth.StartQRLogin(ctx)
	if err != nil {
		return &LoginResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get QR code: %v", err),
		}, err
	}

	// Display QR code
	if displayQR != nil {
		if err := displayQR(qrResp.QRCodeURL); err != nil {
			return &LoginResult{
				Success: false,
				Message: fmt.Sprintf("Failed to display QR code: %v", err),
			}, err
		}
	}

	// Wait for login with status updates
	statusChan := make(chan int, 1)
	resultChan := make(chan *LoginResult, 1)

	go func() {
		tokenInfo, err := l.auth.WaitForLogin(ctx, qrResp.SessionKey, func(status int) {
			select {
			case statusChan <- status:
			default:
			}
		})

		if err != nil {
			resultChan <- &LoginResult{
				Success: false,
				Message: fmt.Sprintf("Login failed: %v", err),
			}
			return
		}

		// Save token
		if err := l.auth.SaveToken(l.accountID, tokenInfo); err != nil {
			logger.Warn("Failed to save token", zap.Error(err))
		}

		resultChan <- &LoginResult{
			Success:  true,
			UserID:   tokenInfo.UserID,
			NickName: tokenInfo.NickName,
			Message:  "Login successful",
		}
	}()

	// Print status updates
	for {
		select {
		case <-ctx.Done():
			return &LoginResult{
				Success: false,
				Message: "Login cancelled",
			}, ctx.Err()
		case status := <-statusChan:
			switch status {
			case QRCodeStatusScanned:
				fmt.Println("\n✓ QR code scanned! Waiting for confirmation...")
			case QRCodeStatusExpired:
				return &LoginResult{
					Success: false,
					Message: "QR code expired, please try again",
				}, nil
			}
		case result := <-resultChan:
			return result, nil
		}
	}
}

// displayQRCode displays the QR code in terminal
func (l *WeixinLogin) displayQRCode(qrURL string) error {
	fmt.Println("\n📱 Weixin Login")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Scan the QR code below with Weixin:")
	fmt.Println()
	fmt.Println("  1. Open Weixin on your phone")
	fmt.Println("  2. Go to 'Me' > 'Settings' > 'Devices'")
	fmt.Println("  3. Tap 'Scan QR Code' or use scan from chat")
	fmt.Println("  4. Scan the QR code image")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	// Download the QR code image from the URL
	qrImage, err := downloadQRCodeImage(qrURL)
	if err != nil {
		// If we can't download the image, just show the URL
		fmt.Println("QR Code URL:", qrURL)
		fmt.Println()
		fmt.Println("Please open the URL above in a browser to see the QR code.")
		fmt.Println()
		fmt.Println("Waiting for scan...")
		return nil
	}

	// Display the QR code image in terminal
	displayImageInTerminal(qrImage)

	fmt.Println()
	fmt.Println("QR Code URL:", qrURL)
	fmt.Println()
	fmt.Println("Waiting for scan...")

	return nil
}

// downloadQRCodeImage downloads a QR code image from a URL
func downloadQRCodeImage(url string) (image.Image, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download QR code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download QR code: status %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode QR code image: %w", err)
	}

	return img, nil
}

// displayImageInTerminal displays an image in the terminal using ASCII art
func displayImageInTerminal(img image.Image) {
	// Resize image to fit terminal (approximately 40 characters wide)
	resized := imaging.Resize(img, 40, 0, imaging.Lanczos)
	bounds := resized.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// ASCII characters for different brightness levels
	chars := " .:-=+*#%@"

	for y := 0; y < height; y++ {
		var line string
		for x := 0; x < width; x++ {
			c := resized.At(x, y)
			// Convert to grayscale
			r, g, b, _ := c.RGBA()
			gray := (r + g + b) / 3
			// Map to character
			charIndex := int(gray * uint32(len(chars)-1)) / 65535
			if charIndex >= len(chars) {
				charIndex = len(chars) - 1
			}
			line += string(chars[charIndex]) + string(chars[charIndex])
		}
		fmt.Println(line)
	}
}

// displayQRCodeFromData encodes data directly into a QR code and displays it
func displayQRCodeFromData(data string) {
	qrterminal.Generate(data, qrterminal.L, os.Stdout)
}

// displayQRCodeAsBase64 downloads and converts to base64 for debugging
func displayQRCodeAsBase64(url string) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("Failed to fetch QR code image")
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read QR code image")
		return
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	fmt.Printf("QR Code Base64 (len=%d): %s...\n", len(encoded), encoded[:min(100, len(encoded))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Logout logs out from Weixin
func (l *WeixinLogin) Logout() error {
	return l.auth.DeleteToken(l.accountID)
}

// Status checks the login status
func (l *WeixinLogin) Status() (*TokenInfo, error) {
	return l.auth.LoadToken(l.accountID)
}

// RunWeixinLogin is the main entry point for CLI login
func RunWeixinLogin(accountID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	login, err := NewWeixinLogin(accountID)
	if err != nil {
		return fmt.Errorf("failed to create login handler: %w", err)
	}

	result, err := login.Login(ctx)
	if err != nil {
		return err
	}

	if result.Success {
		fmt.Printf("\n✅ Login successful!\n")
		fmt.Printf("   User ID: %s\n", result.UserID)
		fmt.Printf("   Nickname: %s\n", result.NickName)
		fmt.Printf("\nYou can now start the goclaw service with the weixin channel enabled.\n")
	} else {
		fmt.Printf("\n❌ Login failed: %s\n", result.Message)
	}

	return nil
}

// RunWeixinLogout logs out from Weixin
func RunWeixinLogout(accountID string) error {
	login, err := NewWeixinLogin(accountID)
	if err != nil {
		return fmt.Errorf("failed to create login handler: %w", err)
	}

	if err := login.Logout(); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	fmt.Printf("✅ Logged out from account: %s\n", accountID)
	return nil
}

// RunWeixinStatus checks the login status
func RunWeixinStatus(accountID string) error {
	login, err := NewWeixinLogin(accountID)
	if err != nil {
		return fmt.Errorf("failed to create login handler: %w", err)
	}

	tokenInfo, err := login.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if tokenInfo == nil {
		fmt.Printf("❌ Not logged in for account: %s\n", accountID)
		fmt.Println("Run 'goclaw weixin login' to login.")
		return nil
	}

	fmt.Printf("Account: %s\n", accountID)
	fmt.Printf("  User ID: %s\n", tokenInfo.UserID)
	fmt.Printf("  Nickname: %s\n", tokenInfo.NickName)

	if tokenInfo.ExpiresAt > 0 {
		expiresAt := time.Unix(tokenInfo.ExpiresAt, 0)
		if time.Now().After(expiresAt) {
			fmt.Printf("  Status: ❌ Token expired at %s\n", expiresAt.Format(time.RFC3339))
		} else {
			remaining := time.Until(expiresAt).Round(time.Minute)
			fmt.Printf("  Status: ✅ Logged in (expires in %s)\n", remaining)
		}
	} else {
		fmt.Printf("  Status: ✅ Logged in\n")
	}

	return nil
}
