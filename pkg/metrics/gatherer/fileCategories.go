package gatherer

var (
	// FileCategories is a map of asset names to the categories of the metrics that it belongs to.
	FileCategories = map[string]string{
		"Cloud Provider Config":     ModificationConfigMetricName,
		"Infrastructure Config":     ModificationConfigMetricName,
		"KubeCloudConfig":           ModificationConfigMetricName,
		"KubeSystemConfigmapRootCA": ModificationConfigMetricName,

		"Ingress Config": ModificationDNSMetricName,
		"DNS Config":     ModificationDNSMetricName,

		"Proxy Config":   ModificationNetworkMetricName,
		"Network Config": ModificationNetworkMetricName,

		"Scheduler Config": ModificationSchedulerMetricName,

		"CVOOverrides": ModificationSchedulerMetricName,

		"EtcdCAConfigMap":              ModificationEtcdMetricName,
		"EtcdMetricServingCAConfigMap": ModificationEtcdMetricName,
		"EtcdServingCAConfigMap":       ModificationEtcdMetricName,

		"MachineConfigServerTLSSecret":   ModificationMachineConfigMetricName,
		"OpenshiftMachineConfigOperator": ModificationMachineConfigMetricName,

		"OpenshiftConfigSecretPullSecret": ModificationPullSecretMetricName,

		"Bootstrap Ignition Config": ModificationBootstrapMetricName,

		"Master Ignition Config": ModificationMasterMetricName,
		"Master Machines":        ModificationMasterMetricName,

		"Worker Ignition Config": ModificationWorkerMetricName,
		"Worker Machines":        ModificationWorkerMetricName,
	}
)
