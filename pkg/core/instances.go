package core

import (
	"fmt"
	"time"

	"github.com/supergiant/guber"
	"github.com/supergiant/supergiant/pkg/model"
	"github.com/supergiant/supergiant/pkg/util"
)

type Instances struct {
	Collection
}

func (c *Instances) CreateVolumes(id *int64, m *model.Instance) error {
	action := &Action{
		Status: &model.ActionStatus{
			Description: "creating volumes",
			MaxRetries:  1, // see NOTE in Volume
		},
		core:  c.core,
		scope: c.core.DB.Preload("Component.App.Kube.CloudAccount").Preload("Component.CurrentRelease").Preload("Component.TargetRelease").Preload("Release").Preload("Volumes.Kube.CloudAccount"), // voluemes preloaded in case of retry and loading
		model: m,
		id:    id,
		fn: func(_ *Action) error {
			return c.inParallel(m.Release.Config.Volumes, func(vc interface{}) error {
				volConf := vc.(*model.VolumeBlueprint)
				var volume *model.Volume

				for _, existingVol := range m.Volumes {
					if existingVol.Name == volConf.Name {
						volume = existingVol
						break
					}
				}

				if volume == nil {
					volume = &model.Volume{
						Instance:   m,
						InstanceID: m.ID,
						Kube:       m.Component.App.Kube,
						KubeID:     m.Component.App.KubeID,
						Name:       volConf.Name,
						Type:       volConf.Type,
						Size:       volConf.Size,
					}
					if err := c.core.Volumes.Create(volume); err != nil {
						return err
					}
					m.Volumes = append(m.Volumes, volume)
				}

				if volume.ProviderID == "" {
					if err := c.core.Volumes.Provision(volume.ID, volume).Now(); err != nil {
						return err
					}
				}

				return nil
			})
		},
	}
	return action.Now()
}

func (c *Instances) Start(id *int64, m *model.Instance) *Action {
	return &Action{
		Status: &model.ActionStatus{
			Description: "starting",
			MaxRetries:  5,
		},
		core:  c.core,
		scope: c.core.DB.Preload("Component.App.Kube.CloudAccount").Preload("Component.PrivateImageKeys.Key").Preload("Component.CurrentRelease").Preload("Component.TargetRelease").Preload("Release").Preload("Volumes.Kube.CloudAccount"),
		model: m,
		id:    id,
		fn: func(_ *Action) error {
			if m.Started {
				return fmt.Errorf("Instance %d already started", m.ID)
			}

			// Update ReleaseID first, so that the correct config can be used following.
			// (Only needs to be done if already exists with current release)
			if *m.ReleaseID != *m.Component.TargetReleaseID { // <- no need to do this on first Release
				m.ReleaseID = m.Component.TargetReleaseID
				m.Release = m.Component.TargetRelease
				if err := c.core.DB.Save(m); err != nil {
					return err
				}
			}

			// Load ServiceSet
			serviceSet, err := c.serviceSet(m)
			if err != nil {
				return err
			}
			if err := serviceSet.provision(); err != nil {
				return err
			}
			if err := serviceSet.addNewPorts(); err != nil {
				return err
			}
			if err := serviceSet.removeOldPorts(); err != nil {
				return err
			}

			// Save Addresses
			externalAddrs, err := serviceSet.externalAddresses()
			if err != nil {
				return err
			}
			internalAddrs, err := serviceSet.internalAddresses()
			if err != nil {
				return err
			}
			m.Addresses = &model.Addresses{
				External: externalAddrs,
				Internal: internalAddrs,
			}

			// TODO sloppy... we need a way (can't do this at initialize due to how we set
			// Release above) to get the volume conf from the Release for a volume
			err = c.inParallel(m.Volumes, func(vi interface{}) error {
				volume := vi.(*model.Volume)
				for _, volConf := range m.Release.Config.Volumes {
					if volume.Name == volConf.Name && volume.Size != volConf.Size {
						volume.Size = volConf.Size
						if err := c.core.Volumes.Resize(volume.ID, volume).Now(); err != nil {
							return err
						}
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

			if err := c.provisionReplicationController(m); err != nil {
				return err
			}

			err = util.WaitFor(fmt.Sprintf("Pod of Instance %d to start", m.ID), 20*time.Minute, 3*time.Second, func() (bool, error) {
				pod, err := c.pod(m)
				if err != nil {
					if _, podNotFound := err.(*PodNotFoundError); podNotFound {
						return false, nil
					} else {
						return false, err
					}
				}
				return pod.IsReady(), nil
			})
			if err != nil {
				return err
			}
			return c.core.DB.Model(m).Update("started", true).Error
		},
	}
}

func (c *Instances) Stop(id *int64, m *model.Instance) *Action {
	return &Action{
		Status: &model.ActionStatus{
			Description: "stopping",
			MaxRetries:  5,
		},
		core:  c.core,
		scope: c.core.DB.Preload("Component.App.Kube.CloudAccount").Preload("Component.CurrentRelease").Preload("Component.TargetRelease").Preload("Release").Preload("Volumes.Kube.CloudAccount"),
		model: m,
		id:    id,
		fn: func(_ *Action) error {
			if err := c.deleteReplicationControllerAndPod(m); err != nil {
				return err
			}

			return c.core.DB.Model(m).Update("started", false).Error
		},
	}
}

func (c *Instances) Delete(id *int64, m *model.Instance) *Action {
	return &Action{
		Status: &model.ActionStatus{
			Description: "deleting",
			MaxRetries:  5,
		},
		core:           c.core,
		scope:          c.core.DB.Preload("Component.App.Kube.CloudAccount").Preload("Component.CurrentRelease").Preload("Component.TargetRelease").Preload("Release").Preload("Volumes.Kube.CloudAccount"),
		model:          m,
		id:             id,
		cancelExisting: true,
		fn: func(_ *Action) error {
			if err := c.deleteReplicationControllerAndPod(m); err != nil {
				return err
			}
			for _, volume := range m.Volumes {
				if err := c.core.Volumes.Delete(volume.ID, volume).Now(); err != nil {
					return err
				}
			}
			serviceSet, err := c.serviceSet(m)
			if err != nil {
				return err
			}
			if err := serviceSet.delete(); err != nil {
				return err
			}
			return c.Collection.Delete(id, m)
		},
	}
}

func (c *Instances) Log(m *model.Instance) (string, error) {
	pod, err := c.pod(m)
	if err != nil {
		return "", err
	}
	return pod.Log(m.Release.Config.Containers[0].NameOrDefault())
}

////////////////////////////////////////////////////////////////////////////////
// Private methods                                                            //
////////////////////////////////////////////////////////////////////////////////

type PodNotFoundError struct {
	InstanceID *int64
}

func (err *PodNotFoundError) Error() string {
	return fmt.Sprintf("Pod not found for Instance %d", err.InstanceID)
}

func (c *Instances) pod(m *model.Instance) (*guber.Pod, error) {
	q := &guber.QueryParams{
		LabelSelector: "instance=" + m.Name,
	}
	pods, err := c.core.K8S(m.Component.App.Kube).Pods(m.Component.App.Name).Query(q)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 1 {
		return pods.Items[0], nil
	}
	return nil, &PodNotFoundError{m.ID}
}

func (c *Instances) serviceSet(m *model.Instance) (*ServiceSet, error) {
	labelSelector := map[string]string{"instance_service": m.Name}
	portFilter := func(port *model.Port) bool { return port.PerInstance }
	return NewServiceSet(c.core, m.Component, m.Release, m.Name, labelSelector, portFilter)
}

func (c *Instances) provisionReplicationController(m *model.Instance) error {
	if _, err := c.core.K8S(m.Component.App.Kube).ReplicationControllers(m.Component.App.Name).Get(m.Name); err == nil {
		return nil // already provisioned
	} else if !isKubeNotFoundErr(err) {
		return err
	}

	var containers []*guber.Container
	for _, blueprint := range m.Release.Config.Containers {
		containers = append(containers, asKubeContainer(blueprint, m))
	}

	var kubeVolumes []*guber.Volume
	for _, volume := range m.Volumes {
		kubeVolume := c.core.CloudAccounts.provider(volume.Kube.CloudAccount).KubernetesVolumeDefinition(volume)
		kubeVolumes = append(kubeVolumes, kubeVolume)
	}

	var pullSecrets []*guber.ImagePullSecret
	for _, compKey := range m.Component.PrivateImageKeys {
		secret := &guber.ImagePullSecret{
			Name: compKey.Key.Username,
		}
		pullSecrets = append(pullSecrets, secret)
	}

	rc := &guber.ReplicationController{
		Metadata: &guber.Metadata{
			Name: m.Name,
		},
		Spec: &guber.ReplicationControllerSpec{
			Selector: map[string]string{
				"instance": m.Name,
			},
			Replicas: 1,
			Template: &guber.PodTemplate{
				Metadata: &guber.Metadata{
					Name: m.Name, // pod base name is same as RC
					Labels: map[string]string{
						"service":          m.Component.Name, // for Service
						"instance":         m.Name,           // for RC (above)
						"instance_service": m.Name,           // for Instance Service
					},
				},
				Spec: &guber.PodSpec{
					Volumes:                       kubeVolumes,
					Containers:                    containers,
					ImagePullSecrets:              pullSecrets,
					TerminationGracePeriodSeconds: m.Release.Config.TerminationGracePeriod,
				},
			},
		},
	}
	_, err := c.core.K8S(m.Component.App.Kube).ReplicationControllers(m.Component.App.Name).Create(rc)
	return err
}

func (c *Instances) deleteReplicationControllerAndPod(m *model.Instance) error {
	pod, err := c.pod(m)
	if err != nil {
		if _, podNotFound := err.(*PodNotFoundError); podNotFound {
			return nil
		}
		return err
	}

	if err := c.core.K8S(m.Component.App.Kube).ReplicationControllers(m.Component.App.Name).Delete(m.Name); err != nil && !isKubeNotFoundErr(err) {
		return err
	}

	if err := pod.Delete(); err != nil {
		return err
	}
	// Wait for volume detach
	for _, volume := range m.Volumes {
		if err := c.core.Volumes.WaitForAvailable(volume.ID, volume); err != nil {
			return err
		}
	}
	return nil
}
