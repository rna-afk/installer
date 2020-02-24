package metrics

import (
	"math"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/openshift/installer/pkg/version"
)

var (

	clusterInstallationInvocationLabels = []string{"command", "target", "result", "returnCode",
			"platform", "os", "version"}

	// ClusterInstallationInvocationJobName is the name of the metric that must be sent to
	// prometheus.
	ClusterInstallationInvocationJobName   = "cluster_installation_invocation"
)

// Invocation type is an extension of the MetricInitializer type which has the functions
// to build the required prometheus object, set value to it and populate the object with
// labels before pushing it.
type Invocation struct {
	collector *prometheus.HistogramVec
	value float64
	invocationRegistry map[string]string
}

func (invocation *Invocation) objectBuilder() prometheus.Collector {
	invocation.invocationRegistry = make(map[string]string)
	maxBucketValue := 60.0
	invocation.collector = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "cluster_installation_invocation",
			Help: "This metric keeps track of the count of the number of times " +
				"the user ran a specific sequence of commands in the given OS " +
				"took the time that is lesser than or equal to the value in the duration label.",
			Buckets: prometheus.LinearBuckets(0, BucketSize, int(math.Ceil(maxBucketValue/BucketSize))+1),
		},
		clusterInstallationInvocationLabels,
	)

	return invocation.collector
}

func (invocation *Invocation) labelSetter(key string, value string) {
	invocation.invocationRegistry[key] = value
}

func (invocation *Invocation) valueSetter(value float64) {
	invocation.value = value
}

func (invocation *Invocation) multiValueSetter(keyValueMap map[string]string, value float64) {
	invocation.collector.With(prometheus.Labels(keyValueMap)).Observe(value)
}

func (invocation *Invocation) objectPopulator() {
	versionNumber, _ := version.Version()
	invocation.invocationRegistry["version"] = versionNumber
	invocation.invocationRegistry["os"] = runtime.GOOS
	invocation.collector.With(prometheus.Labels(invocation.invocationRegistry)).Observe(invocation.value)
}

func (*Invocation) getMetricName() string {
	return ClusterInstallationInvocationJobName
}

func (invocation *Invocation) cleanup() {
	invocation.value = 0
	invocation.objectBuilder()
}

func (invocation *Invocation) getCollectorObject() prometheus.Collector{
	return invocation.collector
}
