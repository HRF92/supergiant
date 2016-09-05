package core

import (
	"errors"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/supergiant/supergiant/pkg/model"
	"golang.org/x/oauth2"
)

// DOProvider Holds DO account info.
type DOProvider struct {
	core  *Core
	Token string
}

// ValidateAccount Valitades DO account info.
func (p *DOProvider) ValidateAccount(m *model.CloudAccount) error {
	client := p.newClient()
	_, _, err := client.Droplets.List(new(godo.ListOptions))
	return err
}

// CreateKube creates a new DO kubernetes cluster.
func (p *DOProvider) CreateKube(m *model.Kube, action *Action) error {
	procedure := &Procedure{
		core:  p.core,
		name:  "Create Kube",
		model: m,
	}

	client := p.newClient()

	procedure.AddStep("creating global tags for Kube", func() error {
		// These are created once, and then attached by name to created resource
		globalTags := []string{
			"Kubernetes-Cluster",
			m.Name,
			m.Name + "-master",
			m.Name + "-minion",
		}
		for _, tag := range globalTags {
			createInput := &godo.TagCreateRequest{
				Name: tag,
			}
			if _, _, err := client.Tags.Create(createInput); err != nil {
				// TODO
				p.core.Log.Warnf("Failed to create Digital Ocean tag '%s': %s", tag, err)
			}
		}
		return nil
	})

	procedure.AddStep("creating master", func() error {
		if m.MasterPublicIP != "" {
			return nil
		}

		// TODO we should just load this once if no interpolation
		masterUserdata, err := ioutil.ReadFile("config/providers/digitalocean/master.yaml")
		if err != nil {
			return err
		}

		dropletRequest := &godo.DropletCreateRequest{
			Name:              m.Name + "-master",
			Region:            m.DOConfig.Region,
			Size:              m.MasterNodeSize,
			PrivateNetworking: false,
			UserData:          string(masterUserdata),
			SSHKeys: []godo.DropletCreateSSHKey{
				{
					Fingerprint: m.DOConfig.SSHKeyFingerprint,
				},
			},
			Image: godo.DropletCreateImage{
				Slug: "coreos-stable",
			},
		}
		tags := []string{"Kubernetes-Cluster", m.Name, dropletRequest.Name}

		masterDroplet, publicIP, err := p.createDroplet(dropletRequest, tags)
		if err != nil {
			return err
		}

		// Save immediately after getting master ID
		m.DOConfig.MasterID = masterDroplet.ID
		m.MasterPublicIP = publicIP
		return nil
	})

	procedure.AddStep("building Kubernetes minion", func() error {
		// Load Nodes to see if we've already created a minion
		// TODO -- I think we can get rid of a lot of this do-unless behavior if we
		// modify Procedure to save progess on Action (which is easy to implement).
		if err := p.core.DB.Model(m).Association("Nodes").Find(&m.Nodes).Error; err != nil {
			return err
		}
		if len(m.Nodes) > 0 {
			return nil
		}

		node := &model.Node{
			KubeID: m.ID,
			Size:   m.NodeSizes[0],
		}
		return p.core.Nodes.Create(node)
	})

	// TODO repeated in provider_aws.go
	procedure.AddStep("waiting for Kubernetes", func() error {
		return action.CancellableWaitFor("Kubernetes API and first minion", 20*time.Minute, 3*time.Second, func() (bool, error) {
			nodes, err := p.core.K8S(m).Nodes().List()
			if err != nil {
				return false, nil
			}
			return len(nodes.Items) > 0, nil
		})
	})

	return procedure.Run()
}

// DeleteKube deletes a DO kubernetes cluster.
func (p *DOProvider) DeleteKube(m *model.Kube) error {
	// New Client
	client := p.newClient()
	// Step procedure
	procedure := &Procedure{
		core:  p.core,
		name:  "Delete Kube",
		model: m,
	}

	procedure.AddStep("deleting master", func() error {
		if m.DOConfig.MasterID == 0 {
			return nil
		}
		if _, err := client.Droplets.Delete(m.DOConfig.MasterID); err != nil {
			return err
		}
		m.DOConfig.MasterID = 0
		return nil
	})

	return procedure.Run()
}

// CreateNode creates a new minion on DO kubernetes cluster.
func (p *DOProvider) CreateNode(m *model.Node, action *Action) error {
	userdata, err := ioutil.ReadFile("config/providers/digitalocean/minion.yaml")
	if err != nil {
		return err
	}

	dropletRequest := &godo.DropletCreateRequest{
		Name:              m.Kube.Name + "-minion",
		Region:            m.Kube.DOConfig.Region,
		Size:              m.Size,
		PrivateNetworking: true,
		UserData:          string(userdata),
		SSHKeys: []godo.DropletCreateSSHKey{
			{
				Fingerprint: m.Kube.DOConfig.SSHKeyFingerprint,
			},
		},
		Image: godo.DropletCreateImage{
			Slug: "coreos-stable",
		},
	}
	tags := []string{"Kubernetes-Cluster", m.Kube.Name, dropletRequest.Name}

	minionDroplet, publicIP, err := p.createDroplet(dropletRequest, tags)
	if err != nil {
		return err
	}

	// Parse creation timestamp
	createdAt, err := time.Parse("2006-01-02T15:04:05Z", minionDroplet.Created)
	if err != nil {
		// TODO need to return on error here
		p.core.Log.Warnf("Could not parse Droplet creation timestamp string '%s': %s", minionDroplet.Created, err)
	}

	// Save info before waiting on IP
	m.ProviderID = strconv.Itoa(minionDroplet.ID)
	m.ProviderCreationTimestamp = createdAt
	m.ExternalIP = publicIP

	return p.core.DB.Save(m)
}

// DeleteNode deletes a minsion on a DO kubernetes cluster.
func (p *DOProvider) DeleteNode(m *model.Node) error {
	client := p.newClient()

	intID, err := strconv.Atoi(m.ProviderID)
	if err != nil {
		return err
	}
	_, err = client.Droplets.Delete(intID)
	return err
}

// CreateVolume createss a Volume on DO for Kubernetes
func (p *DOProvider) CreateVolume(m *model.Volume, action *Action) error {
	return errors.New("butt")
}

// ResizeVolume re-sizes volume on DO kubernetes cluster.
func (p *DOProvider) ResizeVolume(m *model.Volume, action *Action) error {
	return errors.New("butt")
}

// WaitForVolumeAvailable waits for DO volume to become available.
func (p *DOProvider) WaitForVolumeAvailable(m *model.Volume, action *Action) error {
	return errors.New("butt")
}

// DeleteVolume deletes a DO volume.
func (p *DOProvider) DeleteVolume(m *model.Volume) error {
	return errors.New("butt")
}

// CreateEntrypoint creates a new Load Balancer for Kubernetes in DO
func (p *DOProvider) CreateEntrypoint(m *model.Entrypoint, action *Action) error {
	return errors.New("butt")
}

//AddPortToEntrypoint adds an external entrypoint to a Loadbalancer in DO.
func (p *DOProvider) AddPortToEntrypoint(m *model.Entrypoint, lbPort int64, nodePort int64) error {
	return errors.New("butt")
}

// RemovePortFromEntrypoint removes external entrypoint from Loadbalancer on DO.
func (p *DOProvider) RemovePortFromEntrypoint(m *model.Entrypoint, lbPort int64) error {
	return errors.New("butt")
}

// DeleteEntrypoint deletes load balancer from DO.
func (p *DOProvider) DeleteEntrypoint(m *model.Entrypoint) error {
	return errors.New("butt")
}

////////////////////////////////////////////////////////////////////////////////
// Private methods                                                            //
////////////////////////////////////////////////////////////////////////////////

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

// DO Client
func (p *DOProvider) newClient() *godo.Client {
	token := &TokenSource{
		AccessToken: p.Token,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, token)
	return godo.NewClient(oauthClient)
}

// Create droplet
func (p *DOProvider) createDroplet(req *godo.DropletCreateRequest, tags []string) (*godo.Droplet, string, error) {
	client := p.newClient()

	// Create
	droplet, _, err := client.Droplets.Create(req)
	if err != nil {
		return nil, "", err
	}

	// Tag (TODO error handling needs work for atomicity / idempotence)
	for _, tag := range tags {
		input := &godo.TagResourcesRequest{
			Resources: []godo.Resource{
				{
					ID:   strconv.Itoa(droplet.ID),
					Type: godo.DropletResourceType,
				},
			},
		}
		if _, err = client.Tags.TagResources(tag, input); err != nil {
			// TODO
			p.core.Log.Warnf("Failed to tag Droplet %d with value %s", droplet.ID, tag)
			// return nil, err
		}
	}

	// NOTE we have to reload to get the IP -- even with a looping wait, the
	// droplet returned from create resp never loads the IP.
	droplet, _, err = client.Droplets.Get(droplet.ID)
	if err != nil {
		return nil, "", err
	}
	publicIP, err := droplet.PublicIPv4()
	if err != nil {
		return nil, "", err
	}

	return droplet, publicIP, nil
}