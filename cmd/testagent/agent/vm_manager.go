package agent

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"fulcrumproject.org/core/cmd/testagent/config"
)

// VMState represents the possible states of a VM
type VMState string

const (
	VMStateNONE     VMState = "NONE"
	VMStateCREATING VMState = "CREATING"
	VMStateCREATED  VMState = "CREATED"
	VMStateSTARTING VMState = "STARTING"
	VMStateRUNNING  VMState = "RUNNING"
	VMStateSTOPPING VMState = "STOPPING"
	VMStateSTOPPED  VMState = "STOPPED"
	VMStateDELETING VMState = "DELETING"
	VMStateDELETED  VMState = "DELETED"
	VMStateERROR    VMState = "ERROR"
)

// VM represents a simulated virtual machine
type VM struct {
	ID           string
	Name         string
	State        VMState
	CreatedAt    time.Time
	CPU          float64 // Simulated CPU usage (0-100%)
	Memory       float64 // Simulated memory usage (0-100%)
	Disk         float64 // Simulated disk usage (0-100%)
	Network      float64 // Simulated network throughput (Mbps)
	ErrorMessage string  // Contains error message if State is ERROR
}

// VMManager handles the simulation of VM lifecycles
type VMManager struct {
	vms        map[string]*VM
	mutex      sync.RWMutex
	config     *config.Config
	nextVMID   int
	metrics    *MetricsCollector
	errorRate  float64
	delayRange [2]time.Duration // Min and max operation delay
}

// NewVMManager creates a new VM manager
func NewVMManager(config *config.Config, metrics *MetricsCollector) *VMManager {
	return &VMManager{
		vms:       make(map[string]*VM),
		config:    config,
		metrics:   metrics,
		nextVMID:  1,
		errorRate: config.ErrorRate,
		delayRange: [2]time.Duration{
			config.OperationDelayMin,
			config.OperationDelayMax,
		},
	}
}

// GetVMs returns all managed VMs
func (m *VMManager) GetVMs() []*VM {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	vms := make([]*VM, 0, len(m.vms))
	for _, vm := range m.vms {
		vms = append(vms, vm)
	}
	return vms
}

// GetVM returns a VM by ID
func (m *VMManager) GetVM(id string) (*VM, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	vm, exists := m.vms[id]
	return vm, exists
}

// CreateVM starts the VM creation process
func (m *VMManager) CreateVM(name string) (*VM, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	vmID := fmt.Sprintf("vm-%d", m.nextVMID)
	m.nextVMID++

	vm := &VM{
		ID:        vmID,
		Name:      name,
		State:     VMStateNONE,
		CreatedAt: time.Now(),
		CPU:       0,
		Memory:    0,
		Disk:      0,
		Network:   0,
	}

	m.vms[vmID] = vm

	// Start the creation process asynchronously
	go m.simulateCreateVM(vmID)

	return vm, nil
}

// StartVM starts a stopped VM
func (m *VMManager) StartVM(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return fmt.Errorf("VM not found: %s", id)
	}

	if vm.State != VMStateCREATED && vm.State != VMStateSTOPPED {
		return fmt.Errorf("VM cannot be started from state %s", vm.State)
	}

	vm.State = VMStateSTARTING

	// Start the VM asynchronously
	go m.simulateStartVM(id)

	return nil
}

// StopVM stops a running VM
func (m *VMManager) StopVM(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return fmt.Errorf("VM not found: %s", id)
	}

	if vm.State != VMStateRUNNING {
		return fmt.Errorf("VM cannot be stopped from state %s", vm.State)
	}

	vm.State = VMStateSTOPPING

	// Stop the VM asynchronously
	go m.simulateStopVM(id)

	return nil
}

// DeleteVM deletes a VM
func (m *VMManager) DeleteVM(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return fmt.Errorf("VM not found: %s", id)
	}

	if vm.State != VMStateCREATED && vm.State != VMStateSTOPPED {
		return fmt.Errorf("VM cannot be deleted from state %s", vm.State)
	}

	vm.State = VMStateDELETING

	// Delete the VM asynchronously
	go m.simulateDeleteVM(id)

	return nil
}

// Simulation methods
func (m *VMManager) simulateCreateVM(id string) {
	start := time.Now()

	delay := m.randomDelay()
	time.Sleep(delay)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return // VM was removed during simulation
	}

	duration := time.Since(start)

	// Simulate random failures
	if m.shouldFail() {
		vm.State = VMStateERROR

		if m.metrics != nil {
			m.metrics.RecordVMOperationFailure(vm, "create")
		}

		vm.ErrorMessage = "Failed to create VM: simulated error"
		return
	}

	// Initialize VM properties
	vm.State = VMStateCREATED
	vm.Disk = 10.0 + rand.Float64()*30.0 // 10-40% initial disk usage

	// Record metrics if metrics collector is available
	if m.metrics != nil {
		m.metrics.RecordVMStateChange(vm)
		m.metrics.RecordVMOperationDuration(vm, "create", duration)
	}
}

func (m *VMManager) simulateStartVM(id string) {
	start := time.Now()

	delay := m.randomDelay()
	time.Sleep(delay)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return // VM was removed during simulation
	}

	duration := time.Since(start)

	// Simulate random failures
	if m.shouldFail() {
		vm.State = VMStateERROR

		if m.metrics != nil {
			m.metrics.RecordVMOperationFailure(vm, "start")
		}

		vm.ErrorMessage = "Failed to start VM: simulated error"
		return
	}

	// Initialize runtime properties
	vm.State = VMStateRUNNING
	vm.CPU = 5.0 + rand.Float64()*45.0       // 5-50% initial CPU usage
	vm.Memory = 20.0 + rand.Float64()*40.0   // 20-60% initial memory usage
	vm.Network = 50.0 + rand.Float64()*100.0 // 50-150 Mbps initial network throughput

	// Record metrics if metrics collector is available
	if m.metrics != nil {
		m.metrics.RecordVMStateChange(vm)
		m.metrics.RecordVMOperationDuration(vm, "start", duration)
	}
}

func (m *VMManager) simulateStopVM(id string) {
	start := time.Now()

	delay := m.randomDelay()
	time.Sleep(delay)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return // VM was removed during simulation
	}

	duration := time.Since(start)

	// Simulate random failures
	if m.shouldFail() {
		vm.State = VMStateERROR

		if m.metrics != nil {
			m.metrics.RecordVMOperationFailure(vm, "stop")
		}

		vm.ErrorMessage = "Failed to stop VM: simulated error"
		return
	}

	// Update VM properties
	vm.State = VMStateSTOPPED
	vm.CPU = 0
	vm.Memory = 0
	vm.Network = 0

	// Record metrics if metrics collector is available
	if m.metrics != nil {
		m.metrics.RecordVMStateChange(vm)
		m.metrics.RecordVMOperationDuration(vm, "stop", duration)
	}
}

func (m *VMManager) simulateDeleteVM(id string) {
	start := time.Now()

	delay := m.randomDelay()
	time.Sleep(delay)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return // VM was removed during simulation
	}

	duration := time.Since(start)

	// Simulate random failures
	if m.shouldFail() {
		vm.State = VMStateERROR

		if m.metrics != nil {
			m.metrics.RecordVMOperationFailure(vm, "delete")
		}

		vm.ErrorMessage = "Failed to delete VM: simulated error"
		return
	}

	// Mark as deleted
	vm.State = VMStateDELETED

	// Record metrics if metrics collector is available
	if m.metrics != nil {
		m.metrics.RecordVMStateChange(vm)
		m.metrics.RecordVMOperationDuration(vm, "delete", duration)
	}

	// Remove from map after a short delay
	go func() {
		time.Sleep(5 * time.Second)
		m.mutex.Lock()
		delete(m.vms, id)
		m.mutex.Unlock()
	}()
}

// UpdateVMResources periodically updates resource usage for running VMs
func (m *VMManager) UpdateVMResources() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, vm := range m.vms {
		if vm.State == VMStateRUNNING {
			// Simulate changing resource utilization
			// CPU fluctuates more than memory
			vm.CPU = clamp(vm.CPU+(rand.Float64()*20.0-10.0), 1.0, 95.0)
			vm.Memory = clamp(vm.Memory+(rand.Float64()*10.0-3.0), 5.0, 90.0)
			vm.Disk = clamp(vm.Disk+(rand.Float64()*2.0-0.5), 10.0, 95.0)
			vm.Network = clamp(vm.Network+(rand.Float64()*30.0-15.0), 1.0, 500.0)

			// Record resource metrics if metrics collector is available
			if m.metrics != nil {
				m.metrics.RecordVMResourceUsage(vm)
			}
		}
	}
}

// GetStateCounts returns the count of VMs in each state
func (m *VMManager) GetStateCounts() map[VMState]int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	counts := make(map[VMState]int)
	for _, vm := range m.vms {
		counts[vm.State]++
	}

	// Record VM count metric if metrics collector is available
	if m.metrics != nil {
		m.metrics.RecordVMCount(len(m.vms))
	}

	return counts
}

// Helper methods
func (m *VMManager) randomDelay() time.Duration {
	minDelay := m.delayRange[0]
	maxDelay := m.delayRange[1]

	// Calculate a random duration between min and max
	delta := maxDelay - minDelay
	if delta <= 0 {
		return minDelay
	}

	randomMs := rand.Int63n(int64(delta))
	return minDelay + time.Duration(randomMs)
}

func (m *VMManager) shouldFail() bool {
	return rand.Float64() < m.errorRate
}

// Helper function to clamp a value between min and max
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
