package pushclient

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// PushClient is used to send Prometheus metrics objects to any Prometheus
// Push Gateway. It stores the URL and the client that will be used to push
// to the desired gateway.
type PushClient struct {
	URL     string
	Client  *http.Client
	JobName string
}

// Push takes care of the actual code that pushes to the prometheus
// aggregation gateway. It takes a list of collectors and pushes all of them to the desired url.
func (p *PushClient) Push(collectors ...prometheus.Collector) error {
	pushClient := push.New(p.URL, p.JobName).Client(p.Client).Format(expfmt.FmtText)

	for _, value := range collectors {
		pushClient.Collector(value)
	}

	err := pushClient.Push()
	if err != nil {
		return errors.Wrap(err, "failed to push metrics")
	}
	return nil
}
