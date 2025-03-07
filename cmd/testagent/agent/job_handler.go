package agent

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
)

// JobHandler processes jobs from the Fulcrum Core job queue
type JobHandler struct {
	client    FulcrumClient
	vmManager *VMManager
	stats     struct {
		processed int
		succeeded int
		failed    int
	}
}

// NewJobHandler creates a new job handler
func NewJobHandler(client FulcrumClient, vmManager *VMManager) *JobHandler {
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

	if len(jobs) == 0 {
		log.Printf("Pending jobs not found")
		return nil
	}
	// First
	job := jobs[0]

	// Increment processed count
	h.stats.processed++

	// Claim the job
	if err := h.client.ClaimJob(job.ID); err != nil {
		log.Printf("Failed to claim job %s: %v", job.ID, err)
		h.stats.failed++
		return err
	}

	log.Printf("Processing job %s of type %s", job.ID, job.Action)

	// Process the job
	err = h.processJob(job)
	if err != nil {
		// Mark job as failed
		log.Printf("Job %s failed: %v", job.ID, err)
		h.stats.failed++

		if failErr := h.client.FailJob(job.ID, err.Error()); failErr != nil {
			log.Printf("Failed to mark job %s as failed: %v", job.ID, failErr)
			return failErr
		}
	} else {
		// Job succeeded
		// Fake resources
		resources := map[string]interface{}{
			"ts": time.Now().Format(time.RFC850),
		}
		if complErr := h.client.CompleteJob(job.ID, resources); complErr != nil {
			log.Printf("Failed to mark job %s as completed: %v", job.ID, complErr)
			return complErr
		}
		h.stats.succeeded++
		log.Printf("Job %s completed successfully", job.ID)
	}

	return nil
}

// processJob processes a job based on its type
func (h *JobHandler) processJob(job *Job) error {
	switch job.Action {
	case JobActionServiceCreate, JobActionServiceColdUpdate, JobActionServiceHotUpdate:
		cpu, mem, err := CPUMemoryFromJob(job)
		if err != nil {
			return err
		}
		switch job.Action {
		case JobActionServiceCreate:
			_, err = h.vmManager.CreateVM(job.Service.ID, job.Service.Name, cpu, mem)
			return err
		case JobActionServiceHotUpdate, JobActionServiceColdUpdate:
			return h.vmManager.UpdateVM(job.Service.ID, job.Service.Name, cpu, mem)
		default:
			return fmt.Errorf("unknown job type: %s", job.Action)
		}
	case JobActionServiceStart:
		return h.vmManager.StartVM(job.Service.ID)
	case JobActionServiceStop:
		return h.vmManager.StopVM(job.Service.ID)
	case JobActionServiceDelete:
		return h.vmManager.DeleteVM(job.Service.ID)
	default:
		return fmt.Errorf("unknown job type: %s", job.Action)
	}
}

func CPUMemoryFromJob(job *Job) (int, int, error) {
	cpu := 2
	memory := 1
	if vv, ok := job.Service.TargetAttributes["cpu"]; ok {
		if len(vv) == 0 {
			return 0, 0, errors.New("no cpu value")
		}
		v, err := strconv.Atoi(vv[0])
		if err != nil {
			return 0, 0, errors.New("invalid cpu value")
		}
		cpu = v
	}
	if vv, ok := job.Service.TargetAttributes["memory"]; ok {
		if len(vv) == 0 {
			return 0, 0, errors.New("no memory value")
		}
		v, err := strconv.Atoi(vv[0])
		if err != nil {
			return 0, 0, errors.New("invalid memory value")
		}
		memory = v
	}
	return cpu, memory, nil
}

// GetStats returns the job processing statistics
func (h *JobHandler) GetStats() (processed, succeeded, failed int) {
	return h.stats.processed, h.stats.succeeded, h.stats.failed
}
