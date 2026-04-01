package channels

import (
	"fmt"
)

// GetDefaultCommandResponse returns standard command responses
// This helps reduce code duplication across different channel implementations
func GetDefaultCommandResponse(command string, isRunning bool) string {
	switch command {
	case "/start":
		return "👋 欢迎使用 goclaw!\n\n我可以帮助你完成各种任务。发送 /help 查看可用命令。"

	case "/help":
		return `🐾 goclaw 命令列表：

/start - 开始使用
/help - 显示帮助

你可以直接与我对话，我会尽力帮助你！`

	case "/status":
		status := "🔴 离线"
		if isRunning {
			status = "🟢 在线"
		}
		return fmt.Sprintf("✅ goclaw 运行中\n\n通道状态: %s", status)

	default:
		return ""
	}
}

// CommandResponse represents the response to a command
type CommandResponse struct {
	Content string
	Media   []bus.Media
	ReplyTo string
}

// DefaultCommandHandler provides standard command handling logic
type DefaultCommandHandler struct {
	channelName string
	isRunning   func() bool
}

// NewDefaultCommandHandler creates a new default command handler
func NewDefaultCommandHandler(channelName string, isRunning func() bool) *DefaultCommandHandler {
	return &DefaultCommandHandler{
		channelName: channelName,
		isRunning:   isRunning,
	}
}

// HandleCommand implements CommandHandler interface
func (h *DefaultCommandHandler) HandleCommand(command string, chatID string) (*CommandResponse, error) {
	switch command {
	case "/start":
		return &CommandResponse{
			Content: "👋 欢迎使用 goclaw!\n\n我可以帮助你完成各种任务。发送 /help 查看可用命令。",
		}, nil

	case "/help":
		helpText := `🐾 goclaw 命令列表：

/start - 开始使用
/help - 显示帮助

你可以直接与我对话，我会尽力帮助你！`
		return &CommandResponse{
			Content: helpText,
		}, nil

	case "/status":
		status := "🔴 离线"
		if h.isRunning() {
			status = "🟢 在线"
		}
		return &CommandResponse{
			Content: fmt.Sprintf("✅ goclaw 运行中\n\n通道状态: %s", status),
		}, nil

	default:
		return nil, nil
	}
}
