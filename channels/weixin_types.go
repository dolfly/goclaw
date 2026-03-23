package channels

// Message item types for Weixin
const (
	MessageItemTypeText  = 1
	MessageItemTypeImage = 2
	MessageItemTypeVoice = 3
	MessageItemTypeFile  = 4
	MessageItemTypeVideo = 5
)

// WeixinMessage represents a message from Weixin
type WeixinMessage struct {
	Seq          int64         `json:"seq,omitempty"`
	MessageID    int64         `json:"message_id,omitempty"`
	FromUserID   string        `json:"from_user_id,omitempty"`
	ToUserID     string        `json:"to_user_id,omitempty"`
	CreateTimeMs int64         `json:"create_time_ms,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
	MessageType  int           `json:"message_type,omitempty"`
	MessageState int           `json:"message_state,omitempty"`
	ItemList     []MessageItem `json:"item_list,omitempty"`
	ContextToken string        `json:"context_token,omitempty"`
}

// MessageItem represents an item in a Weixin message
type MessageItem struct {
	ItemType int            `json:"item_type,omitempty"`
	Text     *TextItem      `json:"text,omitempty"`
	Image    *CDNMedia      `json:"image,omitempty"`
	Voice    *CDNMedia      `json:"voice,omitempty"`
	File     *FileItem      `json:"file,omitempty"`
	Video    *VideoItem     `json:"video,omitempty"`
}

// TextItem represents a text message item
type TextItem struct {
	Text string `json:"text,omitempty"`
}

// CDNMedia represents a CDN media reference
type CDNMedia struct {
	CDNMediaID      string `json:"cdn_media_id,omitempty"`
	AESKey          string `json:"aes_key,omitempty"`
	EncryptedSize   int64  `json:"encrypted_size,omitempty"`
	OriginalSize    int64  `json:"original_size,omitempty"`
	DownloadURL     string `json:"download_url,omitempty"`
	DownloadParam   string `json:"download_encrypted_query_param,omitempty"`
	FileName        string `json:"file_name,omitempty"`
	FileExtension   string `json:"file_extension,omitempty"`
	MimeType        string `json:"mime_type,omitempty"`
	DurationSeconds int    `json:"duration_seconds,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
}

// FileItem represents a file message item
type FileItem struct {
	CDNMedia
	FileSize int64 `json:"file_size,omitempty"`
}

// VideoItem represents a video message item
type VideoItem struct {
	CDNMedia
	ThumbCDNMediaID string `json:"thumb_cdn_media_id,omitempty"`
	ThumbAESKey     string `json:"thumb_aes_key,omitempty"`
}

// GetUpdatesReq is the request for getUpdates API
type GetUpdatesReq struct {
	GetUpdatesBuf string `json:"get_updates_buf,omitempty"`
}

// GetUpdatesResp is the response from getUpdates API
type GetUpdatesResp struct {
	BaseResponse
	GetUpdatesBuf string          `json:"get_updates_buf,omitempty"`
	Msgs          []*WeixinMessage `json:"msgs,omitempty"`
}

// SendMessageReq is the request for sendMessage API
type SendMessageReq struct {
	ToUserID     string       `json:"to_user_id,omitempty"`
	ContextToken string       `json:"context_token,omitempty"`
	ItemList     []MessageItem `json:"item_list,omitempty"`
}

// SendMessageResp is the response from sendMessage API
type SendMessageResp struct {
	BaseResponse
	MessageID int64 `json:"message_id,omitempty"`
}

// GetUploadUrlReq is the request for getUploadUrl API
type GetUploadUrlReq struct {
	ToUserID         string `json:"to_user_id,omitempty"`
	FileName         string `json:"file_name,omitempty"`
	FileExtension    string `json:"file_extension,omitempty"`
	FileSize         int64  `json:"file_size,omitempty"`
	MimeType         string `json:"mime_type,omitempty"`
	DurationSeconds  int    `json:"duration_seconds,omitempty"` // for audio/video
	Width            int    `json:"width,omitempty"`           // for image/video
	Height           int    `json:"height,omitempty"`          // for image/video
}

// GetUploadUrlResp is the response from getUploadUrl API
type GetUploadUrlResp struct {
	BaseResponse
	CDNMediaID    string `json:"cdn_media_id,omitempty"`
	AESKey        string `json:"aes_key,omitempty"`
	UploadURL     string `json:"upload_url,omitempty"`
	UploadAuthKey string `json:"upload_auth_key,omitempty"`
}

// GetConfigReq is the request for getConfig API
type GetConfigReq struct {
	UserID string `json:"user_id,omitempty"`
}

// GetConfigResp is the response from getConfig API
type GetConfigResp struct {
	BaseResponse
	TypingTicket string `json:"typing_ticket,omitempty"`
}

// SendTypingReq is the request for sendTyping API
type SendTypingReq struct {
	ToUserID     string `json:"to_user_id,omitempty"`
	TypingTicket string `json:"typing_ticket,omitempty"`
	TypingStatus int    `json:"typing_status,omitempty"` // 1: typing, 0: not typing
}

// SendTypingResp is the response from sendTyping API
type SendTypingResp struct {
	BaseResponse
}

// GetBotQRCodeReq is the request for get_bot_qrcode API
type GetBotQRCodeReq struct {
	// Empty request
}

// GetBotQRCodeResp is the response from get_bot_qrcode API
type GetBotQRCodeResp struct {
	BaseResponse
	QRCodeURL  string `json:"qrcode_url,omitempty"`
	SessionKey string `json:"session_key,omitempty"`
}

// GetQRCodeStatusReq is the request for get_qrcode_status API
type GetQRCodeStatusReq struct {
	SessionKey string `json:"session_key,omitempty"`
}

// GetQRCodeStatusResp is the response from get_qrcode_status API
type GetQRCodeStatusResp struct {
	BaseResponse
	Status      int    `json:"status,omitempty"`   // 0: waiting, 1: scanned, 2: confirmed, 3: expired
	Token       string `json:"token,omitempty"`    // available when status == 2
	ExpiresIn   int    `json:"expires_in,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	NickName    string `json:"nick_name,omitempty"`
	HeadImgURL  string `json:"head_img_url,omitempty"`
}

// QRCode status constants
const (
	QRCodeStatusWaiting  = 0
	QRCodeStatusScanned  = 1
	QRCodeStatusConfirmed = 2
	QRCodeStatusExpired  = 3
)

// BaseResponse is the common response structure
type BaseResponse struct {
	ErrCode int    `json:"err_code,omitempty"`
	ErrMsg  string `json:"err_msg,omitempty"`
}

// IsSuccess checks if the response indicates success
func (r *BaseResponse) IsSuccess() bool {
	return r.ErrCode == 0
}

// Error returns the error message
func (r *BaseResponse) Error() string {
	return r.ErrMsg
}

// WeixinConfig is the configuration for Weixin channel
type WeixinConfig struct {
	BaseChannelConfig
	BaseURL    string `mapstructure:"base_url" json:"base_url"`
	CDNBaseURL string `mapstructure:"cdn_base_url" json:"cdn_base_url"`
	Token      string `mapstructure:"token" json:"token"`
}

// TokenInfo stores token information
type TokenInfo struct {
	Token     string `json:"token,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	NickName  string `json:"nick_name,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}
