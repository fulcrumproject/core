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
	VMStateCREATED VMState = "CREATED"
	VMStateSTARTED VMState = "STARTED"
	VMStateSTOPPED VMState = "STOPPED"
	VMStateDELETED VMState = "DELETED"
)

// VM represents a simulated virtual machine
type VM struct {
	ID           string
	Name         string
	State        VMState
	CreatedAt    time.Time
	CPU          int
	Memory       int
	CPUUsage     float64 // Simulated CPU usage (0-100%)
	MemoryUsage  float64 // Simulated memory usage (0-100%)
	DiskUsage    float64 // Simulated disk usage (0-100%)
	NetworkUsage float64 // Simulated network throughput (Mbps)
	ErrorMessage string  // Contains error message if State is ERROR
}

// VMManager handles the simulation of VM lifecycles
type VMManager struct {
	vms        map[string]*VM
	mutex      sync.RWMutex
	config     *config.Config
	nextVMID   int // TODO
	errorRate  float64
	delayRange [2]time.Duration // Min and max operation delay
}

// NewVMManager creates a new VM manager
func NewVMManager(config *config.Config) *VMManager {
	return &VMManager{
		vms:       make(map[string]*VM),
		config:    config,
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
func (m *VMManager) CreateVM(id, name string, cpu int, memory int) (*VM, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm := &VM{
		ID:           id,
		Name:         name,
		State:        VMStateCREATED,
		CreatedAt:    time.Now(),
		CPUUsage:     0,
		MemoryUsage:  0,
		DiskUsage:    0,
		NetworkUsage: 0,
	}

	m.vms[id] = vm

	delay := m.randomDelay()
	time.Sleep(delay)

	// Simulate random failures
	if m.shouldFail() {
		vm.ErrorMessage = "Failed to create VM: simulated error"
		return vm, nil
	}

	// Initialize VM properties
	// vm.State = VMStateRUNNING
	vm.CPU = cpu
	vm.Memory = memory
	vm.DiskUsage = 10.0 + rand.Float64()*30.0 // 10-40% initial disk usage

	return vm, nil
}

// StartVM starts a stopped VM
func (m *VMManager) UpdateVM(id, name string, cpu int, memory int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return fmt.Errorf("VM not found: %s", id)
	}

	if vm.State != VMStateSTOPPED && vm.State != VMStateSTARTED {
		return fmt.Errorf("VM cannot be updated from state %s", vm.State)
	}

	delay := m.randomDelay()
	time.Sleep(delay)

	// Simulate random failures
	if m.shouldFail() {
		vm.ErrorMessage = "Failed to start VM: simulated error"
		return nil
	}

	// Initialize runtime properties
	vm.Name = name
	vm.CPU = cpu
	vm.Memory = memory
	vm.CPUUsage = 5.0 + rand.Float64()*45.0       // 5-50% initial CPU usage
	vm.MemoryUsage = 20.0 + rand.Float64()*40.0   // 20-60% initial memory usage
	vm.NetworkUsage = 50.0 + rand.Float64()*100.0 // 50-150 Mbps initial network throughput

	return nil
}

// StartVM starts a stopped VM
func (m *VMManager) StartVM(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return fmt.Errorf("VM not found: %s", id)
	}

	if vm.State != VMStateSTOPPED && vm.State != VMStateCREATED {
		return fmt.Errorf("VM cannot be started from state %s", vm.State)
	}

	delay := m.randomDelay()
	time.Sleep(delay)

	// Simulate random failures
	if m.shouldFail() {
		vm.ErrorMessage = "Failed to start VM: simulated error"
		return nil
	}

	// Initialize runtime properties
	vm.State = VMStateSTARTED
	vm.CPUUsage = 5.0 + rand.Float64()*45.0       // 5-50% initial CPU usage
	vm.MemoryUsage = 20.0 + rand.Float64()*40.0   // 20-60% initial memory usage
	vm.NetworkUsage = 50.0 + rand.Float64()*100.0 // 50-150 Mbps initial network throughput

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

	if vm.State != VMStateSTARTED {
		return fmt.Errorf("VM cannot be stopped from state %s", vm.State)
	}

	delay := m.randomDelay()
	time.Sleep(delay)

	// Simulate random failures
	if m.shouldFail() {
		vm.ErrorMessage = "Failed to stop VM: simulated error"
		return nil
	}

	// Update VM properties
	vm.State = VMStateSTOPPED
	vm.CPUUsage = 0
	vm.MemoryUsage = 0
	vm.NetworkUsage = 0

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

	if vm.State != VMStateSTOPPED {
		return fmt.Errorf("VM cannot be deleted from state %s", vm.State)
	}

	delay := m.randomDelay()
	time.Sleep(delay)

	// Simulate random failures
	if m.shouldFail() {
		vm.ErrorMessage = "Failed to delete VM: simulated error"
		return nil
	}

	// Mark as deleted
	vm.State = VMStateDELETED

	return nil
}

// Retry
func (m *VMManager) Retry(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	vm, exists := m.vms[id]
	if !exists {
		return fmt.Errorf("VM not found: %s", id)
	}

	// Not in error
	if vm.ErrorMessage == "" {
		return nil
	}

	delay := m.randomDelay()
	time.Sleep(delay)

	// Simulate random failures
	if m.shouldFail() {
		vm.ErrorMessage = "Failed to delete VM: simulated error"
		return nil
	}

	// Reset error
	vm.ErrorMessage = ""

	return nil
}

// UpdateVMResources periodically updates resource usage for running VMs
func (m *VMManager) UpdateVMResources() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, vm := range m.vms {
		if vm.State == VMStateSTARTED {
			// Simulate changing resource utilization
			// CPU fluctuates more than memory
			vm.CPUUsage = clamp(vm.CPUUsage+(rand.Float64()*20.0-10.0), 1.0, 95.0)
			vm.MemoryUsage = clamp(vm.MemoryUsage+(rand.Float64()*10.0-3.0), 5.0, 90.0)
			vm.DiskUsage = clamp(vm.DiskUsage+(rand.Float64()*2.0-0.5), 10.0, 95.0)
			vm.NetworkUsage = clamp(vm.NetworkUsage+(rand.Float64()*30.0-15.0), 1.0, 500.0)
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
