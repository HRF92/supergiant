package core

import (
	"regexp"
	"strconv"

	"github.com/supergiant/guber"
	"github.com/supergiant/supergiant/pkg/model"
)

type Nodes struct {
	Collection
}

func (c *Nodes) Create(m *model.Node) error {
	if err := c.Collection.Create(m); err != nil {
		return err
	}

	provision := &Action{
		Status: &model.ActionStatus{
			Description: "provisioning",

			// TODO
			// This resource has an issue with retryable provisioning -- which in this
			// context means creating an remote asset from the local record.
			//
			// Apps, for example, which use their user-set Name field as the actual
			// identifier for the provisioned Kubernetes Namespace. That makes the
			// creation of the Namespace retryable, because it is IDEMPOTENT.
			//
			// The problem here, is that WE CANNOT SET AN IDENTIFIER UP FRONT. The ID
			// is given to us upon successful creation of the remote asset.
			//
			// We currently do not have a great solution in place for this problem.
			// In the meantime, MaxRetries is set low to prevent creating several
			// duplicate, billable assets in the user's cloud account. If there is an
			// error, the user will know about it quickly, instead of after 20 retries.
			MaxRetries: 0,
		},
		core:  c.core,
		scope: c.core.DB.Preload("Kube.CloudAccount").Preload("Kube.Entrypoints.Kube.CloudAccount"),
		model: m,
		id:    m.ID,
		fn: func(a *Action) error {
			return c.core.CloudAccounts.provider(m.Kube.CloudAccount).CreateNode(m, a)
		},
	}
	return provision.Async()
}

func (c *Nodes) Delete(id *int64, m *model.Node) *Action {
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
			if m.ProviderID == "" {
				c.core.Log.Warnf("Deleting Node %d which has no provider_id", *m.ID)
			} else {
				if err := c.core.CloudAccounts.provider(m.Kube.CloudAccount).DeleteNode(m); err != nil {
					return err
				}
			}
			return c.Collection.Delete(id, m)
		},
	}
}

////////////////////////////////////////////////////////////////////////////////
// Private methods                                                            //
////////////////////////////////////////////////////////////////////////////////

func (c *Nodes) hasPodsWithReservedResources(m *model.Node) (bool, error) {
	q := &guber.QueryParams{
		FieldSelector: "spec.nodeName=" + m.Name + ",status.phase=Running",
	}
	pods, err := c.core.K8S(m.Kube).Pods("").Query(q)
	if err != nil {
		return false, err
	}

	rxp := regexp.MustCompile("[0-9]+")

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			reqs := container.Resources.Requests

			if reqs == nil {
				continue
			}

			values := [2]string{
				reqs.CPU,
				reqs.Memory,
			}

			for _, val := range values {
				numstr := rxp.FindString(val)
				num := 0
				var err error
				if numstr != "" {
					num, err = strconv.Atoi(numstr)
					if err != nil {
						return false, err
					}
				}

				if num > 0 {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
