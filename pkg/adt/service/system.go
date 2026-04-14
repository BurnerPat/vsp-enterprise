package service

import (
	"context"

	"github.com/oisee/vibing-steampunk/pkg/adt/transport"
)

// SystemService provides system-level operations.
type SystemService interface {
	GetSystemInfo(ctx context.Context) (*SystemInfo, error)
	GetInstalledComponents(ctx context.Context) ([]InstalledComponent, error)
}

// SystemInfo represents SAP system information.
type SystemInfo struct {
	SystemID        string `json:"systemId"`
	Client          string `json:"client"`
	SAPRelease      string `json:"sapRelease"`
	KernelRelease   string `json:"kernelRelease,omitempty"`
	DatabaseRelease string `json:"databaseRelease,omitempty"`
	DatabaseSystem  string `json:"databaseSystem,omitempty"`
	HostName        string `json:"hostName,omitempty"`
	InstallNumber   string `json:"installNumber,omitempty"`
	ABAPRelease     string `json:"abapRelease,omitempty"`
}

// InstalledComponent represents an installed software component.
type InstalledComponent struct {
	Name        string `json:"name"`
	Release     string `json:"release"`
	SupportPack string `json:"supportPack,omitempty"`
	Description string `json:"description,omitempty"`
}

type systemService struct {
	baseService
}

// NewSystemService creates a new SystemService.
func NewSystemService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) SystemService {
	return &systemService{baseService{sender: sender, safety: safety, config: cfg}}
}

func (s *systemService) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	// TODO: migrate from client.go — calls RunQuery on T000, CVERS tables
	// For now, delegate to the sender for the basic discovery call.
	_ = ctx
	return nil, nil
}

func (s *systemService) GetInstalledComponents(ctx context.Context) ([]InstalledComponent, error) {
	// TODO: migrate from client.go
	_ = ctx
	return nil, nil
}
