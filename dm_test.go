package dm

import (
	"github.com/stretchr/testify/suite"
	"google.golang.org/api/compute/v1"
	"testing"
)

const (
	// ProjectID is the testing project to run integration tests on
	ProjectID = "flowcloud-153422"
)

type DeploymentManagerSuite struct {
	suite.Suite

	man *Manager
}

func (s *DeploymentManagerSuite) SetupSuite() {
	s.man = NewDefault()

	s.man.Delete(ProjectID, "my-deployment")
}

func (s *DeploymentManagerSuite) TestComputeInstanceInsert() {
	const computeType = "compute.v1.instance"
	i := compute.Instance{
		Labels: map[string]string{
			"category": "test",
		},
		Zone:        "us-central1-f",
		MachineType: "https://www.googleapis.com/compute/v1/projects/" + ProjectID + "/zones/us-central1-f/machineTypes/f1-micro",
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Network: "https://www.googleapis.com/compute/v1/projects/" + ProjectID + "/global/networks/default",
				AccessConfigs: []*compute.AccessConfig{
					{
						Name: "External NAT",
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		Disks: []*compute.AttachedDisk{
			{
				DeviceName: "boot",
				Type:       "PERSISTENT",
				Boot:       true,
				AutoDelete: true,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/family/debian-9",
				},
			},
		},
	}

	err := s.man.Insert(ProjectID, "my-deployment", []Resource{
		{
			Name:       "my-instance",
			Type:       computeType,
			Properties: i,
		},
	})
	s.NoError(err)

	//err = s.man.Delete(ProjectID, "my-deployment")
	//s.NoError(err)
}

func TestDeploymentManagerSuite(t *testing.T) {
	suite.Run(t, &DeploymentManagerSuite{})
}
