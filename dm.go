package dm

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-yaml/yaml"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	gcpdm "google.golang.org/api/deploymentmanager/v2beta"
	"net/http"
	"os"
	"time"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)

	log.Warn()
}

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
	log.WithFields(log.Fields{
		"ProjectID": project,
		"DeploymentName": name,
		"Resources": resources,
	}).Trace("Inserting new Deployment")

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

	log.Debug("Generated configuration:", string(res))

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
	log.WithFields(log.Fields{
		"ProjectID": project,
		"DeploymentName": name,
		"Resources": resources,
	}).Trace("Updating an existing Deployment")

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
	log.WithFields(log.Fields{
		"ProjectID": project,
		"DeploymentName": name,
	}).Trace("Deleting a Deployment")

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

		log.WithFields(log.Fields{
			"ProjectID": project,
			"OperationType": op.OperationType,
			"Status": opStatus.Status,
		}).Infof("Operation update")
		if opStatus.Status == "DONE" {
			if opStatus.Error != nil && len(opStatus.Error.Errors) > 0 {
				return errors.New(opStatus.Error.Errors[0].Message)
			}
			break
		}

		time.Sleep(time.Second * 6)
	}

	return nil
}

// remarshal takes a slice of resources, marshals them to JSON to omit empty
// fields and remove any formatting that could affect the YAML configuration.
// It remarshalls the JSON back into a struct and finally into the resulting YAML
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
