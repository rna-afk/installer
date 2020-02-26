package metrics

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func logInitialize() {
	logrus.SetOutput(ioutil.Discard)
	logrus.SetLevel(logrus.TraceLevel)
}

func getTestingMetricBuilderForAddLabelValueAndSetValue(t *testing.T) MetricBuilder {
	return MetricBuilder{
		labels:         []string{},
		labelKeyValues: make(map[string]string),
		desc:           "This test metric stub is used for testing",
		name:           "cluster_installation_metric_test",
		value:          0,
		buckets:        nil,
		metricType:     "duration",
	}
}

func testInvocationObject(m MetricBuilder, t *testing.T) {
	if len(m.labels) != 0 {
		t.Error("labels field overridden during function call")
	}

	if len(m.labelKeyValues) != 6 {
		t.Errorf("Not the correct number of labels, expected %d, got %d", 6, len(m.labelKeyValues))
	}
	labelValues := []string{"testValue", "testValue2", "testValue3", "os", "version", "platform"}

	for index, item := range []string{"testLabel", "testLabel2", "testLabel3"} {
		if value, found := m.labelKeyValues[item]; !found {
			t.Errorf("Label %s not found in labels map", item)
		} else if value != labelValues[index] {
			t.Errorf("Label value for label %s not getting set right. Expected:%s, Actual:%s", item, labelValues[index], value)
		}
	}

	if m.name != "cluster_installation_metric_test" {
		t.Errorf("Metric name expected is %s, got %s", "cluster_installation_metric_test", m.name)
	}

	if m.desc != "This test metric stub is used for testing" {
		t.Errorf("Description getting mutated, expected: %s, got %s", "This test metric stub is used for testing", m.desc)
	}

	if m.value != 1 {
		t.Error("Value set is wrong")
	}

	if m.buckets != nil {
		t.Error("Buckets must be set to nil but is getting set")
	}
}

func runTests(metricName string, t *testing.T) prometheus.Collector {

	AddLabelValue(metricName, "testLabel", "testValue")
	AddLabelValue(metricName, "testLabel2", "testValue2")
	AddLabelValue(metricName, "testLabel3", "testValue3")

	SetValue(metricName, 1)

	return metricRegistry[metricName].build()
}

func TestAddLabelValueWithCustomBuilderObject(t *testing.T) {
	logInitialize()
	metricName := "testMetric"
	Initialize()
	metricRegistry[metricName] = getTestingMetricBuilderForAddLabelValueAndSetValue(t)
	runTests(metricName, t)
	testInvocationObject(metricRegistry[metricName], t)
}

func getTestingMetricBuilderForStopMetrics(t *testing.T) MetricBuilder {
	return MetricBuilder{
		labels:         []string{},
		labelKeyValues: make(map[string]string),
		desc:           "This test metric stub is used for testing",
		name:           "cluster_installation_metric_test",
		value:          0,
		buckets:        nil,
		metricType:     "duration",
	}
}

func testInvocationObject2(m MetricBuilder, t *testing.T) {
	if len(m.labels) != 0 {
		t.Error("labels field overridden during function call")
	}

	if len(m.labelKeyValues) != 3 {
		t.Errorf("Not the correct number of labels, expected %d, got %d", 3, len(m.labelKeyValues))
	}

	if m.name != "cluster_installation_metric_test" {
		t.Errorf("Metric name expected is %s, got %s", "cluster_installation_metric_test", m.name)
	}

	if m.desc != "This test metric stub is used for testing" {
		t.Errorf("Description getting mutated, expected: %s, got %s", "This test metric stub is used for testing", m.desc)
	}

	if m.value != 0 {
		t.Error("Value set is wrong")
	}

	if m.buckets != nil {
		t.Error("Buckets must be set to nil but is getting set")
	}
}

func TestStopMetrics(t *testing.T) {
	logInitialize()
	metricName := "testMetric"
	os.Setenv("OPENSHIFT_INSTALL_DISABLE_METRICS", "1")
	Initialize()
	metricRegistry[metricName] = getTestingMetricBuilderForStopMetrics(t)
	runTests(metricName, t)
	testInvocationObject2(metricRegistry[metricName], t)
}

func TestDataStructurePopulation(t *testing.T) {
	logInitialize()
	Initialize()
	listOfKeys := []string{
		ClusterInstallationCreateJobName,
		ClusterInstallationDestroyJobName,
		ClusterInstallationDurationAPIJobName,
		ClusterInstallationDurationBootstrapJobName,
		ClusterInstallationDurationInfrastructureJobName,
		ClusterInstallationDurationOperatorsJobName,
		ClusterInstallationDurationProvisioningJobName,
		ClusterInstallationGatherJobName,
		ClusterInstallationIgnitionJobName,
		ClusterInstallationInstallConfigJobName,
		ClusterInstallationManifestsJobName,
		ClusterInstallationModificationBootstrapJobName,
		ClusterInstallationModificationCVOJobName,
		ClusterInstallationModificationCaJobName,
		ClusterInstallationModificationConfigJobName,
		ClusterInstallationModificationDNSJobName,
		ClusterInstallationModificationEtcdJobName,
		ClusterInstallationModificationMachineConfigJobName,
		ClusterInstallationModificationMasterJobName,
		ClusterInstallationModificationNetworkJobName,
		ClusterInstallationModificationPullSecretJobName,
		ClusterInstallationModificationSchedulerJobName,
		ClusterInstallationModificationWorkerJobName,
		ClusterInstallationWaitforJobName,
	}
	for _, key := range listOfKeys {
		if _, found := metricRegistry[key]; !found {
			t.Errorf("Metrics Registry does not have the %s metric builder", key)
		}
	}
}

func TestAddLabelValues(t *testing.T) {
	logInitialize()
	Initialize()
	isEnabled = true
	metricName := ClusterInstallationManifestsJobName
	AddLabelValue(metricName, "testLabel", "testValue")

	if builder, found := metricRegistry[metricName]; !found {
		t.Errorf("Metric %s does not exist", metricName)
	} else if value, found := builder.labelKeyValues["testLabel"]; !found {
		t.Errorf("Label %s not being set", "testLabel")
	} else if value != "testValue" {
		t.Errorf("Label value not right, expected %s, got %s", "testValue", value)
	}
}

func TestSetValue(t *testing.T) {
	logInitialize()
	Initialize()
	isEnabled = true
	metricName := ClusterInstallationManifestsJobName
	SetValue(metricName, 1)

	if builder, found := metricRegistry[metricName]; !found {
		t.Errorf("Metric %s does not exist", metricName)
	} else if builder.value != 1 {
		t.Errorf("Value is not being set right, expected %f, got %f", 1.0, builder.value)
	}
}
