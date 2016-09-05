package ui

import (
	"net/http"

	"github.com/supergiant/supergiant/pkg/client"
	"github.com/supergiant/supergiant/pkg/model"
)

func NewKube(sg *client.Client, w http.ResponseWriter, r *http.Request) error {

	// TODO we shouldn't use a map for the model, because attr ordering is screwed
	// up. But it's difficult to initialize blank values with omitemptys (which
	// are needed for certain things), so we should probably have special structs.

	return renderTemplate(w, "new", map[string]interface{}{
		"title":      "Kubes",
		"formAction": "/ui/kubes",
		"formMethod": "POST",
		"model": map[string]interface{}{
			"cloud_account_id": nil,
			"name":             "",
			"master_node_size": "m4.large",
			"node_sizes": []string{
				"m4.large",
				"m4.xlarge",
				"m4.2xlarge",
				"m4.4xlarge",
			},
			"aws_config": map[string]interface{}{
				"region":                 "us-east-1",
				"availability_zone":      "us-east-1b",
				"vpc_ip_range":           "172.20.0.0/16",
				"public_subnet_ip_range": "172.20.0.0/24",
				"master_private_ip":      "172.20.0.9",
			},
		},
	})
}

func CreateKube(sg *client.Client, w http.ResponseWriter, r *http.Request) error {
	m := new(model.Kube)
	if err := unmarshalFormInto(r, m); err != nil {
		return err
	}
	if err := sg.Kubes.Create(m); err != nil {
		return renderTemplate(w, "new", map[string]interface{}{
			"title":      "Kubes",
			"formAction": "/ui/kubes",
			"formMethod": "POST",
			"model":      m,
			"error":      err.Error(),
		})
	}
	http.Redirect(w, r, "/ui/kubes", http.StatusFound)
	return nil
}

func ListKubes(sg *client.Client, w http.ResponseWriter, r *http.Request) error {
	fields := []map[string]interface{}{
		{
			"title": "Name",
			"type":  "field_value",
			"field": "name",
		},
		{
			"title": "Master Size",
			"type":  "field_value",
			"field": "master_node_size",
		},
	}
	return renderTemplate(w, "index", map[string]interface{}{
		"title":       "Kubes",
		"uiBasePath":  "/ui/kubes",
		"apiListPath": "/api/v0/kubes",
		"fields":      fields,
		"showNewLink": true,
		"batchActionPaths": map[string]string{
			"Delete": "/delete",
		},
	})
}

func GetKube(sg *client.Client, w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r)
	if err != nil {
		return err
	}
	item := new(model.Kube)
	if err := sg.Kubes.Get(id, item); err != nil {
		return err
	}
	return renderTemplate(w, "show", map[string]interface{}{
		"title": "Kubes",
		"model": item,
	})
}

func DeleteKube(sg *client.Client, w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r)
	if err != nil {
		return err
	}
	item := new(model.Kube)
	if err := sg.Kubes.Delete(id, item); err != nil {
		return err
	}
	// http.Redirect(w, r, "/ui/kubes", http.StatusFound)
	return nil
}
