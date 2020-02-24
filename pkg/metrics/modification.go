package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	clusterInstallationModificationLabels = []string{"asset"}
	// ClusterInstallationModificationJobName is the name of the metric that must be shown as is in
	// the prometheus database.
	ClusterInstallationModificationJobName     = "cluster_installation_modification"
)

// Modification type is an extension of the MetricInitializer type which has the functions
// to build the required prometheus object, set value to it and populate the object with
// labels before pushing it.
type Modification struct {
	collector *prometheus.CounterVec
	value float64
}

func (modification *Modification) objectBuilder() prometheus.Collector {
	modification.collector = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cluster_installation_modification",
			Help: "This metric keeps track of all the assets that were modified " +
				"by the user before the invocation of the create command in the installer",
		},
		clusterInstallationModificationLabels,
	)

	return modification.collector
}

func (modification *Modification) labelSetter(key string, value string) {
	modification.collector.With(prometheus.Labels{key: value}).Inc()
}

func (modification *Modification) valueSetter(value float64) {
}

func (*Modification) objectPopulator() {
	
}

func (modification *Modification) getCollectorObject() prometheus.Collector {
	return modification.collector
}

func (*Modification) getMetricName() string {
	return ClusterInstallationModificationJobName
}

func (modification *Modification) cleanup() {
	modification.value = 0
	modification.objectBuilder()
}

func (modification *Modification) multiValueSetter(keyValueMap map[string]string, value float64) {
	modification.collector.With(prometheus.Labels(keyValueMap)).Add(value)
}
