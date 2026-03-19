package gateway

import (
	"reflect"

	"github.com/smallnest/goclaw/acp"
)

// registerAcpMethods 注册 ACP 方法
func (h *Handler) registerAcpMethods() {
	// Check if ACP manager is available
	var acpManager *acp.Manager
	if h.acpMgr != nil && !reflect.ValueOf(h.acpMgr).IsNil() {
		if mgr, ok := h.acpMgr.(*acp.Manager); ok {
			acpManager = mgr
		}
	}

	// Register ACP methods - if ACP is not enabled, methods will return appropriate error
	RegisterAcpMethods(h.registry, h.cfg, acpManager)
}
