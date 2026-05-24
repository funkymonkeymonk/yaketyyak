package temporal

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func RunWorker() {
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}

	w := worker.New(c, TaskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(BarberWorkflow, workflow.RegisterOptions{
		Name: "BarberWorkflow",
	})
	w.RegisterWorkflowWithOptions(ShaveWorkflow, workflow.RegisterOptions{
		Name: "ShaveWorkflow",
	})
	w.RegisterActivity(YxSync)
	w.RegisterActivity(YakTriageG2G)
	w.RegisterActivity(YakTriage)
	w.RegisterActivity(YakClaim)
	w.RegisterActivity(YakRemoveG2GTag)
	w.RegisterActivity(DispatchAgent)
	w.RegisterActivity(WatchPRCI)
	w.RegisterActivity(MergePR)
	w.RegisterActivity(YakMarkDone)
	w.RegisterActivity(YakMarkRefinement)
	w.RegisterActivity(CheckRefinement)
	w.RegisterActivity(ShaveInitWorkspace)
	w.RegisterActivity(ShaveImplement)
	w.RegisterActivity(ShaveValidate)
	w.RegisterActivity(ShaveAdversarialReview)
	w.RegisterActivity(ShaveCreatePR)
	w.RegisterActivity(ShaveCleanup)

	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	log.Printf("Worker started, polling task queue: %s", TaskQueue)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	w.Stop()
	c.Close()
}
