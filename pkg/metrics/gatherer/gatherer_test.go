package gatherer

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricsCollection(t *testing.T) {
	cases := []struct {
		enableMetrics  bool
		labelValueMap  map[string]string
		metricName     string
		testName       string
		value          float64
		expectedOutput string
	}{
		{
			enableMetrics:  false,
			labelValueMap:  map[string]string{"test1": "test1"},
			metricName:     createMetricName,
			testName:       "basic histogram metric test",
			value:          10,
			expectedOutput: "",
		},
		{
			enableMetrics:  true,
			labelValueMap:  map[string]string{"test1": "test1"},
			metricName:     createMetricName,
			testName:       "basic disabled metrics test",
			value:          10,
			expectedOutput: "",
		},
		{
			enableMetrics:  false,
			labelValueMap:  map[string]string{"test1": "test1"},
			metricName:     ModificationBootstrapMetricName,
			testName:       "basic counter metric test",
			value:          10,
			expectedOutput: "",
		},
		{
			enableMetrics:  false,
			labelValueMap:  nil,
			metricName:     ModificationBootstrapMetricName,
			testName:       "basic counter metric test, no labels",
			value:          10,
			expectedOutput: "",
		},
		{
			enableMetrics:  false,
			labelValueMap:  nil,
			metricName:     ModificationBootstrapMetricName,
			testName:       "basic counter metric test",
			value:          10,
			expectedOutput: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(t *testing.T) {
			if tc.enableMetrics {
				os.Setenv("OPENSHIFT_INSTALL_DISABLE_METRICS", "TRUE")
			}
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				buf := new(strings.Builder)
				_, err := io.Copy(buf, req.Body)
				assert.NoError(t, err)
				assert.EqualValues(t, tc.expectedOutput, buf.String())
			}))
			defer testServer.Close()
			os.Setenv("OPENSHIFT_INSTALL_METRICS_ENDPOINT", testServer.URL)

			Initialize()
			for key, value := range tc.labelValueMap {
				AddLabelValue(tc.metricName, key, value)
			}
			SetValue(tc.metricName, tc.value)
			Push(tc.metricName)
		})
	}
}
