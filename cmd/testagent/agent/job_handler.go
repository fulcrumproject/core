package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// JobHandler processes jobs from the Fulcrum Core job queue
type JobHandler struct {
	client    *FulcrumClient
	vmManager *VMManager
	stats     struct {
		processed int
		succeeded int
		failed    int
	}
}

// NewJobHandler creates a new job handler
func NewJobHandler(client *FulcrumClient, vmManager *VMManager) *JobHandler {
	return &JobHandler{
		client:    client,
		vmManager: vmManager,
	}
}

// PollAndProcessJobs polls for pending jobs and processes them
func (h *JobHandler) PollAndProcessJobs() error {
	// Get pending jobs
	jobs, err := h.client.GetPendingJobs()
	if err != nil {
		return fmt.Errorf("failed to get pending jobs: %w", err)
	}

	if len(jobs) > 0 {
		log.Printf("Found %d pending jobs", len(jobs))
	}

	for _, job := range jobs {
		// Increment processed count
		h.stats.processed++

		// Claim the job
		if err := h.client.ClaimJob(job.ID); err != nil {
			log.Printf("Failed to claim job %s: %v", job.ID, err)
			h.stats.failed++
			continue
		}

		log.Printf("Processing job %s of type %s", job.ID, job.Type)

		// Process the job
		err := h.processJob(job)

		if err != nil {
			// Mark job as failed
			log.Printf("Job %s failed: %v", job.ID, err)
			h.stats.failed++

			if failErr := h.client.FailJob(job.ID, err.Error()); failErr != nil {
				log.Printf("Failed to mark job %s as failed: %v", job.ID, failErr)
			}
		} else {
			// Job succeeded
			h.stats.succeeded++
			log.Printf("Job %s completed successfully", job.ID)
		}
	}

	return nil
}

// processJob processes a job based on its type
func (h *JobHandler) processJob(job Job) error {
	switch job.Type {
	case JobTypeServiceCreate:
		return h.handleServiceCreate(job)
	case JobTypeServiceUpdate:
		return h.handleServiceUpdate(job)
	case JobTypeServiceDelete:
		return h.handleServiceDelete(job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

// handleServiceCreate processes service creation jobs
func (h *JobHandler) handleServiceCreate(job Job) error {
	// Parse request data
	var request struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(job.RequestData, &request); err != nil {
		return fmt.Errorf("failed to parse request data: %w", err)
	}

	// Validate request
	if request.Name == "" {
		return fmt.Errorf("VM name is required")
	}

	// Create the VM
	vm, err := h.vmManager.CreateVM(request.Name)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	// Wait for VM to reach CREATED state (with timeout)
	err = h.waitForVMState(vm.ID, VMStateCREATED, VMStateERROR, 30*time.Second)
	if err != nil {
		return err
	}

	// Complete the job with result data
	result := map[string]interface{}{
		"vmId":   vm.ID,
		"status": string(vm.State),
	}

	if err := h.client.CompleteJob(job.ID, result); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	return nil
}

// handleServiceUpdate processes service update jobs
func (h *JobHandler) handleServiceUpdate(job Job) error {
	// Parse request data
	var request struct {
		VMID   string `json:"vmId"`
		Action string `json:"action"`
	}

	if err := json.Unmarshal(job.RequestData, &request); err != nil {
		return fmt.Errorf("failed to parse request data: %w", err)
	}

	// Validate request
	if request.VMID == "" {
		return fmt.Errorf("VM ID is required")
	}

	// Check if VM exists
	vm, exists := h.vmManager.GetVM(request.VMID)
	if !exists {
		return fmt.Errorf("VM not found: %s", request.VMID)
	}

	// Perform the requested action
	var targetState VMState
	var err error

	switch request.Action {
	case "start":
		err = h.vmManager.StartVM(request.VMID)
		targetState = VMStateRUNNING
	case "stop":
		err = h.vmManager.StopVM(request.VMID)
		targetState = VMStateSTOPPED
	default:
		return fmt.Errorf("unknown update action: %s", request.Action)
	}

	if err != nil {
		return fmt.Errorf("failed to %s VM: %w", request.Action, err)
	}

	// Wait for VM to reach target state (with timeout)
	err = h.waitForVMState(request.VMID, targetState, VMStateERROR, 30*time.Second)
	if err != nil {
		return err
	}

	// Complete the job with result data
	result := map[string]interface{}{
		"vmId":   request.VMID,
		"status": string(vm.State),
		"action": request.Action,
	}

	if err := h.client.CompleteJob(job.ID, result); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	return nil
}

// handleServiceDelete processes service deletion jobs
func (h *JobHandler) handleServiceDelete(job Job) error {
	// Parse request data
	var request struct {
		VMID string `json:"vmId"`
	}

	if err := json.Unmarshal(job.RequestData, &request); err != nil {
		return fmt.Errorf("failed to parse request data: %w", err)
	}

	// Validate request
	if request.VMID == "" {
		return fmt.Errorf("VM ID is required")
	}

	// Check if VM exists
	_, exists := h.vmManager.GetVM(request.VMID)
	if !exists {
		return fmt.Errorf("VM not found: %s", request.VMID)
	}

	// Delete the VM
	if err := h.vmManager.DeleteVM(request.VMID); err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	// Wait for VM to be deleted (with timeout)
	err := h.waitForVMState(request.VMID, VMStateDELETED, VMStateERROR, 30*time.Second)
	if err != nil {
		return err
	}

	// Complete the job with result data
	result := map[string]interface{}{
		"vmId":   request.VMID,
		"status": "deleted",
	}

	if err := h.client.CompleteJob(job.ID, result); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	return nil
}

// waitForVMState waits for a VM to reach a target state or error state
func (h *JobHandler) waitForVMState(vmID string, targetState, errorState VMState, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		vm, exists := h.vmManager.GetVM(vmID)
		if !exists {
			// If we're waiting for DELETED state and VM doesn't exist anymore, consider it successful
			if targetState == VMStateDELETED {
				return nil
			}
			return fmt.Errorf("VM not found: %s", vmID)
		}

		if vm.State == targetState {
			return nil
		}

		if vm.State == errorState {
			return fmt.Errorf("VM operation failed: %s", vm.ErrorMessage)
		}

		// Wait a bit before checking again
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for VM %s to reach state %s", vmID, targetState)
}

// GetStats returns the job processing statistics
func (h *JobHandler) GetStats() (processed, succeeded, failed int) {
	return h.stats.processed, h.stats.succeeded, h.stats.failed
}
