package core

import "github.com/supergiant/supergiant/pkg/model"

type EntrypointListeners struct {
	Collection
}

func (c *EntrypointListeners) Create(m *model.EntrypointListener) error {
	if err := c.Collection.Create(m); err != nil {
		return err
	}
	if err := c.core.DB.Preload("Kube.CloudAccount").First(m.Entrypoint, m.EntrypointID); err != nil {
		return err
	}
	action := &Action{
		Status: &model.ActionStatus{
			Description: "provisioning",
			MaxRetries:  5,
		},
		core:       c.core,
		resourceID: m.UUID,
		model:      m,
		fn: func(_ *Action) error {
			return c.core.CloudAccounts.provider(m.Entrypoint.Kube.CloudAccount).CreateEntrypointListener(m)
		},
	}
	return action.Now()
}

func (c *EntrypointListeners) Delete(id *int64, m *model.EntrypointListener) *Action {
	return &Action{
		Status: &model.ActionStatus{
			Description: "deleting",
			MaxRetries:  5,
		},
		core:       c.core,
		scope:      c.core.DB.Preload("Entrypoint.Kube.CloudAccount"),
		model:      m,
		id:         id,
		resourceID: m.UUID,
		fn: func(_ *Action) error {
			return c.core.CloudAccounts.provider(m.Entrypoint.Kube.CloudAccount).DeleteEntrypointListener(m)
		},
	}
}
