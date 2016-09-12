package core

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/supergiant/supergiant/pkg/model"
	kapi "k8s.io/kubernetes/pkg/api"
)

type HeapsterMetric struct {
	Name     string `json:"name"`
	CPUUsage int64  `json:"cpuUsage"`
	RAMUsage int64  `json:"memUsage"`
}

type KubernetesInterface interface {
	ListNamespaces() ([]kapi.Namespace, error)
	ListNodes() ([]kapi.Node, error)
	ListPods() ([]kapi.Pod, error)
	ListNodeHeapsterMetrics() ([]*HeapsterMetric, error)
	ListPodHeapsterMetrics(namespace string) ([]*HeapsterMetric, error)
}

//------------------------------------------------------------------------------

// TODO
var globalK8SHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

type KubernetesClient struct {
	Kube *model.Kube
}

func (k *KubernetesClient) ListNamespaces(query string) ([]kapi.Namespace, error) {
	list := new(kapi.NamespaceList)
	if err := k.requestInto("GET", "namespaces?"+query, list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *KubernetesClient) ListNodes(query string) ([]kapi.Node, error) {
	list := new(kapi.NodeList)
	if err := k.requestInto("GET", "nodes?"+query, list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *KubernetesClient) ListPods(query string) ([]kapi.Pod, error) {
	list := new(kapi.PodList)
	if err := k.requestInto("GET", "pods?"+query, list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *KubernetesClient) ListEvents(query string) ([]kapi.Event, error) {
	list := new(kapi.EventList)
	if err := k.requestInto("GET", "events?"+query, list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *KubernetesClient) ListNodeHeapsterMetrics() ([]*HeapsterMetric, error) {
	var metrics []*HeapsterMetric
	err := k.requestInto("GET", "proxy/namespaces/kube-system/services/heapster/api/v1/model/nodes", &metrics)
	return metrics, err
}

func (k *KubernetesClient) ListPodHeapsterMetrics(namespace string) ([]*HeapsterMetric, error) {
	var metrics []*HeapsterMetric
	err := k.requestInto("GET", "proxy/namespaces/kube-system/services/heapster/api/v1/model/namespaces/"+namespace+"/pods", &metrics)
	return metrics, err
}

// Private

func (k *KubernetesClient) requestInto(method string, path string, out interface{}) error {
	url := fmt.Sprintf("https://%s/api/v1/%s", k.Kube.MasterPublicIP, path)
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(k.Kube.Username, k.Kube.Password)

	resp, err := globalK8SHTTPClient.Do(req)
	if err != nil {
		return err
	}

	if resp.Status[:2] != "20" {
		return fmt.Errorf("K8S %s error", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}
