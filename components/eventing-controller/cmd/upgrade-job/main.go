package main

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/upgrade-job/clients/deployment"
	eventmesh "github.com/kyma-project/kyma/components/eventing-controller/upgrade-job/clients/event-mesh"
	eventingbackend "github.com/kyma-project/kyma/components/eventing-controller/upgrade-job/clients/eventing-backend"
	"github.com/kyma-project/kyma/components/eventing-controller/upgrade-job/clients/secret"
	"github.com/kyma-project/kyma/components/eventing-controller/upgrade-job/clients/subscription"
	jobprocess "github.com/kyma-project/kyma/components/eventing-controller/upgrade-job/process"
	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"
)

type Config struct {
	ReleaseName     string `envconfig:"RELEASE" required:"true"`
	KymaNamespace    string `envconfig:"KYMA_NAMESPACE" default:"kyma-system"`
	EventingControllerName string `envconfig:"EVENTING_CONTROLLER_NAME" required:"true"`
	EventingPublisherName string `envconfig:"EVENTING_PUBLISHER_NAME" required:"true"`
	LogFormat string `envconfig:"APP_LOG_FORMAT" default:"json"`
	LogLevel  string `envconfig:"APP_LOG_LEVEL" default:"warn"`
}

func main() {
	// Env vars
	cfg := new(Config)
	if err := envconfig.Process("", cfg); err != nil {
		//logger.Fatalf("Start handler failed with error: %s", err)
		panic(err)
	}

	// Create logger instance
	ctrLogger, err := logger.New(cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		panic(errors.Wrapf(err, "failed to initialize logger"))
		//os.Exit(1)
	}
	defer func() {
		if err := ctrLogger.WithContext().Sync(); err != nil {
			panic(errors.Wrapf(err, "failed to flush logger"))
		}
	}()


	// Generate dynamic clients
	k8sConfig := config.GetConfigOrDie()

	// Create dynamic client
	dynamicClient := dynamic.NewForConfigOrDie(k8sConfig)

	// setup clients
	deploymentClient := deployment.NewClient(dynamicClient)
	subscriptionClient := subscription.NewClient(dynamicClient)
	eventingBackendClient := eventingbackend.NewClient(dynamicClient)
	secretClient := secret.NewClient(dynamicClient)
	eventMeshClient := eventmesh.NewClient()

	// Create process
	p := jobprocess.Process{
		Logger: ctrLogger.Logger,
		TimeoutPeriod: 60 * time.Second,
		ReleaseName:  cfg.ReleaseName,
		KymaNamespace: cfg.KymaNamespace,
		ControllerName: cfg.EventingControllerName,
		PublisherName: cfg.EventingPublisherName,
		Clients: jobprocess.Clients{
			Deployment: deploymentClient,
			Subscription: subscriptionClient,
			EventingBackend: eventingBackendClient,
			Secret: secretClient,
			EventMesh: eventMeshClient,
		},
	}

	// Add steps to process
	p.AddSteps()

	// Execute process
	err = p.Execute()
	if err != nil {
		ctrLogger.Logger.WithContext().Error(err)
	}

	ctrLogger.Logger.WithContext().Info("Completed upgrade-hook main 1.24.x")
}
