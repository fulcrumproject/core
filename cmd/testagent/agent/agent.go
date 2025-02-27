package agent

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"fulcrumproject.org/core/cmd/testagent/config"
)

// Agent is the main test agent implementation
type Agent struct {
	cfg        *config.Config
	client     *FulcrumClient
	jobHandler *JobHandler
	metrics    *MetricsCollector
	vmManager  *VMManager
	stopCh     chan struct{}
	wg         sync.WaitGroup
	startTime  time.Time
	connected  bool
	agentID    string
}

// New creates a new test agent
func New(cfg *config.Config) (*Agent, error) {
	// Create a new Fulcrum client with the provided token
	client := NewFulcrumClient(cfg.FulcrumAPIURL, cfg.AgentToken)

	// Create metrics collector
	metrics := NewMetricsCollector(client)

	// Create VM manager
	vmManager := NewVMManager(cfg, metrics)

	// Create job handler
	jobHandler := NewJobHandler(client, vmManager)

	return &Agent{
		cfg:        cfg,
		client:     client,
		metrics:    metrics,
		jobHandler: jobHandler,
		vmManager:  vmManager,
		stopCh:     make(chan struct{}),
		connected:  false,
	}, nil
}

// Start starts the agent
func (a *Agent) Start(ctx context.Context) error {
	a.startTime = time.Now()

	// Get agent information to verify the token is valid
	agentInfo, err := a.client.GetAgentInfo()
	if err != nil {
		return fmt.Errorf("failed to get agent information: %w", err)
	}

	// Extract agent ID from the response
	id, ok := agentInfo["id"].(string)
	if !ok {
		return fmt.Errorf("invalid agent information received")
	}
	a.agentID = id

	// Set agent ID in the metrics collector
	a.metrics.SetAgentID(id)

	// Note: Metric types are expected to already exist in the Fulcrum Core system
	// No need to register them here

	log.Printf("Agent authenticated with ID: %s", id)

	// Update agent status to Connected
	if err := a.client.UpdateAgentStatus("Connected"); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}
	a.connected = true

	log.Printf("Agent status updated to Connected")

	// Start a simple background heartbeat to keep the agent alive
	a.wg.Add(1)
	go a.heartbeat(ctx)

	// Start VM resource updater background task
	a.wg.Add(1)
	go a.updateVMResources(ctx)
	a.initializeVMs(ctx)

	// Start metrics reporting background task
	a.wg.Add(1)
	go a.reportMetrics(ctx)

	// Start job polling background task
	a.wg.Add(1)
	go a.pollJobs(ctx)

	return nil
}

// heartbeat periodically updates the agent status to maintain the connection
func (a *Agent) heartbeat(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(60 * time.Second) // Update status every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.client.UpdateAgentStatus("Connected"); err != nil {
				log.Printf("Failed to update agent status: %v", err)
			} else {
				log.Printf("Heartbeat: Agent status updated")
			}
		case <-a.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// updateVMResources periodically updates the resource utilization of running VMs
func (a *Agent) updateVMResources(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // Update resources every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.vmManager.UpdateVMResources()
		case <-a.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// reportMetrics periodically reports collected metrics
func (a *Agent) reportMetrics(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.cfg.MetricReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count, err := a.metrics.ReportMetrics()
			if err != nil {
				log.Printf("Error reporting metrics: %v", err)
			} else if count > 0 {
				log.Printf("Reported %d metrics", count)
			}
		case <-a.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// pollJobs periodically polls for pending jobs and processes them
func (a *Agent) pollJobs(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.cfg.JobPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.jobHandler.PollAndProcessJobs(); err != nil {
				log.Printf("Error polling jobs: %v", err)
			}
		case <-a.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// initializeVMs creates the initial set of VMs based on configuration
func (a *Agent) initializeVMs(ctx context.Context) {
	if a.cfg.VMCount <= 0 {
		return
	}

	log.Printf("Creating %d initial VMs...", a.cfg.VMCount)
	for i := 0; i < a.cfg.VMCount; i++ {
		vmName := fmt.Sprintf("test-vm-%d", i+1)
		vm, err := a.vmManager.CreateVM(vmName)
		if err != nil {
			log.Printf("Failed to create VM %s: %v", vmName, err)
		} else {
			log.Printf("Created VM %s (ID: %s)", vm.Name, vm.ID)
		}
	}

	// Start a goroutine to periodically perform VM operations if configured
	if a.cfg.VMOperationInterval > 0 {
		a.wg.Add(1)
		go a.simulateVMOperations(ctx)
	}
}

// GetVM returns a VM by ID
func (a *Agent) GetVM(id string) (*VM, bool) {
	return a.vmManager.GetVM(id)
}

// GetVMs returns all managed VMs
func (a *Agent) GetVMs() []*VM {
	return a.vmManager.GetVMs()
}

// Shutdown stops the agent and releases resources
func (a *Agent) Shutdown(ctx context.Context) error {
	// Close the stop channel to signal all goroutines to stop
	close(a.stopCh)

	// Wait for all goroutines to complete with a timeout
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines exited successfully
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for goroutines to exit")
	}

	// Update agent status to Disconnected
	if a.connected {
		if err := a.client.UpdateAgentStatus("Disconnected"); err != nil {
			return fmt.Errorf("failed to update agent status on shutdown: %w", err)
		}
		a.connected = false
		log.Println("Agent status updated to Disconnected")
	}

	log.Println("Agent shut down successfully")
	return nil
}

// GetAgentID returns the agent's ID
func (a *Agent) GetAgentID() string {
	return a.agentID
}

// IsConnected returns whether the agent is connected
func (a *Agent) IsConnected() bool {
	return a.connected
}

// GetUptime returns the agent's uptime
func (a *Agent) GetUptime() time.Duration {
	return time.Since(a.startTime)
}

// CreateVM creates a new virtual machine
func (a *Agent) CreateVM(name string) (*VM, error) {
	return a.vmManager.CreateVM(name)
}

// StartVM starts a VM
func (a *Agent) StartVM(id string) error {
	return a.vmManager.StartVM(id)
}

// StopVM stops a VM
func (a *Agent) StopVM(id string) error {
	return a.vmManager.StopVM(id)
}

// DeleteVM deletes a VM
func (a *Agent) DeleteVM(id string) error {
	return a.vmManager.DeleteVM(id)
}

// GetVMStateCounts returns the count of VMs in each state
func (a *Agent) GetVMStateCounts() map[VMState]int {
	return a.vmManager.GetStateCounts()
}

// GetJobStats returns the job processing statistics
func (a *Agent) GetJobStats() (processed, succeeded, failed int) {
	return a.jobHandler.GetStats()
}

// GetPendingMetricsCount returns the number of metrics waiting to be reported
func (a *Agent) GetPendingMetricsCount() int {
	return a.metrics.GetPendingMetricsCount()
}

// simulateVMOperations periodically performs random operations on VMs
func (a *Agent) simulateVMOperations(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.cfg.VMOperationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.performRandomVMOperation()
		case <-a.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performRandomVMOperation performs a random operation on a random VM
func (a *Agent) performRandomVMOperation() {
	vms := a.vmManager.GetVMs()
	if len(vms) == 0 {
		return
	}

	// Choose a random VM
	randomIndex := rand.Intn(len(vms))
	vm := vms[randomIndex]

	// Perform a random operation based on the VM's state
	switch vm.State {
	case VMStateCREATED:
		// Start the VM
		if err := a.vmManager.StartVM(vm.ID); err != nil {
			log.Printf("Failed to start VM %s: %v", vm.ID, err)
		} else {
			log.Printf("Started VM %s", vm.ID)
		}
	case VMStateRUNNING:
		// Stop the VM
		if err := a.vmManager.StopVM(vm.ID); err != nil {
			log.Printf("Failed to stop VM %s: %v", vm.ID, err)
		} else {
			log.Printf("Stopped VM %s", vm.ID)
		}
	case VMStateSTOPPED:
		// Either start or delete the VM
		if rand.Intn(2) == 0 {
			if err := a.vmManager.StartVM(vm.ID); err != nil {
				log.Printf("Failed to start VM %s: %v", vm.ID, err)
			} else {
				log.Printf("Started VM %s", vm.ID)
			}
		} else {
			if err := a.vmManager.DeleteVM(vm.ID); err != nil {
				log.Printf("Failed to delete VM %s: %v", vm.ID, err)
			} else {
				log.Printf("Deleted VM %s", vm.ID)
			}
		}
	case VMStateERROR:
		// Delete the VM
		if err := a.vmManager.DeleteVM(vm.ID); err != nil {
			log.Printf("Failed to delete VM %s: %v", vm.ID, err)
		} else {
			log.Printf("Deleted VM %s (was in ERROR state)", vm.ID)

			// Create a new VM to replace it
			newName := fmt.Sprintf("replacement-vm-%d", rand.Intn(1000))
			if newVM, err := a.vmManager.CreateVM(newName); err != nil {
				log.Printf("Failed to create replacement VM: %v", err)
			} else {
				log.Printf("Created replacement VM %s (ID: %s)", newVM.Name, newVM.ID)
			}
		}
	}
}
