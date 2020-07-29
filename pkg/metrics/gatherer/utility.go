package gatherer

import (
	"runtime"

	"github.com/openshift/installer/pkg/metrics/timer"
	"github.com/openshift/installer/pkg/version"
)

// InitializeInvocationMetrics is a utility function that adds the common information to create
// an invocation metrics.
func InitializeInvocationMetrics(metricName string) {
	AddLabelValue(metricName, "result", "Success")
	AddLabelValue(metricName, "returnCode", "1")
	AddLabelValue(CurrentInvocationContext, "os", runtime.GOOS)
	CurrentInvocationContext = metricName
}

// LogError sets the result and returnCode to the right values and pushes the information to the
// gateway. Must be called before the installer breaks execution.
func LogError(err string, metricName string) {
	AddLabelValue(metricName, "result", err)
	AddLabelValue(metricName, "returnCode", "0")

	SendPrometheusInvocationData(metricName)
}

// SendPrometheusInvocationData gets the timer information for the duration metric and pushes it to
// the gateway.
func SendPrometheusInvocationData(metricName string) {
	duration := timer.StopTimer(timer.TotalTimeElapsed)
	SetValue(metricName, duration.Minutes())
	version, err := version.Version()
	if err != nil {
		AddLabelValue(CurrentInvocationContext, "version", version)
	}
	PushAll()
}

// UpdateDurationMetricsWithError sets all the duration metrics to error message.
func UpdateDurationMetricsWithError(err string) {
	listOfDurationMetrics := []string{
		DurationAPIMetricName,
		DurationBootstrapMetricName,
		DurationInfrastructureMetricName,
		DurationOperatorsMetricName,
		DurationProvisioningMetricName,
	}

	for _, item := range listOfDurationMetrics {
		AddLabelValue(item, "result", err)
	}
}

// UpdateDurationMetrics initializes the duration metrics to success.
func UpdateDurationMetrics() {
	listOfDurationMetrics := []string{
		DurationAPIMetricName,
		DurationBootstrapMetricName,
		DurationInfrastructureMetricName,
		DurationOperatorsMetricName,
		DurationProvisioningMetricName,
	}

	for _, item := range listOfDurationMetrics {
		AddLabelValue(item, "command", "create")
		AddLabelValue(item, "result", "success")
	}
}
