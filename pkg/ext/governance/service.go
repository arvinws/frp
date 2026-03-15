package governance

import (
	"fmt"
	"strings"

	"github.com/fatedier/frp/pkg/ext/banlist"
	"github.com/fatedier/frp/pkg/ext/clientmgr"
	"github.com/fatedier/frp/pkg/msg"
)

type ProxyMetadata struct {
	ClientID string
}

type ProxyMetadataProvider interface {
	GetProxyMetadata(name string) (ProxyMetadata, bool)
}

type ProxyRuntimeController interface {
	CloseProxyByName(name string) bool
	SendProxyControlByRunID(runID, proxyName, action string) bool
}

type ProxyActionResult struct {
	ProxyName   string `json:"proxyName"`
	Source      string `json:"source,omitempty"`
	Changed     bool   `json:"changed"`
	Disabled    bool   `json:"disabled"`
	ControlSent bool   `json:"controlSent"`
	Closed      bool   `json:"closed"`
	Result      string `json:"result"`
}

type Service struct {
	banStore         banlist.Store
	sessionManager   *clientmgr.Manager
	proxyRuntime     ProxyRuntimeController
	proxyMetadataSrc ProxyMetadataProvider
}

const scheduleProxySourcePrefix = "schedule:"

func NewService(
	banStore banlist.Store,
	sessionManager *clientmgr.Manager,
	proxyRuntime ProxyRuntimeController,
	proxyMetadataSrc ProxyMetadataProvider,
) *Service {
	return &Service{
		banStore:         banStore,
		sessionManager:   sessionManager,
		proxyRuntime:     proxyRuntime,
		proxyMetadataSrc: proxyMetadataSrc,
	}
}

func ManualProxySource() string {
	return banlist.ProxyDisableSourceManual
}

func ScheduleProxySource(taskID string) string {
	return scheduleProxySourcePrefix + strings.TrimSpace(taskID)
}

func (s *Service) DisableProxy(proxyName, source, reason, operator string) (ProxyActionResult, error) {
	proxyName = strings.TrimSpace(proxyName)
	if proxyName == "" {
		return ProxyActionResult{}, fmt.Errorf("proxy name is required")
	}
	if s.banStore == nil {
		return ProxyActionResult{}, fmt.Errorf("banlist store unavailable")
	}
	source = normalizeSource(source)

	changed := s.banStore.DisableProxySource(proxyName, source, strings.TrimSpace(reason), strings.TrimSpace(operator))
	controlSent := s.sendProxyControl(proxyName, msg.ProxyControlActionDisable)
	closed := false
	if s.proxyRuntime != nil {
		closed = s.proxyRuntime.CloseProxyByName(proxyName)
	}

	result := "disabled"
	if controlSent && closed {
		result = "disabled_and_closed"
	} else if controlSent {
		result = "disabled_and_controlled"
	} else if closed {
		result = "disabled_and_closed"
	}

	return ProxyActionResult{
		ProxyName:   proxyName,
		Source:      source,
		Changed:     changed,
		Disabled:    true,
		ControlSent: controlSent,
		Closed:      closed,
		Result:      result,
	}, nil
}

func (s *Service) EnableProxy(proxyName, source string) (ProxyActionResult, error) {
	proxyName = strings.TrimSpace(proxyName)
	if proxyName == "" {
		return ProxyActionResult{}, fmt.Errorf("proxy name is required")
	}
	if s.banStore == nil {
		return ProxyActionResult{}, fmt.Errorf("banlist store unavailable")
	}
	source = normalizeSource(source)

	changed := s.banStore.EnableProxySource(proxyName, source)
	if source == ManualProxySource() {
		changed = s.clearScheduleSources(proxyName) || changed
	}
	disabled := s.banStore.IsProxyDisabled(proxyName)
	controlSent := false
	result := "already_enabled"
	if disabled {
		result = "still_disabled"
	} else {
		controlSent = s.sendProxyControl(proxyName, msg.ProxyControlActionEnable)
		result = "enabled"
		if !changed && !controlSent {
			result = "already_enabled"
		}
	}

	return ProxyActionResult{
		ProxyName:   proxyName,
		Source:      source,
		Changed:     changed,
		Disabled:    disabled,
		ControlSent: controlSent,
		Closed:      false,
		Result:      result,
	}, nil
}

func (s *Service) clearScheduleSources(proxyName string) bool {
	record := s.banStore.GetProxy(proxyName)
	changed := false
	for _, source := range record.Sources {
		if !strings.HasPrefix(strings.TrimSpace(source.Source), scheduleProxySourcePrefix) {
			continue
		}
		changed = s.banStore.EnableProxySource(proxyName, source.Source) || changed
	}
	return changed
}

func (s *Service) sendProxyControl(proxyName, action string) bool {
	if s.proxyRuntime == nil || s.sessionManager == nil || s.proxyMetadataSrc == nil {
		return false
	}
	meta, ok := s.proxyMetadataSrc.GetProxyMetadata(proxyName)
	if !ok || strings.TrimSpace(meta.ClientID) == "" {
		return false
	}
	session, ok := s.sessionManager.GetByClientID(meta.ClientID)
	if !ok || session.RunID == "" {
		return false
	}
	return s.proxyRuntime.SendProxyControlByRunID(session.RunID, proxyName, action)
}

func normalizeSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return banlist.ProxyDisableSourceManual
	}
	return source
}
