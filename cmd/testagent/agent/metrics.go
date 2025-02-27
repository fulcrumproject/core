package agent

import (
	"fmt"
	"sync"
	"time"
)

// MetricEntry represents a single metric measurement
type MetricEntry struct {
	TypeID     string    `json:"typeId"`
	AgentID    string    `json:"agentId"`
	ServiceID  string    `json:"serviceId,omitempty"`
	ResourceID string    `json:"resourceId,omitempty"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
}

// MetricsCollector collects and reports metrics
type MetricsCollector struct {
	agentID    string
	mutex      sync.Mutex
	metrics    []MetricEntry
	metricDefs map[string]string // Map of metric name to type ID
	client     *FulcrumClient
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(client *FulcrumClient) *MetricsCollector {
	return &MetricsCollector{
		metrics:    make([]MetricEntry, 0),
		metricDefs: make(map[string]string),
		client:     client,
	}
}

// SetAgentID sets the agent ID for all metrics
func (m *MetricsCollector) SetAgentID(agentID string) {
	m.agentID = agentID
}

// RegisterMetricType registers a metric type with its ID
func (m *MetricsCollector) RegisterMetricType(name, typeID string) {
	m.metricDefs[name] = typeID
}

// RecordVMStateChange records a change in VM state
func (m *MetricsCollector) RecordVMStateChange(vm *VM) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.agentID == "" {
		return // Agent ID not set yet
	}

	typeID, ok := m.metricDefs["vm.state.count"]
	if !ok {
		return // Metric type not registered
	}

	// Create a metric for the VM state count
	// ResourceID format: "vm:{id}:{name}:{state}"
	resourceID := fmt.Sprintf("vm:%s:%s:%s", vm.ID, vm.Name, vm.State)

	m.metrics = append(m.metrics, MetricEntry{
		TypeID:     typeID,
		AgentID:    m.agentID,
		ResourceID: resourceID,
		Value:      1.0, // Increment count for this state
		Timestamp:  time.Now(),
	})
}

// RecordVMOperationDuration records the duration of a VM operation
func (m *MetricsCollector) RecordVMOperationDuration(vm *VM, operation string, duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.agentID == "" {
		return // Agent ID not set yet
	}

	metricName := "vm." + operation + ".duration"
	typeID, ok := m.metricDefs[metricName]
	if !ok {
		return // Metric type not registered
	}

	// ResourceID format: "vm:{id}:{name}:operation:{operation}"
	resourceID := fmt.Sprintf("vm:%s:%s:operation:%s", vm.ID, vm.Name, operation)

	// Create a metric for the operation duration in seconds
	m.metrics = append(m.metrics, MetricEntry{
		TypeID:     typeID,
		AgentID:    m.agentID,
		ResourceID: resourceID,
		Value:      duration.Seconds(),
		Timestamp:  time.Now(),
	})
}

// RecordVMOperationFailure records a failed VM operation
func (m *MetricsCollector) RecordVMOperationFailure(vm *VM, operation string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.agentID == "" {
		return // Agent ID not set yet
	}

	typeID, ok := m.metricDefs["vm.operation.failure"]
	if !ok {
		return // Metric type not registered
	}

	// ResourceID format: "vm:{id}:{name}:failure:{operation}"
	resourceID := fmt.Sprintf("vm:%s:%s:failure:%s", vm.ID, vm.Name, operation)

	// Create a metric for the operation failure
	m.metrics = append(m.metrics, MetricEntry{
		TypeID:     typeID,
		AgentID:    m.agentID,
		ResourceID: resourceID,
		Value:      1.0, // Count of failures
		Timestamp:  time.Now(),
	})
}

// RecordVMResourceUsage records resource usage for a VM
func (m *MetricsCollector) RecordVMResourceUsage(vm *VM) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.agentID == "" {
		return // Agent ID not set yet
	}

	now := time.Now()

	// Record CPU usage
	if typeID, ok := m.metricDefs["vm.cpu.usage"]; ok {
		resourceID := fmt.Sprintf("vm:%s:%s:resource:cpu", vm.ID, vm.Name)
		m.metrics = append(m.metrics, MetricEntry{
			TypeID:     typeID,
			AgentID:    m.agentID,
			ResourceID: resourceID,
			Value:      vm.CPU,
			Timestamp:  now,
		})
	}

	// Record memory usage
	if typeID, ok := m.metricDefs["vm.memory.usage"]; ok {
		resourceID := fmt.Sprintf("vm:%s:%s:resource:memory", vm.ID, vm.Name)
		m.metrics = append(m.metrics, MetricEntry{
			TypeID:     typeID,
			AgentID:    m.agentID,
			ResourceID: resourceID,
			Value:      vm.Memory,
			Timestamp:  now,
		})
	}

	// Record disk usage
	if typeID, ok := m.metricDefs["vm.disk.usage"]; ok {
		resourceID := fmt.Sprintf("vm:%s:%s:resource:disk", vm.ID, vm.Name)
		m.metrics = append(m.metrics, MetricEntry{
			TypeID:     typeID,
			AgentID:    m.agentID,
			ResourceID: resourceID,
			Value:      vm.Disk,
			Timestamp:  now,
		})
	}

	// Record network throughput
	if typeID, ok := m.metricDefs["vm.network.throughput"]; ok {
		resourceID := fmt.Sprintf("vm:%s:%s:resource:network", vm.ID, vm.Name)
		m.metrics = append(m.metrics, MetricEntry{
			TypeID:     typeID,
			AgentID:    m.agentID,
			ResourceID: resourceID,
			Value:      vm.Network,
			Timestamp:  now,
		})
	}
}

// RecordVMCount records the total count of VMs
func (m *MetricsCollector) RecordVMCount(count int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.agentID == "" {
		return // Agent ID not set yet
	}

	typeID, ok := m.metricDefs["vm.count"]
	if !ok {
		return // Metric type not registered
	}

	// Create a metric for the VM count
	m.metrics = append(m.metrics, MetricEntry{
		TypeID:     typeID,
		AgentID:    m.agentID,
		ResourceID: "vm:count",
		Value:      float64(count),
		Timestamp:  time.Now(),
	})
}

// RecordAgentMetric records a general agent metric
func (m *MetricsCollector) RecordAgentMetric(metricName string, value float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.agentID == "" {
		return // Agent ID not set yet
	}

	typeID, ok := m.metricDefs[metricName]
	if !ok {
		return // Metric type not registered
	}

	// Create the metric
	m.metrics = append(m.metrics, MetricEntry{
		TypeID:     typeID,
		AgentID:    m.agentID,
		ResourceID: fmt.Sprintf("agent:%s", metricName),
		Value:      value,
		Timestamp:  time.Now(),
	})
}

// GetPendingMetricsCount returns the number of metrics waiting to be reported
func (m *MetricsCollector) GetPendingMetricsCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.metrics)
}

// ReportMetrics sends collected metrics to Fulcrum Core
func (m *MetricsCollector) ReportMetrics() (int, error) {
	m.mutex.Lock()

	// Make a copy of metrics and clear the collector
	metrics := m.metrics
	count := len(metrics)
	if count == 0 {
		m.mutex.Unlock()
		return 0, nil // No metrics to report
	}

	// Clear the metrics slice to avoid sending duplicates
	m.metrics = make([]MetricEntry, 0, cap(metrics))

	// Release the lock before making the API call to avoid blocking collection
	m.mutex.Unlock()

	// Report metrics to Fulcrum Core using the client
	err := m.client.ReportMetrics(metrics)
	if err != nil {
		return 0, fmt.Errorf("failed to report metrics: %w", err)
	}

	return count, nil
}
