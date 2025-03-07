package agent

import (
	"fmt"
	"log"
)

// Metric type name constants
const (
	MetricTypeVMCPUUsage          = "vm.cpu.usage"
	MetricTypeVMMemoryUsage       = "vm.memory.usage"
	MetricTypeVMDiskUsage         = "vm.disk.usage"
	MetricTypeVMNetworkThroughput = "vm.network.throughput"
)

type MetricsReporter struct {
	client    FulcrumClient
	vmManager *VMManager
}

func NewMetricsReporter(client FulcrumClient, vmManager *VMManager) *MetricsReporter {
	return &MetricsReporter{
		client:    client,
		vmManager: vmManager,
	}
}

func (r *MetricsReporter) Report() (int, error) {
	vms := r.vmManager.GetVMs()

	allMetrics := []MetricEntry{}
	for _, vm := range vms {
		if vm.State != VMStateSTARTED {
			log.Printf("Not reporting metrics for VM %s state %s", vm.ID, vm.State)
			continue
		}
		log.Printf("Reporting metrics for VM %s", vm.ID)
		metrics := []MetricEntry{
			{
				TypeName:   MetricTypeVMCPUUsage,
				Value:      vm.CPUUsage,
				ResourceID: vm.ID,
				ExternalID: vm.ID,
			},
			{
				TypeName:   MetricTypeVMMemoryUsage,
				Value:      vm.MemoryUsage,
				ResourceID: vm.ID,
				ExternalID: vm.ID,
			},
			{
				TypeName:   MetricTypeVMDiskUsage,
				Value:      vm.DiskUsage,
				ResourceID: vm.ID,
				ExternalID: vm.ID,
			},
			{
				TypeName:   MetricTypeVMNetworkThroughput,
				Value:      vm.NetworkUsage,
				ResourceID: vm.ID,
				ExternalID: vm.ID,
			},
		}

		allMetrics = append(allMetrics, metrics...)
	}

	for _, metric := range allMetrics {
		if err := r.client.ReportMetric(&metric); err != nil {
			return 0, fmt.Errorf("failed to report metrics: %w", err)
		}
	}

	return len(allMetrics), nil
}
