package main

import (
	"strings"
	"time"
)

type clusterComponentState int

const (
	clusterPending clusterComponentState = iota
	clusterRunning
	clusterDone
)

type clusterComponent struct {
	Name        string
	Description string
	State       clusterComponentState
	ChangedAt   time.Time
}

type clusterDiagram struct {
	runID      int
	components []clusterComponent
	selected   int
}

func newClusterDiagram(runID int) *clusterDiagram {
	return &clusterDiagram{
		runID: runID,
		components: []clusterComponent{
			{Name: "Network", Description: "VPC, subnets, and security groups"},
			{Name: "DNS", Description: "Cluster DNS records and zones"},
			{Name: "Load balancers", Description: "API and ingress load balancers"},
			{Name: "Bootstrap machine", Description: "Temporary bootstrap host"},
			{Name: "Temporary control plane", Description: "Bootstrap-hosted Kubernetes API and etcd"},
			{Name: "Control plane machines", Description: "Permanent control plane hosts"},
			{Name: "Worker machines", Description: "Compute machines"},
			{Name: "Bootstrap complete", Description: "Bootstrap configmap reports completion"},
			{Name: "Cluster operators", Description: "Operators become available and stable"},
			{Name: "Install complete", Description: "Cluster initialization finished"},
		},
	}
}

func (d *clusterDiagram) set(index int, state clusterComponentState) {
	if index < 0 || index >= len(d.components) {
		return
	}
	component := &d.components[index]
	if state <= component.State {
		return
	}
	component.State = state
	component.ChangedAt = time.Now()
}

func (d *clusterDiagram) advance(index int) {
	for i := 0; i < index; i++ {
		d.set(i, clusterDone)
	}
	d.set(index, clusterRunning)
}

func (d *clusterDiagram) complete(index int) {
	d.set(index, clusterDone)
}

func (d *clusterDiagram) completeAll() {
	for i := range d.components {
		d.set(i, clusterDone)
	}
}

func (d *clusterDiagram) visibleComponents() []int {
	visible := make([]int, 0, len(d.components))
	for i, component := range d.components {
		if component.State != clusterPending {
			visible = append(visible, i)
		}
	}
	return visible
}

func (d *clusterDiagram) ensureVisibleSelection() {
	if d.selected >= 0 && d.selected < len(d.components) && d.components[d.selected].State != clusterPending {
		return
	}
	if visible := d.visibleComponents(); len(visible) > 0 {
		d.selected = visible[0]
	}
}

func (d *clusterDiagram) moveSelection(delta int) {
	visible := d.visibleComponents()
	if len(visible) == 0 {
		return
	}

	position := -1
	for i, index := range visible {
		if index == d.selected {
			position = i
			break
		}
	}
	if position < 0 {
		position = 0
	} else {
		position += delta
	}
	if position < 0 {
		position = 0
	}
	if position >= len(visible) {
		position = len(visible) - 1
	}
	d.selected = visible[position]
}

func (d *clusterDiagram) observe(line string) {
	lower := strings.ToLower(line)
	has := func(values ...string) bool {
		for _, value := range values {
			if strings.Contains(lower, value) {
				return true
			}
		}
		return false
	}

	switch {
	case has("install complete", "cluster is initialized"):
		d.completeAll()
	case has("all cluster operators have completed progressing", "cluster operators stable"):
		d.advance(8)
		d.complete(8)
	case has("bootstrap status: complete", "bootstrap complete", "safe to remove the bootstrap resources"):
		d.advance(7)
		d.complete(7)
	case has("worker machine", "compute machine", "worker machines"):
		d.advance(6)
	case has("control plane machine", "control-plane machine", "control plane hosts", "master machine"):
		d.advance(5)
	case has("waiting for the kubernetes api", "kubernetes api", "temporary control plane", "api up"):
		d.advance(4)
	case has("bootstrap machine", "creating bootstrap", "bootstrap host"):
		d.advance(3)
	case has("load balancer", "loadbalancer", "elastic load balancer", " nlb", " elb"):
		d.advance(2)
	case has("dns", "route 53", "route53", "hosted zone"):
		d.advance(1)
	case has("creating infrastructure resources", "network", "vpc", "subnet", "security group"):
		d.advance(0)
	}
	d.ensureVisibleSelection()
}
