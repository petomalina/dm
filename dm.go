package dm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-yaml/yaml"
	"golang.org/x/oauth2/google"
	gcpdm "google.golang.org/api/deploymentmanager/v2beta"
	"net/http"
	"time"
)

type Manager struct {
	dms *gcpdm.Service
}

func New(client *http.Client) *Manager {
	dms, err := gcpdm.New(client)
	if err != nil {
		panic(err)
	}

	return &Manager{
		dms: dms,
	}
}

func NewDefault() *Manager {
	client, err := google.DefaultClient(
		context.Background(),
		gcpdm.CloudPlatformScope,
	)
	if err != nil {
		panic(err)
	}

	return New(client)
}

type Resources struct {
	Resources []Resource `json:"resources"`
}

type Resource struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Properties interface{} `json:"properties"`
}

func (m *Manager) Insert(project, name string, resources []Resource) error {
	wrappedResources := Resources{
		Resources: resources,
	}

	// marshal to json first
	resJson, err := json.Marshal(wrappedResources)
	if err != nil {
		return err
	}

	// unmarshal into the resource struct to perform no-tag dereference
	jsonStruct := Resources{}
	_ = json.Unmarshal(resJson, &jsonStruct)

	// marshal back into yaml
	res, err := yaml.Marshal(jsonStruct)
	if err != nil {
		return err
	}

	fmt.Println(string(res))

	call := m.dms.Deployments.Insert(project, &gcpdm.Deployment{
		Name: name,
		Target: &gcpdm.TargetConfiguration{
			Config: &gcpdm.ConfigFile{
				Content: string(res),
			},
		},
	})

	op, err := call.Do()
	if err != nil {
		return err
	}

	return m.waitUntilDone(project, op)
}

func (m *Manager) Update(project, name string, resources []Resource) error {
	res, err := remarshal(resources)
	if err != nil {
		return err
	}

	call := m.dms.Deployments.Update(project, name, &gcpdm.Deployment{
		Name: name,
		Target: &gcpdm.TargetConfiguration{
			Config: &gcpdm.ConfigFile{
				Content: string(res),
			},
		},
	})

	op, err := call.Do()
	if err != nil {
		return err
	}

	return m.waitUntilDone(project, op)
}

func (m *Manager) Delete(project, name string) error {
	call := m.dms.Deployments.Delete(project, name)

	op, err := call.Do()
	if err != nil {
		return err
	}

	return m.waitUntilDone(project, op)
}

func (m *Manager) waitUntilDone(project string, op *gcpdm.Operation) error {
	for {
		opStatus, err := m.dms.Operations.Get(project, op.Name).Do()
		if err != nil {
			return err
		}

		fmt.Printf("Pending [%s] with status [%s]\n", op.OperationType, opStatus.Status)
		if opStatus.Status == "DONE" {
			if opStatus.Error != nil && len(opStatus.Error.Errors) > 0 {
				return errors.New(opStatus.Error.Errors[0].Message)
			}
			break
		}

		time.Sleep(time.Second * 10)
	}

	return nil
}

func remarshal(res []Resource) ([]byte, error) {
	wrappedResources := Resources{
		Resources: res,
	}

	// marshal to json first
	resJson, err := json.Marshal(wrappedResources)
	if err != nil {
		return nil, err
	}

	// unmarshal into the resource struct to perform no-tag dereference
	jsonStruct := Resources{}
	_ = json.Unmarshal(resJson, &jsonStruct)

	// marshal back into yaml
	return yaml.Marshal(jsonStruct)
}
