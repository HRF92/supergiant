package core

import (
	"github.com/supergiant/supergiant/pkg/model"
	"github.com/supergiant/supergiant/pkg/util"
)

//------------------------------------------------------------------------------

type Kubes struct {
	Collection
}

func (c *Kubes) Create(m *model.Kube) error {
	// Defaults
	if m.Username == "" && m.Password == "" {
		m.Username = util.RandomString(16)
		m.Password = util.RandomString(8)
	}

	if err := c.Collection.Create(m); err != nil {
		return err
	}

	provision := &Action{
		Status: &model.ActionStatus{
			Description: "provisioning",
			MaxRetries:  20,
		},
		core:       c.core,
		resourceID: m.UUID,
		model:      m,
		fn: func(a *Action) error {
			return c.core.CloudAccounts.provider(m.CloudAccount).CreateKube(m, a)
		},
	}
	return provision.Async()
}

func (c *Kubes) Delete(id *int64, m *model.Kube) *Action {
	return &Action{
		Status: &model.ActionStatus{
			Description: "deleting",
			MaxRetries:  5,
		},
		core:           c.core,
		scope:          c.core.DB.Preload("CloudAccount").Preload("Entrypoints").Preload("Volumes.Kube.CloudAccount").Preload("Nodes.Kube.CloudAccount"),
		model:          m,
		id:             id,
		cancelExisting: true,
		fn: func(_ *Action) error {
			for _, entrypoint := range m.Entrypoints {
				if err := c.core.Entrypoints.Delete(entrypoint.ID, entrypoint).Now(); err != nil {
					return err
				}
			}
			// Delete nodes first to get rid of any potential hanging volumes
			for _, node := range m.Nodes {
				if err := c.core.Nodes.Delete(node.ID, node).Now(); err != nil {
					return err
				}
			}
			// Delete Volumes
			for _, volume := range m.Volumes {
				if err := c.core.Volumes.Delete(volume.ID, volume).Now(); err != nil {
					return err
				}
			}
			if err := c.core.CloudAccounts.provider(m.CloudAccount).DeleteKube(m); err != nil {
				return err
			}
			return c.Collection.Delete(id, m)
		},
	}
}
