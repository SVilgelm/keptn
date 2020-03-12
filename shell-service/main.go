package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	cloudeventshttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/types"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"

	configutils "github.com/keptn/go-utils/pkg/configuration-service/utils"
	keptnevents "github.com/keptn/go-utils/pkg/events"
	keptnutils "github.com/keptn/go-utils/pkg/utils"
)

const (
	eventbroker = "EVENTBROKER"
	scriptName  = "tests/run.sh"
)

type envConfig struct {
	// Port on which to listen for cloudevents
	Port int    `envconfig:"RCV_PORT" default:"8080"`
	Path string `envconfig:"RCV_PATH" default:"/"`
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("Failed to process env var: %s", err)
	}
	os.Exit(_main(os.Args[1:], env))
}

func gotEvent(ctx context.Context, event cloudevents.Event) error {
	var shkeptncontext string
	event.Context.ExtensionAs("shkeptncontext", &shkeptncontext)

	logger := keptnutils.NewLogger(shkeptncontext, event.Context.GetID(), "shell-service")

	data := &keptnevents.DeploymentFinishedEventData{}
	if err := event.DataAs(data); err != nil {
		logger.Error(fmt.Sprintf("Got Data Error: %s", err.Error()))
		return err
	}

	if event.Type() != keptnevents.DeploymentFinishedEventType {
		const errorMsg = "Received unexpected keptn event"
		logger.Error(errorMsg)
		return errors.New(errorMsg)
	}

	if data.TestStrategy == "real-user" {
		logger.Info("Received 'real-user' test strategy, hence no tests are triggered")
		return nil
	}
	go runShellTest(event, shkeptncontext, *data, logger)

	return nil
}

func runShellTest(event cloudevents.Event, shkeptncontext string, data keptnevents.DeploymentFinishedEventData, logger *keptnutils.Logger) {
	testInfo := getTestInfo(data)
	id := uuid.New().String()
	startedAt := time.Now()

	var res bool
	var err error
	res, err = runTest(data, id, logger)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	if !res {
		if err := sendTestsFinishedEvent(shkeptncontext, event, startedAt, "fail", logger); err != nil {
			logger.Error(fmt.Sprintf("Error sending test finished event: %s", err.Error()) + ". " + testInfo.ToString())
		}
		return
	}
	logger.Info("Shell test passed = " + strconv.FormatBool(res) + ". " + testInfo.ToString())
	if err := sendTestsFinishedEvent(shkeptncontext, event, startedAt, "pass", logger); err != nil {
		logger.Error(fmt.Sprintf("Error sending test finished event: %s", err.Error()) + ". " + testInfo.ToString())
	}
}

// TestInfo contains information about which test to execute
type TestInfo struct {
	Project      string
	Stage        string
	Service      string
	TestStrategy string
}

// ToString returns a string representation of a TestInfo object
func (ti *TestInfo) ToString() string {
	return "Project: " + ti.Project + ", Service: " + ti.Service + ", Stage: " + ti.Stage + ", TestStrategy: " + ti.TestStrategy
}

func getTestInfo(data keptnevents.DeploymentFinishedEventData) *TestInfo {
	return &TestInfo{
		Project:      data.Project,
		Service:      data.Service,
		Stage:        data.Stage,
		TestStrategy: data.TestStrategy,
	}
}

func getConfigurationServiceURL() string {
	if os.Getenv("env") == "production" {
		return "configuration-service:8080"
	}
	return "localhost:8080"
}

func getServiceURL(data keptnevents.DeploymentFinishedEventData) string {
	serviceURL := data.Service + "." + data.Project + "-" + data.Stage
	if data.DeploymentStrategy == "blue_green_service" {
		serviceURL = data.Service + "-canary" + "." + data.Project + "-" + data.Stage
	}
	return serviceURL
}

func runTest(data keptnevents.DeploymentFinishedEventData, id string, logger *keptnutils.Logger) (bool, error) {
	testInfo := getTestInfo(data)
	resourceHandler := configutils.NewResourceHandler(getConfigurationServiceURL())
	testScriptResource, err := resourceHandler.GetServiceResource(testInfo.Project, testInfo.Stage, testInfo.Service, scriptName)

	// if no test file has been found, we assume that no tests should be executed
	if err != nil || testScriptResource == nil || testScriptResource.ResourceContent == "" {
		logger.Debug("Skipping test execution because no tests have been defined.")
		return true, nil
	}

	tmpfile, err := ioutil.TempFile("", "test_*.sh")
	if err != nil {
		logger.Error(err.Error())
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(testScriptResource.ResourceContent)); err != nil {
		logger.Error(err.Error())
	}
	if err := tmpfile.Close(); err != nil {
		logger.Error(err.Error())
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	testInfoStr := testInfo.ToString() + ", scriptName: " + scriptName + ", serverURL: " + getServiceURL(data)
	logger.Debug("Starting Shell test. " + testInfoStr)
	res, err := keptnutils.ExecuteCommand(shell, []string{"-c", tmpfile.Name()})

	logger.Info(res)
	if err != nil {
		logger.Error(err.Error())
		return false, err
	}
	return true, nil
}

func sendTestsFinishedEvent(shkeptncontext string, incomingEvent cloudevents.Event, startedAt time.Time, result string, logger *keptnutils.Logger) error {
	source, _ := url.Parse("shell-service")
	contentType := "application/json"

	testFinishedData := keptnevents.TestsFinishedEventData{}
	// fill in data from incoming event (e.g., project, service, stage, teststrategy, deploymentstrategy)
	if err := incomingEvent.DataAs(&testFinishedData); err != nil {
		logger.Error(fmt.Sprintf("Got Data Error: %s", err.Error()))
		return err
	}
	// fill in timestamps
	testFinishedData.Start = startedAt.Format(time.RFC3339)
	testFinishedData.End = time.Now().Format(time.RFC3339)
	// set test result
	testFinishedData.Result = result

	event := cloudevents.Event{
		Context: cloudevents.EventContextV02{
			ID:          uuid.New().String(),
			Time:        &types.Timestamp{Time: time.Now()},
			Type:        keptnevents.TestsFinishedEventType,
			Source:      types.URLRef{URL: *source},
			ContentType: &contentType,
			Extensions:  map[string]interface{}{"shkeptncontext": shkeptncontext},
		}.AsV02(),
		Data: testFinishedData,
	}

	return sendEvent(event)
}

func _main(args []string, env envConfig) int {

	ctx := context.Background()

	t, err := cloudeventshttp.New(
		cloudeventshttp.WithPort(env.Port),
		cloudeventshttp.WithPath(env.Path),
	)

	if err != nil {
		log.Fatalf("failed to create transport, %v", err)
	}
	c, err := client.New(t)
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	log.Println("Starting reciver")
	log.Fatalf("failed to start receiver: %s", c.StartReceiver(ctx, gotEvent))

	return 0
}

func sendEvent(event cloudevents.Event) error {
	endPoint, err := getServiceEndpoint(eventbroker)
	if err != nil {
		return errors.New("Failed to retrieve endpoint of eventbroker. %s" + err.Error())
	}

	if endPoint.Host == "" {
		return errors.New("Host of eventbroker not set")
	}

	transport, err := cloudeventshttp.New(
		cloudeventshttp.WithTarget(endPoint.String()),
		cloudeventshttp.WithEncoding(cloudeventshttp.StructuredV02),
	)
	if err != nil {
		return errors.New("Failed to create transport:" + err.Error())
	}

	c, err := client.New(transport)
	if err != nil {
		return errors.New("Failed to create HTTP client:" + err.Error())
	}

	if _, err := c.Send(context.Background(), event); err != nil {
		return errors.New("Failed to send cloudevent:, " + err.Error())
	}
	return nil
}

// getServiceEndpoint gets an endpoint stored in an environment variable and sets http as default scheme
func getServiceEndpoint(service string) (url.URL, error) {
	url, err := url.Parse(os.Getenv(service))
	if err != nil {
		return *url, fmt.Errorf("Failed to retrieve value from ENVIRONMENT_VARIABLE: %s", service)
	}

	if url.Scheme == "" {
		url.Scheme = "http"
	}

	return *url, nil
}
