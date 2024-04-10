package instancegroups

import (
	"testing"
	"time"

	"github.com/go-logr/logr"
	compute "google.golang.org/api/compute/v1"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	informerv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/ingress-gce/pkg/utils"
)

func TestNodeStatusChanged(t *testing.T) {
	testCases := []struct {
		desc   string
		mutate func(node *api_v1.Node)
		expect bool
	}{
		{
			"no change",
			func(node *api_v1.Node) {},
			false,
		},
		{
			"unSchedulable changes",
			func(node *api_v1.Node) {
				node.Spec.Unschedulable = true
			},
			true,
		},
		{
			"readiness changes",
			func(node *api_v1.Node) {
				node.Status.Conditions[0].Status = api_v1.ConditionFalse
				node.Status.Conditions[0].LastTransitionTime = meta_v1.NewTime(time.Now())
			},
			true,
		},
		{
			"new heartbeat",
			func(node *api_v1.Node) {
				node.Status.Conditions[0].LastHeartbeatTime = meta_v1.NewTime(time.Now())
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			node := testNode()
			tc.mutate(node)
			res := nodeStatusChanged(testNode(), node)
			if res != tc.expect {
				t.Fatalf("Test case %q got: %v, expected: %v", tc.desc, res, tc.expect)
			}
		})
	}
}

func testNode() *api_v1.Node {
	return &api_v1.Node{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "ns",
			Name:      "node",
			Annotations: map[string]string{
				"key1": "value1",
			},
		},
		Spec: api_v1.NodeSpec{
			Unschedulable: false,
		},
		Status: api_v1.NodeStatus{
			Conditions: []api_v1.NodeCondition{
				{
					Type:               api_v1.NodeReady,
					Status:             api_v1.ConditionTrue,
					LastHeartbeatTime:  meta_v1.NewTime(time.Date(2000, 01, 1, 1, 0, 0, 0, time.UTC)),
					LastTransitionTime: meta_v1.NewTime(time.Date(2000, 01, 1, 1, 0, 0, 0, time.UTC)),
				},
			},
		},
	}
}

func TestSync(t *testing.T) {
	config := &ControllerConfig{}
	informer := informerv1.NewNodeInformer(fake.NewSimpleClientset(), 2*time.Second, utils.NewNamespaceIndexer())
	config.NodeInformer = informer
	fakeManager := &IGManagerFake{}
	config.IGManager = fakeManager
	config.HasSynced = func() bool {
		return true
	}

	controller := NewController(config, logr.Logger{})

	channel := make(chan struct{})
	go informer.Run(channel)
	go controller.Run()

	firstNode := testNode()
	secondNode := testNode()
	secondNode.Name = "n2"

	informer.GetIndexer().Add(node1)
	informer.GetIndexer().Add(node2)
	node1.Annotations["key"] = "true"
	node2.Annotations["key"] = "true"
	informer.GetIndexer().Update(node1)
	informer.GetIndexer().Update(node2)
	time.Sleep(5 * time.Second)
	informer.GetIndexer().Update(node2)
	if len(fakeManager.syncedNodes) != 1 {
		t.Errorf("synced too many times %d, %v", len(fakeManager.syncedNodes), fakeManager.syncedNodes)
	}
}

type IGManagerFake struct {
	syncedNodes [][]string
}

func (igmf *IGManagerFake) Sync(nodeNames []string) error {
	igmf.syncedNodes = append(igmf.syncedNodes, nodeNames)
	return nil
}

func (igmf *IGManagerFake) EnsureInstanceGroupsAndPorts(name string, ports []int64) ([]*compute.InstanceGroup, error) {
	return []*compute.InstanceGroup{}, nil
}

func (igmf *IGManagerFake) DeleteInstanceGroup(name string) error {
	return nil
}

func (igmf *IGManagerFake) Get(name, zone string) (*compute.InstanceGroup, error) {
	ig := compute.InstanceGroup{}
	return &ig, nil
}

func (igmf *IGManagerFake) List() ([]string, error) {
	return []string{}, nil
}
