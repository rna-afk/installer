package gatherer

import (
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/openshift/installer/pkg/metrics/builder"
	"github.com/openshift/installer/pkg/metrics/pushclient"
)

var (
	// metricRegistry holds all the MetricBuilder objects for all the metrics that have been designed
	// and ready for extraction from the installer.
	metricRegistry = make(map[string]*builder.MetricBuilder)

	// CreateCommandMetricName is the map of the targets to the respective metric names. Installer can
	// look this up for the correct metric name given a target.
	CreateCommandMetricName = map[string]string{"cluster": createMetricName,
		"ignition-configs": ignitionMetricName,
		"install-config":   installConfigMetricName,
		"manifests":        manifestsMetricName}

	// CurrentInvocationContext is used to store the current Invocation metric name used for building context.
	// Invocation data information is gathered from different parts of the installer and it gets hard for the
	// installer to keep track of the current command and target that is being run. Hence, the current invocation
	// metric name is stored here in this variable.
	CurrentInvocationContext string

	// enableMetrics holds the user preference for sending metrics. If the user chooses not to send, this will hold
	// the value false.
	enableMetrics bool

	// prometheusURL is the standard URL to send metrics to. The user can override this information to any URL they choose.
	prometheusURL = "https://localhost:9092/"

	// pushClient holds the URL, MetricName and the http client information to send metrics.
	pushClient pushclient.PushClient
)

// Initialize holds all the initial setup information like creating all the metrics that are allowed to
// be extracted, checking for user input about disabling metrics, URL and for creating the Push Client.
// Must be run before the metircs are collected.
func Initialize() {
	enableMetrics = false
	if value, ok := os.LookupEnv("OPENSHIFT_INSTALL_METRICS_ENDPOINT"); ok {
		prometheusURL = value
		enableMetrics = true
	}
	pushClient = pushclient.PushClient{URL: prometheusURL, Client: &http.Client{}, JobName: "openshift_installer_metrics"}
}

// AddLabelValue adds a label key/value pair for a given metric. It keeps track of all the
// labels for all metrics if added through this function.
func AddLabelValue(metricName string, labelName string, labelValue string) {
	if enableMetrics {
		if _, found := metricRegistry[metricName]; !found {
			opts := optsRegistry[metricName]
			metricBuilder, err := builder.NewMetricBuilder(*opts, 0, nil)
			if err == nil {
				metricRegistry[metricName] = metricBuilder
			}
		}
		metricRegistry[metricName].AddLabelValue(labelName, labelValue)
	}
}

// SetValue sets the value to the specific metric before sending to prometheus.
func SetValue(metricName string, value float64) {
	if enableMetrics {
		if _, found := metricRegistry[metricName]; !found {
			opts := optsRegistry[metricName]
			metricBuilder, err := builder.NewMetricBuilder(*opts, 0, nil)
			if err == nil {
				metricRegistry[metricName] = metricBuilder
			}
		}
		metricRegistry[metricName].SetValue(value)
	}
}

// Push is a driver function that sends the metrics created with all the information and pushes it
// to Prometheus through the Push Client.
func Push(metricName string) {
	if enableMetrics {
		if _, found := metricRegistry[metricName]; found {
			collector, err := metricRegistry[metricName].PromCollector()
			if err == nil {
				pushClient.Push(collector)
			}

		}
	}
}

// PushAll function takes all the metrics whose label values are set and pushes them to Prometheus.
func PushAll() {
	if enableMetrics {
		var collectors []prometheus.Collector
		for _, value := range metricRegistry {
			collector, err := value.PromCollector()
			if err == nil {
				collectors = append(collectors, collector)
			}

		}
		pushClient.Push(collectors...)
	}
}
