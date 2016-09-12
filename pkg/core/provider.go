package core

import (
	"github.com/supergiant/guber"
	"github.com/supergiant/supergiant/pkg/model"
)

type Provider interface {
	ValidateAccount(*model.CloudAccount) error

	CreateKube(*model.Kube, *Action) error
	DeleteKube(*model.Kube) error

	CreateNode(*model.Node, *Action) error
	DeleteNode(*model.Node) error

	CreateVolume(*model.Volume, *Action) error
	KubernetesVolumeDefinition(*model.Volume) *guber.Volume
	WaitForVolumeAvailable(*model.Volume, *Action) error
	ResizeVolume(*model.Volume, *Action) error
	DeleteVolume(*model.Volume) error

	CreateEntrypoint(*model.Entrypoint, *Action) error
	AddPortToEntrypoint(*model.Entrypoint, int64, int64) error
	RemovePortFromEntrypoint(*model.Entrypoint, int64) error
	DeleteEntrypoint(*model.Entrypoint) error
}
