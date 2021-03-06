package core

import (
	"fmt"

	"github.com/supergiant/supergiant/pkg/model"
)

type Entrypoints struct {
	Collection
}

func (c *Entrypoints) Create(m *model.Entrypoint) error {
	if err := c.Collection.Create(m); err != nil {
		return err
	}

	// Load Kube and CloudAccount
	if err := c.core.DB.Preload("Nodes").Preload("CloudAccount").First(m.Kube, m.KubeID); err != nil {
		return err
	}

	provision := &Action{
		Status: &model.ActionStatus{
			Description: "provisioning",
			MaxRetries:  5,
		},
		core:       c.core,
		resourceID: m.UUID,
		model:      m,
		fn: func(a *Action) error {
			return c.core.CloudAccounts.provider(m.Kube.CloudAccount).CreateEntrypoint(m, a)
		},
	}
	return provision.Async()
}

func (c *Entrypoints) Delete(id *int64, m *model.Entrypoint) *Action {
	return &Action{
		Status: &model.ActionStatus{
			Description: "deleting",
			MaxRetries:  5,
		},
		core:  c.core,
		scope: c.core.DB.Preload("Kube.CloudAccount"),
		model: m,
		id:    id,
		fn: func(_ *Action) error {
			if err := c.core.CloudAccounts.provider(m.Kube.CloudAccount).DeleteEntrypoint(m); err != nil {
				return err
			}
			return c.Collection.Delete(id, m)
		},
	}
}

func (c *Entrypoints) SetPort(id *int64, m *model.Entrypoint, elbPort int64, instancePort int64) error {
	action := &Action{
		Status: &model.ActionStatus{
			Description: fmt.Sprintf("setting port %d:%d", elbPort, instancePort),
			MaxRetries:  5,
		},
		core:       c.core,
		scope:      c.core.DB.Preload("Kube.CloudAccount"),
		model:      m,
		id:         id,
		resourceID: m.UUID,
		fn: func(_ *Action) error {
			return c.core.CloudAccounts.provider(m.Kube.CloudAccount).AddPortToEntrypoint(m, elbPort, instancePort)
		},
	}
	return action.Now()
}

func (c *Entrypoints) RemovePort(id *int64, m *model.Entrypoint, elbPort int64) error {
	action := &Action{
		Status: &model.ActionStatus{
			Description: fmt.Sprintf("removing port %d", elbPort),
			MaxRetries:  5,
		},
		core:       c.core,
		scope:      c.core.DB.Preload("Kube.CloudAccount"),
		model:      m,
		id:         id,
		resourceID: m.UUID,
		fn: func(_ *Action) error {
			return c.core.CloudAccounts.provider(m.Kube.CloudAccount).RemovePortFromEntrypoint(m, elbPort)
		},
	}
	return action.Now()
}
