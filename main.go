package main

import (
    "context"
    "time"
    "net/http"
    "go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
    "go.uber.org/cadence/compatibility"
    "go.uber.org/cadence/worker"
    "go.uber.org/cadence/workflow"
    "go.uber.org/cadence/activity"

    apiv1 "github.com/uber/cadence-idl/go/proto/api/v1"
    "github.com/uber-go/tally"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "go.uber.org/yarpc"
    "go.uber.org/yarpc/transport/grpc"
    activities "github.com/sanya-sharma/hangry-ghc24/wfactivities"
)

var HostPort = "127.0.0.1:7833"
var Domain = "test-domain"
var TaskListName = "test-worker"
var ClientName = "test-worker"
var CadenceService = "cadence-frontend"

func main() {
    startWorker(buildLogger(), buildCadenceClient())
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        panic(err)
    }
}

func buildLogger() *zap.Logger {
    config := zap.NewDevelopmentConfig()
    config.Level.SetLevel(zapcore.InfoLevel)

    var err error
    logger, err := config.Build()
    if err != nil {
        panic("Failed to setup logger")
    }

    return logger
}

func buildCadenceClient() workflowserviceclient.Interface {
    dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: ClientName,
		Outbounds: yarpc.Outbounds{
		  CadenceService: {Unary: grpc.NewTransport().NewSingleOutbound(HostPort)},
		},
	  })
	  if err := dispatcher.Start(); err != nil {
		panic("Failed to start dispatcher")
	  }

	  clientConfig := dispatcher.ClientConfig(CadenceService)

	  return compatibility.NewThrift2ProtoAdapter(
		apiv1.NewDomainAPIYARPCClient(clientConfig),
		apiv1.NewWorkflowAPIYARPCClient(clientConfig),
		apiv1.NewWorkerAPIYARPCClient(clientConfig),
		apiv1.NewVisibilityAPIYARPCClient(clientConfig),
	  )
}

func startWorker(logger *zap.Logger, service workflowserviceclient.Interface) {
    // TaskListName identifies set of client workflows, activities, and workers.
    // It could be your group or client or application name.
    workerOptions := worker.Options{
        Logger:       logger,
        MetricsScope: tally.NewTestScope(TaskListName, map[string]string{}),
    }

    worker := worker.New(
        service,
        Domain,
        TaskListName,
        workerOptions)
    err := worker.Start()
    if err != nil {
        panic("Failed to start worker")
    }

    logger.Info("Started Worker.", zap.String("worker", TaskListName))
}

func helloWorldWorkflow(ctx workflow.Context, name string) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("helloworld workflow started")

	var helloworldResult string
	err := workflow.ExecuteActivity(ctx, helloWorldActivity, name).Get(ctx, &helloworldResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	var orderReceivedResult string
	err = workflow.ExecuteActivity(ctx, activities.OrderReceived, name).Get(ctx, &orderReceivedResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	var notificationResult string
	err = workflow.ExecuteActivity(ctx, activities.Notification, name).Get(ctx, &notificationResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	var deliveryAgentResult string
	err = workflow.ExecuteActivity(ctx, activities.DeliveryAgent, name).Get(ctx, &deliveryAgentResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	var orderReady string
	err = workflow.ExecuteActivity(ctx, activities.OrderReady, name).Get(ctx, &orderReady)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	var orderCollectedResult string
	err = workflow.ExecuteActivity(ctx, activities.OrderCollected, name).Get(ctx, &orderCollectedResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	var deliveryResult string
	err = workflow.ExecuteActivity(ctx, activities.Delivery, name).Get(ctx, &deliveryResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	logger.Info("Workflow completed.", zap.String("Result", helloworldResult))

	return nil
}

func helloWorldActivity(ctx context.Context, name string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("helloworld activity started")
	return "Hello " + name + "!", nil
}

func init() {
    workflow.Register(helloWorldWorkflow)
    activity.Register(helloWorldActivity)
    activity.Register(activities.OrderReceived)
    activity.Register(activities.Notification)
    activity.Register(activities.DeliveryAgent)
    activity.Register(activities.OrderReady)
    activity.Register(activities.OrderCollected)
    activity.Register(activities.Delivery)
}
