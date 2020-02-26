package metrics

import (
	"math"
	"net/http"
	"os"
	"runtime"

	"github.com/openshift/installer/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
)

// aggClient is a custom HTTP client for use with
// WeaveWorks Prometheus Aggregation PushGateway
// We will embed http.Client and essentially
// "override" the Do function
type aggClient struct {
	// embed a stdlib http.Client
	*http.Client
}

const aggregationGatewayURL = "http://localhost:9092/api/ui/metrics"

// Do is a custom written client to send data to the prometheus aggregate
// gateway. It creates a new client and pushes the body that was passed to this function.
// https://godoc.org/github.com/prometheus/client_golang/prometheus/push#HTTPDoer
func (a *aggClient) Do(req *http.Request) (resp *http.Response, err error) {
	newReq, _ := http.NewRequest("PUT", aggregationGatewayURL, req.Body)
	return a.Client.Do(newReq)
}

// MetricBuilder is the basic metric type that must be used by any potential metric added to
// the system. Any metric added must have the following functions which would be used in the
// workflow.
type MetricBuilder struct {
	labels         []string
	labelKeyValues map[string]string
	desc           string
	name           string
	value          float64
	buckets        []float64
	metricType     string
}

func (m MetricBuilder) build() prometheus.Collector {
	switch m.metricType {
	case "modification":
		return getCounter(m)
	case "invocation":
		return getHistogram(m)
	case "duration":
		return getHistogram(m)
	default:
		return nil
	}
}

var startTimeEpoch int64

var url = "http://localhost:9092"

var duration string
var isEnabled bool

// Platform is a variable that can be used to store the current platform information
// and is automatically picked up by the metrics before pushing.
var Platform string

var (
	// BucketSize is the step size difference between each bucket.
	BucketSize = 5.0
	// INVOCATION METRIC

	// ClusterInstallationCreateJobName is the variable that stores the metric name for the create cluster details.
	ClusterInstallationCreateJobName = "cluster_installation_create"
	// ClusterInstallationIgnitionJobName is the variable that stores the metric name for the create ignition details.
	ClusterInstallationIgnitionJobName = "cluster_installation_ignition"
	// ClusterInstallationManifestsJobName is the variable that stores the metric name for the create manifests details.
	ClusterInstallationManifestsJobName = "cluster_installation_manifests"
	// ClusterInstallationInstallConfigJobName is the variable that stores the metric name for the create install configs details.
	ClusterInstallationInstallConfigJobName = "cluster_installation_install_config"
	// ClusterInstallationWaitforJobName is the variable that stores the metric name for the waitfor command details.
	ClusterInstallationWaitforJobName = "cluster_installation_waitfor"
	// ClusterInstallationGatherJobName is the variable that stores the metric name for the gather command configs details.
	ClusterInstallationGatherJobName = "cluster_installation_gather"
	// ClusterInstallationDestroyJobName is the variable that stores the metric name for the destroy command configs details.
	ClusterInstallationDestroyJobName = "cluster_installation_destroy"

	// DURATION METRIC

	// ClusterInstallationDurationProvisioningJobName is the variable that stores the metric name for the duration provisioning details.
	ClusterInstallationDurationProvisioningJobName = "cluster_installation_duration_provisioning"
	// ClusterInstallationDurationInfrastructureJobName is the variable that stores the metric name for the duration infrastructure details.
	ClusterInstallationDurationInfrastructureJobName = "cluster_installation_duration_infrastructure"
	// ClusterInstallationDurationOperatorsJobName is the variable that stores the metric name for the duration operators details.
	ClusterInstallationDurationOperatorsJobName = "cluster_installation_duration_operators"
	// ClusterInstallationDurationBootstrapJobName is the variable that stores the metric name for the duration bootstrap details.
	ClusterInstallationDurationBootstrapJobName = "cluster_installation_duration_bootstrap"
	// ClusterInstallationDurationAPIJobName is the variable that stores the metric name for the duration API details.
	ClusterInstallationDurationAPIJobName = "cluster_installation_duration_api"

	// MODIFICATION METRIC

	// ClusterInstallationModificationConfigJobName is the variable that stores the metric name for the modification of config manifests details.
	ClusterInstallationModificationConfigJobName = "cluster_installation_modification_config_manifest"
	// ClusterInstallationModificationDNSJobName is the variable that stores the metric name for the modification of DNS manifests details.
	ClusterInstallationModificationDNSJobName = "cluster_installation_modification_dns_manifest"
	// ClusterInstallationModificationNetworkJobName is the variable that stores the metric name for the modification of network manifests details.
	ClusterInstallationModificationNetworkJobName = "cluster_installation_modification_network_manifest"
	// ClusterInstallationModificationSchedulerJobName is the variable that stores the metric name for the modification of scheduler manifests details.
	ClusterInstallationModificationSchedulerJobName = "cluster_installation_modification_scheduler_manifest"
	// ClusterInstallationModificationCVOJobName is the variable that stores the metric name for the modification of CVO manifests details.
	ClusterInstallationModificationCVOJobName = "cluster_installation_modification_cvo_manifest"
	// ClusterInstallationModificationEtcdJobName is the variable that stores the metric name for the modification of etcd manifests details.
	ClusterInstallationModificationEtcdJobName = "cluster_installation_modification_etcd_manifest"
	// ClusterInstallationModificationMachineConfigJobName is the variable that stores the metric name for the modification of machine config manifests details.
	ClusterInstallationModificationMachineConfigJobName = "cluster_installation_modification_machineconfig_manifest"
	// ClusterInstallationModificationCaJobName is the variable that stores the metric name for the modification of ca manifests details.
	ClusterInstallationModificationCaJobName = "cluster_installation_modification_ca_manifest"
	// ClusterInstallationModificationPullSecretJobName is the variable that stores the metric name for the modification of pull secret manifests details.
	ClusterInstallationModificationPullSecretJobName = "cluster_installation_modification_pullsecret_manifest"
	// ClusterInstallationModificationBootstrapJobName is the variable that stores the metric name for the modification of bootstrap manifests details.
	ClusterInstallationModificationBootstrapJobName = "cluster_installation_modification_bootstrap_ignition"
	// ClusterInstallationModificationMasterJobName is the variable that stores the metric name for the modification of master manifests details.
	ClusterInstallationModificationMasterJobName = "cluster_installation_modification_master_ignition"
	// ClusterInstallationModificationWorkerJobName is the variable that stores the metric name for the modification of worker manifests details.
	ClusterInstallationModificationWorkerJobName = "cluster_installation_modification_worker_manifest"

	// FileCategories is a map of asset names to the categories of the metrics that it belongs to.
	FileCategories = make(map[string]string)
	metricsBuilt   = make(map[string][]prometheus.Collector)
	metricRegistry = make(map[string]MetricBuilder)

	// CreateCommandMetricName is the map of the targets to the respective metric names.
	CreateCommandMetricName = make(map[string]string)

	//CurrentInvocationContext is used to store the current Invocation metric name used for building context.
	CurrentInvocationContext string
)

func getCounter(m MetricBuilder) prometheus.Collector {
	collector := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:        m.name,
			Help:        m.desc,
			ConstLabels: m.labelKeyValues,
		},
	)

	collector.Add(m.value)

	return collector
}

func getHistogram(m MetricBuilder) prometheus.Collector {
	mapOfLabels := m.labelKeyValues
	versionNo, _ := version.Version()
	mapOfLabels["os"] = runtime.GOOS
	mapOfLabels["version"] = versionNo
	mapOfLabels["platform"] = Platform

	collector := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:        m.name,
			Help:        m.desc,
			Buckets:     m.buckets,
			ConstLabels: m.labelKeyValues,
		},
	)

	collector.Observe(m.value)

	return collector
}

func getNewDurationBuilder(metricName string, metricShortDesc string) MetricBuilder {
	return MetricBuilder{
		labels: []string{"provisioning", "infrastructure", "bootstrap",
			"api", "operators", "success"},
		labelKeyValues: make(map[string]string),
		desc: "This metric keeps track of the " + metricShortDesc + " stage" +
			"of the installer create command execution and the time it" +
			"took to complete the given stage. The values are kept as labels",
		name:       metricName,
		value:      0,
		buckets:    nil,
		metricType: "duration",
	}
}

// Initialize must be called before using the pushing mechanism.
// This function starts a timer and initializes the values of the list of label values for a given
// job, the object to be created for a given job name and also the map that holds the key/value
// labels to be used over the course of the installer to collect data.
func Initialize() {
	isEnabled = true

	if _, ok := os.LookupEnv("OPENSHIFT_INSTALL_DISABLE_METRICS"); ok {
		isEnabled = false
	}

	if value, ok := os.LookupEnv("OPENSHIFT_INSTALL_METRICS_ENDPOINT"); ok {
		url = value
	}

	initializeInvocationMetrics()
	initializeDurationMetrics()
	initializeModificationCategories()
	initializeModificationMetrics()

	initializeCreateCommandMetricName()
}

func initializeCreateCommandMetricName() {
	CreateCommandMetricName["cluster"] = ClusterInstallationCreateJobName
	CreateCommandMetricName["ignition-configs"] = ClusterInstallationIgnitionJobName
	CreateCommandMetricName["install-config"] = ClusterInstallationInstallConfigJobName
	CreateCommandMetricName["manifests"] = ClusterInstallationManifestsJobName
}

func getNewInvocationMetrics(jobName string, desc string, initialBucketValue float64, maxBucketValue float64, stepSize float64) MetricBuilder {
	return MetricBuilder{
		labels:         []string{"result", "returnCode", "platform", "os", "version"},
		labelKeyValues: make(map[string]string),
		desc: "This metric keeps track of the count of the number of times " +
			"the user ran " + desc + " command in the given OS " +
			"took the time that is lesser than or equal to the value in the duration label.",
		name:       jobName,
		value:      0,
		buckets:    prometheus.LinearBuckets(initialBucketValue, stepSize, int(math.Ceil(maxBucketValue/stepSize))+1),
		metricType: "invocation",
	}
}

func initializeInvocationMetrics() {
	metricRegistry[ClusterInstallationCreateJobName] =
		getNewInvocationMetrics(ClusterInstallationCreateJobName, "create cluster command", 15, 60, 5)

	metricRegistry[ClusterInstallationManifestsJobName] =
		getNewInvocationMetrics(ClusterInstallationManifestsJobName, "create manifests command", 15, 60, 5)

	metricRegistry[ClusterInstallationIgnitionJobName] =
		getNewInvocationMetrics(ClusterInstallationIgnitionJobName, "create ignition configs command", 15, 60, 5)

	metricRegistry[ClusterInstallationInstallConfigJobName] =
		getNewInvocationMetrics(ClusterInstallationInstallConfigJobName, "create install configs command", 15, 60, 5)

	metricRegistry[ClusterInstallationDestroyJobName] =
		getNewInvocationMetrics(ClusterInstallationDestroyJobName, "destroy command", 5, 30, 5)

	metricRegistry[ClusterInstallationWaitforJobName] =
		getNewInvocationMetrics(ClusterInstallationWaitforJobName, "waitfor command", 5, 30, 5)

	metricRegistry[ClusterInstallationGatherJobName] =
		getNewInvocationMetrics(ClusterInstallationGatherJobName, "gather command", 15, 60, 5)

}

func initializeModificationCategories() {
	FileCategories["Cloud Provider Config"] = ClusterInstallationModificationConfigJobName
	FileCategories["Infrastructure Config"] = ClusterInstallationModificationConfigJobName
	FileCategories["KubeCloudConfig"] = ClusterInstallationModificationConfigJobName
	FileCategories["KubeSystemConfigmapRootCA"] = ClusterInstallationModificationConfigJobName

	FileCategories["Ingress Config"] = ClusterInstallationModificationDNSJobName
	FileCategories["DNS Config"] = ClusterInstallationModificationDNSJobName

	FileCategories["Proxy Config"] = ClusterInstallationModificationNetworkJobName
	FileCategories["Network Config"] = ClusterInstallationModificationNetworkJobName

	FileCategories["Scheduler Config"] = ClusterInstallationModificationSchedulerJobName

	FileCategories["CVOOverrides"] = ClusterInstallationModificationSchedulerJobName

	FileCategories["EtcdCAConfigMap"] = ClusterInstallationModificationEtcdJobName
	FileCategories["EtcdMetricServingCAConfigMap"] = ClusterInstallationModificationEtcdJobName
	FileCategories["EtcdServingCAConfigMap"] = ClusterInstallationModificationEtcdJobName

	FileCategories["MachineConfigServerTLSSecret"] = ClusterInstallationModificationMachineConfigJobName
	FileCategories["OpenshiftMachineConfigOperator"] = ClusterInstallationModificationMachineConfigJobName

	FileCategories["MachineConfigServerTLSSecret"] = ClusterInstallationModificationCaJobName

	FileCategories["OpenshiftConfigSecretPullSecret"] = ClusterInstallationModificationPullSecretJobName

	FileCategories["Bootstrap Ignition Config"] = ClusterInstallationModificationBootstrapJobName

	FileCategories["Master Ignition Config"] = ClusterInstallationModificationMasterJobName
	FileCategories["Master Machines"] = ClusterInstallationModificationMasterJobName

	FileCategories["Worker Ignition Config"] = ClusterInstallationModificationWorkerJobName
	FileCategories["Worker Machines"] = ClusterInstallationModificationWorkerJobName

}

func initializeDurationMetrics() {
	metricRegistry[ClusterInstallationDurationProvisioningJobName] =
		getNewDurationBuilder(ClusterInstallationDurationProvisioningJobName, "provisioning")

	metricRegistry[ClusterInstallationDurationBootstrapJobName] =
		getNewDurationBuilder(ClusterInstallationDurationBootstrapJobName, "bootstrap")

	metricRegistry[ClusterInstallationDurationOperatorsJobName] =
		getNewDurationBuilder(ClusterInstallationDurationOperatorsJobName, "operators")

	metricRegistry[ClusterInstallationDurationInfrastructureJobName] =
		getNewDurationBuilder(ClusterInstallationDurationInfrastructureJobName, "infrastructure")

	metricRegistry[ClusterInstallationDurationAPIJobName] =
		getNewDurationBuilder(ClusterInstallationDurationAPIJobName, "api")
}

func getNewModificationMetrics(metricName string, desc string) MetricBuilder {
	return MetricBuilder{
		labels:         []string{"result"},
		labelKeyValues: make(map[string]string),
		desc: "This metric keeps track of all the assets in the " + desc + " category that were modified " +
			"by the user before the invocation of the create command in the installer",
		name:       metricName,
		value:      1,
		buckets:    nil,
		metricType: "modification",
	}
}

func initializeModificationMetrics() {
	metricRegistry[ClusterInstallationModificationBootstrapJobName] =
		getNewModificationMetrics(ClusterInstallationModificationBootstrapJobName, "bootstrap")

	metricRegistry[ClusterInstallationModificationCVOJobName] =
		getNewModificationMetrics(ClusterInstallationModificationCVOJobName, "CVO")

	metricRegistry[ClusterInstallationModificationCaJobName] =
		getNewModificationMetrics(ClusterInstallationModificationCaJobName, "CA")

	metricRegistry[ClusterInstallationModificationConfigJobName] =
		getNewModificationMetrics(ClusterInstallationModificationConfigJobName, "configs")

	metricRegistry[ClusterInstallationModificationDNSJobName] =
		getNewModificationMetrics(ClusterInstallationModificationDNSJobName, "DNS")

	metricRegistry[ClusterInstallationModificationEtcdJobName] =
		getNewModificationMetrics(ClusterInstallationModificationEtcdJobName, "Etcd")

	metricRegistry[ClusterInstallationModificationMachineConfigJobName] =
		getNewModificationMetrics(ClusterInstallationModificationMachineConfigJobName, "machine config")

	metricRegistry[ClusterInstallationModificationMasterJobName] =
		getNewModificationMetrics(ClusterInstallationModificationMasterJobName, "master")

	metricRegistry[ClusterInstallationModificationNetworkJobName] =
		getNewModificationMetrics(ClusterInstallationModificationNetworkJobName, "network")

	metricRegistry[ClusterInstallationModificationPullSecretJobName] =
		getNewModificationMetrics(ClusterInstallationModificationPullSecretJobName, "pull secrets")

	metricRegistry[ClusterInstallationModificationSchedulerJobName] =
		getNewModificationMetrics(ClusterInstallationModificationSchedulerJobName, "scheduler")

	metricRegistry[ClusterInstallationModificationWorkerJobName] =
		getNewModificationMetrics(ClusterInstallationModificationWorkerJobName, "workers")
}

// AddLabelValue adds a label key/value pair for a given metric. It keeps track of all the
// labels for all metrics if added through this function.
func AddLabelValue(metricName string, labelName string, labelValue string) {
	if isEnabled {
		metricRegistry[metricName].labelKeyValues[labelName] = labelValue
	}
}

// SetValue sets the value to the specific message before sending to prometheus.
func SetValue(metricName string, value float64) {
	if isEnabled {
		metric := metricRegistry[metricName]
		metric.value = value
		metricRegistry[metricName] = metric
	}
}

// This function is a helper that takes care of the acutal code that pushes to the prometheus
// aggregation gateway.
func pushToAggregationGateway(url string, jobName string, collector prometheus.Collector) {
	err := push.New(url, jobName).
		Collector(collector).
		Client(&aggClient{&http.Client{}}).
		Format(expfmt.FmtText).
		Push()
	if err != nil {
		logrus.Debugf("Metrics pushing failed : ", err)
	}
}

func pushAllToAggregationGateway(url string, collectors []prometheus.Collector) {
	allJobsName := "all"
	pushClient := push.New(url, allJobsName).Client(&aggClient{&http.Client{}}).Format(expfmt.FmtText)

	for _, value := range collectors {
		pushClient.Collector(value)
	}

	err := pushClient.Push()
	if err != nil {
		logrus.Debugf("Metrics pushing failed : ", err)
	}
}

// pushMetricVectorToGateway calles the pushToAggregationGateway for the given metric.
// This function acts as a re-router which currently pushes to the aggregation gateway.
// Any change in gateway logic can be written here to route the metrics.
func pushMetricVectorToGateway(jobName string, collector prometheus.Collector) {
	pushToAggregationGateway(url, jobName, collector)
}

// pushAllVectorsToMetricGateway calls the pushAllToAggregationGateway for the all metrics.
// This function acts as a re-router which currently pushes to the aggregation gateway.
// Any change in gateway logic can be written here to route the metrics.
func pushAllVectorsToMetricGateway(collectors []prometheus.Collector) {
	pushAllToAggregationGateway(url, collectors)
}

// Push is a driver function that calles all other helper functions to perform the actual workflow
// that must be done to collect and send metrics to Prometheus. This function assumes the data
// is set to the package level label variable, the init was called and processes the data to push
// to Prometheus in the order.
func Push(metricName string) {
	if isEnabled {
		if _, found := metricsBuilt[metricName]; found == true {
			for _, collector := range metricsBuilt[metricName] {
				pushMetricVectorToGateway(metricName, collector)
			}
		} else {
			metric := metricRegistry[metricName]
			collector := metric.build()
			pushMetricVectorToGateway(metricName, collector)
		}
	}
}

var logChangedTime int64

// PushAll function takes all the metrics whose label values are set and pushes them to Prometheus.
func PushAll() {
	var collectors []prometheus.Collector
	if isEnabled {
		for key, value := range metricRegistry {
			if len(value.labelKeyValues) != 0 {
				if _, found := metricsBuilt[key]; found == true {
					for _, collector := range metricsBuilt[key] {
						collectors = append(collectors, collector)
					}
				} else {
					metric := metricRegistry[key]
					collector := metric.build()
					collectors = append(collectors, collector)
				}
			}
		}
	}
	if len(collectors) != 0 {
		pushAllVectorsToMetricGateway(collectors)
	}
}
