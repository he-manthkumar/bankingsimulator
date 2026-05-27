package temporalsetup

import (
	"context"
	"log"

	"banking/gin-service/client"
	"banking/gin-service/temporal/activities"
	"banking/gin-service/temporal/workflows"

	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const TaskQueue = "settlement-task-queue"

func StartWorker(tc temporalclient.Client) {
	act := &activities.SettlementActivity{
		BankingClient: client.NewCoreBankingClient(),
	}

	w := worker.New(tc, TaskQueue, worker.Options{})
	w.RegisterWorkflow(workflows.SettlementWorkflow)
	w.RegisterActivity(act) 
	if err := w.Start(); err != nil {
		log.Fatal("Temporal: failed to start worker:", err)
	}
	log.Println("Temporal: worker started, listening on queue:", TaskQueue)
}

func StartSettlementWorkflow(
	tc temporalclient.Client,
	transferID, fromAccount, toAccount string,
	amount float64,
	transferMode, tpin string,
) {
	_, err := tc.ExecuteWorkflow(
		context.Background(),
		temporalclient.StartWorkflowOptions{
			ID:        "settlement-" + transferID, 
			
			TaskQueue: TaskQueue,
		},
		workflows.SettlementWorkflow,
		workflows.SettlementWorkflowInput{
			TransferID:   transferID,
			FromAccount:  fromAccount,
			ToAccount:    toAccount,
			Amount:       amount,
			TransferMode: transferMode,
			Tpin:         tpin,
		},
	)
	if err != nil {
		log.Printf("Temporal: failed to start %s settlement workflow for transfer %s: %v",
			transferMode, transferID, err)
		return
	}
	log.Printf("Temporal: started %s settlement workflow for transfer %s", transferMode, transferID)
}
