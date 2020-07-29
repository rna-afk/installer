package gatherer

import (
	"math"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/openshift/installer/pkg/metrics/builder"
)

var (
	// INVOCATION METRICS

	// CreateMetricName is the variable that stores the metric name for the create cluster details.
	createMetricName = "cluster_installation_create"
	// IgnitionMetricName is the variable that stores the metric name for the create ignition details.
	ignitionMetricName = "cluster_installation_ignition"
	// ManifestsMetricName is the variable that stores the metric name for the create manifests details.
	manifestsMetricName = "cluster_installation_manifests"
	// InstallConfigMetricName is the variable that stores the metric name for the create install configs details.
	installConfigMetricName = "cluster_installation_install_config"
	// WaitforMetricName is the variable that stores the metric name for the waitfor command details.
	WaitforMetricName = "cluster_installation_waitfor"
	// GatherMetricName is the variable that stores the metric name for the gather command configs details.
	GatherMetricName = "cluster_installation_gather"
	// DestroyMetricName is the variable that stores the metric name for the destroy command configs details.
	DestroyMetricName = "cluster_installation_destroy"

	// DURATION METRICS

	// DurationProvisioningMetricName is the variable that stores the metric name for the duration provisioning details.
	DurationProvisioningMetricName = "cluster_installation_duration_provisioning"
	// DurationInfrastructureMetricName is the variable that stores the metric name for the duration infrastructure details.
	DurationInfrastructureMetricName = "cluster_installation_duration_infrastructure"
	// DurationOperatorsMetricName is the variable that stores the metric name for the duration operators details.
	DurationOperatorsMetricName = "cluster_installation_duration_operators"
	// DurationBootstrapMetricName is the variable that stores the metric name for the duration bootstrap details.
	DurationBootstrapMetricName = "cluster_installation_duration_bootstrap"
	// DurationAPIMetricName is the variable that stores the metric name for the duration API details.
	DurationAPIMetricName = "cluster_installation_duration_api"

	// MODIFICATION METRICS

	// ModificationConfigMetricName is the variable that stores the metric name for the modification of config manifests details.
	ModificationConfigMetricName = "cluster_installation_modification_config_manifest"
	// ModificationDNSMetricName is the variable that stores the metric name for the modification of DNS manifests details.
	ModificationDNSMetricName = "cluster_installation_modification_dns_manifest"
	// ModificationNetworkMetricName is the variable that stores the metric name for the modification of network manifests details.
	ModificationNetworkMetricName = "cluster_installation_modification_network_manifest"
	// ModificationSchedulerMetricName is the variable that stores the metric name for the modification of scheduler manifests details.
	ModificationSchedulerMetricName = "cluster_installation_modification_scheduler_manifest"
	// ModificationCVOMetricName is the variable that stores the metric name for the modification of CVO manifests details.
	ModificationCVOMetricName = "cluster_installation_modification_cvo_manifest"
	// ModificationEtcdMetricName is the variable that stores the metric name for the modification of etcd manifests details.
	ModificationEtcdMetricName = "cluster_installation_modification_etcd_manifest"
	// ModificationMachineConfigMetricName is the variable that stores the metric name for the modification of machine config manifests details.
	ModificationMachineConfigMetricName = "cluster_installation_modification_machineconfig_manifest"
	// ModificationCaMetricName is the variable that stores the metric name for the modification of ca manifests details.
	ModificationCaMetricName = "cluster_installation_modification_ca_manifest"
	// ModificationPullSecretMetricName is the variable that stores the metric name for the modification of pull secret manifests details.
	ModificationPullSecretMetricName = "cluster_installation_modification_pullsecret_manifest"
	// ModificationBootstrapMetricName is the variable that stores the metric name for the modification of bootstrap manifests details.
	ModificationBootstrapMetricName = "cluster_installation_modification_bootstrap_ignition"
	// ModificationMasterMetricName is the variable that stores the metric name for the modification of master manifests details.
	ModificationMasterMetricName = "cluster_installation_modification_master_ignition"
	// ModificationWorkerMetricName is the variable that stores the metric name for the modification of worker manifests details.
	ModificationWorkerMetricName = "cluster_installation_modification_worker_manifest"
)

var (
	// optsRegistry holds all the information that every metric needs to create a metric builder.
	optsRegistry = map[string]*builder.MetricOpts{
		createMetricName: &builder.MetricOpts{
			Name:   createMetricName,
			Labels: []string{"result", "returnCode", "platform", "os", "version"},
			Desc: "This metric keeps track of the count of the number of times " +
				"the user ran create cluster command in the given OS " +
				"took the time that is lesser than or equal to the value in the duration label.",
			Buckets:    prometheus.LinearBuckets(15, 5, int(math.Ceil(60/5))+1),
			MetricType: builder.Histogram,
		},
		ignitionMetricName: &builder.MetricOpts{
			Name:   ignitionMetricName,
			Labels: []string{"result", "returnCode", "platform", "os", "version"},
			Desc: "This metric keeps track of the count of the number of times " +
				"the user ran create ignition-config command in the given OS " +
				"took the time that is lesser than or equal to the value in the duration label.",
			Buckets:    prometheus.LinearBuckets(15, 5, int(math.Ceil(60/5))+1),
			MetricType: builder.Histogram,
		},
		manifestsMetricName: &builder.MetricOpts{
			Name:   manifestsMetricName,
			Labels: []string{"result", "returnCode", "platform", "os", "version"},
			Desc: "This metric keeps track of the count of the number of times " +
				"the user ran create manifests command in the given OS " +
				"took the time that is lesser than or equal to the value in the duration label.",
			Buckets:    prometheus.LinearBuckets(15, 5, int(math.Ceil(60/5))+1),
			MetricType: builder.Histogram,
		},
		installConfigMetricName: &builder.MetricOpts{
			Name:   installConfigMetricName,
			Labels: []string{"result", "returnCode", "platform", "os", "version"},
			Desc: "This metric keeps track of the count of the number of times " +
				"the user ran create install-config command in the given OS " +
				"took the time that is lesser than or equal to the value in the duration label.",
			Buckets:    prometheus.LinearBuckets(15, 5, int(math.Ceil(60/5))+1),
			MetricType: builder.Histogram,
		},
		WaitforMetricName: &builder.MetricOpts{
			Name:   WaitforMetricName,
			Labels: []string{"result", "returnCode", "platform", "os", "version"},
			Desc: "This metric keeps track of the count of the number of times " +
				"the user ran wait-for command in the given OS " +
				"took the time that is lesser than or equal to the value in the duration label.",
			Buckets:    prometheus.LinearBuckets(5, 5, int(math.Ceil(30/5))+1),
			MetricType: builder.Histogram,
		},
		DestroyMetricName: &builder.MetricOpts{
			Name:   DestroyMetricName,
			Labels: []string{"result", "returnCode", "platform", "os", "version"},
			Desc: "This metric keeps track of the count of the number of times " +
				"the user ran destroy command in the given OS " +
				"took the time that is lesser than or equal to the value in the duration label.",
			Buckets:    prometheus.LinearBuckets(5, 5, int(math.Ceil(30/5))+1),
			MetricType: builder.Histogram,
		},
		GatherMetricName: &builder.MetricOpts{
			Name:   GatherMetricName,
			Labels: []string{"result", "returnCode", "platform", "os", "version"},
			Desc: "This metric keeps track of the count of the number of times " +
				"the user ran gather command in the given OS " +
				"took the time that is lesser than or equal to the value in the duration label.",
			Buckets:    prometheus.LinearBuckets(5, 5, int(math.Ceil(30/5))+1),
			MetricType: builder.Histogram,
		},
		DurationAPIMetricName: &builder.MetricOpts{
			Name: DurationAPIMetricName,
			Labels: []string{"provisioning", "infrastructure", "bootstrap",
				"api", "operators", "success"},
			Desc: "This metric keeps track of the API stage" +
				"of the installer create command execution and the time it" +
				"took to complete the given stage. The values are kept as labels",
			Buckets:    prometheus.LinearBuckets(5, 5, int(math.Ceil(60/5))+1),
			MetricType: builder.Histogram,
		},
		DurationBootstrapMetricName: &builder.MetricOpts{
			Name: DurationBootstrapMetricName,
			Labels: []string{"provisioning", "infrastructure", "bootstrap",
				"api", "operators", "success"},
			Desc: "This metric keeps track of the bootstrap stage" +
				"of the installer create command execution and the time it" +
				"took to complete the given stage. The values are kept as labels",
			Buckets:    prometheus.LinearBuckets(5, 5, int(math.Ceil(60/5))+1),
			MetricType: builder.Histogram,
		},
		DurationInfrastructureMetricName: &builder.MetricOpts{
			Name: DurationInfrastructureMetricName,
			Labels: []string{"provisioning", "infrastructure", "bootstrap",
				"api", "operators", "success"},
			Desc: "This metric keeps track of the infrastructure stage" +
				"of the installer create command execution and the time it" +
				"took to complete the given stage. The values are kept as labels",
			Buckets:    prometheus.LinearBuckets(5, 5, int(math.Ceil(60/5))+1),
			MetricType: builder.Histogram,
		},
		DurationOperatorsMetricName: &builder.MetricOpts{
			Name: DurationBootstrapMetricName,
			Labels: []string{"provisioning", "infrastructure", "bootstrap",
				"api", "operators", "success"},
			Desc: "This metric keeps track of the operator stage" +
				"of the installer create command execution and the time it" +
				"took to complete the given stage. The values are kept as labels",
			Buckets:    prometheus.LinearBuckets(5, 5, int(math.Ceil(60/5))+1),
			MetricType: builder.Histogram,
		},
		DurationProvisioningMetricName: &builder.MetricOpts{
			Name: DurationBootstrapMetricName,
			Labels: []string{"provisioning", "infrastructure", "bootstrap",
				"api", "operators", "success"},
			Desc: "This metric keeps track of the provisioning stage" +
				"of the installer create command execution and the time it" +
				"took to complete the given stage. The values are kept as labels",
			Buckets:    prometheus.LinearBuckets(5, 5, int(math.Ceil(60/5))+1),
			MetricType: builder.Histogram,
		},
		ModificationBootstrapMetricName: &builder.MetricOpts{
			Name:   ModificationBootstrapMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the bootstrap ignition category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationCVOMetricName: &builder.MetricOpts{
			Name:   ModificationCVOMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the CVO category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationCaMetricName: &builder.MetricOpts{
			Name:   ModificationCVOMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the CA category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationConfigMetricName: &builder.MetricOpts{
			Name:   ModificationConfigMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the config category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationDNSMetricName: &builder.MetricOpts{
			Name:   ModificationDNSMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the DNS category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationEtcdMetricName: &builder.MetricOpts{
			Name:   ModificationEtcdMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the etcd category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationMachineConfigMetricName: &builder.MetricOpts{
			Name:   ModificationMachineConfigMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the machine config category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationMasterMetricName: &builder.MetricOpts{
			Name:   ModificationMasterMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the master category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationNetworkMetricName: &builder.MetricOpts{
			Name:   ModificationNetworkMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the network category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationPullSecretMetricName: &builder.MetricOpts{
			Name:   ModificationPullSecretMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the pull secret category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationSchedulerMetricName: &builder.MetricOpts{
			Name:   ModificationSchedulerMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the scheduler category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
		ModificationWorkerMetricName: &builder.MetricOpts{
			Name:   ModificationWorkerMetricName,
			Labels: []string{"result"},
			Desc: "This metric keeps track of all the assets in the worker category that were modified " +
				"by the user before the invocation of the create command in the installer",
			Buckets:    nil,
			MetricType: builder.Counter,
		},
	}
)
