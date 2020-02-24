package metrics

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
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

// MetricInitializer is the basic metric type that must be used by any potential metric added to
// the system. Any metric added must have the following functions which would be used in the
// workflow.
type MetricInitializer interface {
	getMetricName() string
	objectBuilder() prometheus.Collector
	labelSetter(string, string)
	valueSetter(float64)
	multiValueSetter(map[string]string, float64)
	objectPopulator()
	getCollectorObject() prometheus.Collector
	cleanup()
}

var startTimeEpoch int64

// BucketSize defines the intervals of each label duration the times must be in. This defines
// the size of the buckets along with the intervals.
const BucketSize = 2.5
const url = "http://localhost:9092"

var duration string
var isEnabled bool
var listOfMetrics = []MetricInitializer{
	new(Invocation),
	new(Duration),
	new(Modification),
}

var metricRegistry = make(map[string]MetricInitializer)

// Initialize must be called before using the pushing mechanism.
// This function starts a timer and initializes the values of the list of label values for a given
// job, the object to be created for a given job name and also the map that holds the key/value
// labels to be used over the course of the installer to collect data.
func Initialize() {
	startTimeEpoch = time.Now().Unix()
	isEnabled = true
	for _, metric := range listOfMetrics {
		metricName := metric.getMetricName()
		metricRegistry[metricName] = metric
		metricRegistry[metricName].objectBuilder()
	}
}

// AddLabelValue adds a label key/value pair for a given metric. It keeps track of all the
// labels for all metrics if added through this function.
func AddLabelValue(metricName string, labelName string, labelValue string) {
	if(isEnabled) {
		metricRegistry[metricName].labelSetter(labelName, labelValue)
	}	
}

// AddSample takes the collector object in the metric and uses the multiValueSetter function
// to directly set the value to the Collector.
func AddSample(metricName string, metricValue float64, keyValues ...string) {
	if(isEnabled) {
		keyValueMap := make(map[string]string)
		for i := 1; i<len(keyValues); i+=2 {
			keyValueMap[keyValues[i]] = keyValues[i+1]
		}
		metricRegistry[metricName].multiValueSetter(keyValueMap, metricValue)
	}
}

// SetValue sets the value to the specific message before sending to prometheus.
func SetValue(metricName string, value float64) {
	if(isEnabled){
		metricRegistry[metricName].valueSetter(value)
	}	
}

// GetDuration is used to set the value of the duration of the package level variable to a
// specific value.
// This function finds the amount of time elapsed from calling the init function till now and
// calculates the bucket in which the duration value belongs to based on the buckerSize value
// and sets it to the immediate nearest bucket that is greater than
// this value.
func GetDuration() string {
	difference := float64(time.Now().Unix() - startTimeEpoch)
	durationValue := math.Ceil(difference/BucketSize) * BucketSize
	return fmt.Sprintf("%.1f", durationValue)
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
		fmt.Println(err)
	}
}

// pushMetricVectorToGateway calles the pushToGateway for the three metrics for now. This function can be
// extended to push as many metrics as required. Can be easily reconfigured to send to any other
// gateway if needed.
func pushMetricVectorToGateway(jobName string, collector prometheus.Collector) {
	pushToAggregationGateway(url, jobName, collector)
}

// SendDataToPrometheusAggregationGateway is a driver function that calles all other helper functions to perform the actual workflow
// that must be done to collect and send metrics to Prometheus. This function assumes the data
// is set to the package level label variable, the init was called and processes the data to push
// to Prometheus in the order.
func SendDataToPrometheusAggregationGateway(metricName string) {
	if(isEnabled){
		metricRegistry[metricName].objectPopulator()
		collector := metricRegistry[metricName].getCollectorObject()
		pushMetricVectorToGateway(metricName, collector)
		metricRegistry[metricName].cleanup()
	}	
}

// CheckModifiedAssets checks all the files in a given directory and checks if they are modified. If they are modified, it will
// and it to the metrics and push it later.
func CheckModifiedAssets(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
	}

	for _, file := range files {
		stat := file.Sys().(*syscall.Stat_t)
		atime := time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
		ctime := time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))

		fmt.Println(atime)
		fmt.Println(ctime)
	}
}

// StopMetrics stops sending metrics to us if the user chooses not to send data.
func StopMetrics() {
	isEnabled = false
}
