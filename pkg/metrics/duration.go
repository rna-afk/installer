package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	clusterInstallationDurationLabels = []string{"provisioning", "infrastructure", "bootstrap",
			"api", "operators", "success"}
	//ClusterInstallationDurationJobName is the name of the metric.
	ClusterInstallationDurationJobName     = "cluster_installation_duration"
)

// Duration type is an extension of the MetricInitializer type which has the functions
// to build the required prometheus object, set value to it and populate the object with
// labels before pushing it.
type Duration struct {
	collector *prometheus.CounterVec
	value float64
	durationRegistry map[string]string
}

func (duration *Duration) objectBuilder() prometheus.Collector {
	duration.durationRegistry = make(map[string]string)
	duration.value = 1
	duration.collector = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cluster_installation_duration",
			Help: "This metric keeps track of all the different stages" +
				"of the installer create command execution and the time it" +
				"took to complete the given stage. The values are kept as labels.",
		},
		clusterInstallationDurationLabels,
	)

	return duration.collector
}

func (duration *Duration) labelSetter(key string, value string) {
	duration.durationRegistry[key] = value
}

func (duration *Duration) valueSetter(value float64) {
	duration.value = value
}

func (duration *Duration) objectPopulator() {
	duration.collector.With(prometheus.Labels(duration.durationRegistry)).Add(duration.value)
}

func (*Duration) getMetricName() string {
	return ClusterInstallationDurationJobName
}

func (duration *Duration) getCollectorObject() prometheus.Collector {
	return duration.collector
}

func (duration *Duration) cleanup() {
	duration.value = 1
	duration.objectBuilder()
}

func (duration *Duration) multiValueSetter(keyValueMap map[string]string, value float64) {
	duration.collector.With(prometheus.Labels(keyValueMap)).Add(value)
}
