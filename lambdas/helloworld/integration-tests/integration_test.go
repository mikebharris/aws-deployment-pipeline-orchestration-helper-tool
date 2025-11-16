package integration_tests

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/mikebharris/testcontainernetwork-go"
	"github.com/stretchr/testify/assert"

	"github.com/cucumber/godog"
)

func TestFeatures(t *testing.T) {
	var steps steps
	steps.t = t

	suite := godog.TestSuite{
		TestSuiteInitializer: func(ctx *godog.TestSuiteContext) {
			ctx.BeforeSuite(steps.startContainerNetwork)
			ctx.AfterSuite(steps.stopContainerNetwork)
		},
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^the Lambda is deployed$`, func() {})
			ctx.Step(`^a request is made to the API endpoint$`, steps.aRequestIsMadeToTheApiEndpoint)
			ctx.Step(`^a greeting is returned$`, steps.aGreetingIsReturned)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t, // TestingT is needed to run godog tests with "go test"
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

type steps struct {
	t                   *testing.T
	networkOfContainers testcontainernetwork.NetworkOfDockerContainers
	lambdaContainer     testcontainernetwork.LambdaDockerContainer
}

var responseFromLambda events.APIGatewayProxyResponse

func (s *steps) startContainerNetwork() {
	s.lambdaContainer = testcontainernetwork.LambdaDockerContainer{
		Config: testcontainernetwork.LambdaDockerContainerConfig{
			Hostname:    "lambda",
			Executable:  "../main",
			Environment: map[string]string{},
		},
	}

	s.networkOfContainers =
		testcontainernetwork.NetworkOfDockerContainers{}.
			WithDockerContainer(&s.lambdaContainer)
	_ = s.networkOfContainers.StartWithDelay(5 * time.Second)
}

func (s *steps) stopContainerNetwork() {
	if err := s.networkOfContainers.Stop(); err != nil {
		log.Fatalf("stopping docker containers: %v", err)
	}
}

func (s *steps) aRequestIsMadeToTheApiEndpoint() {
	response, err := http.Post(s.lambdaContainer.InvocationUrl(), "application/json", nil)
	if err != nil {
		log.Fatalf("triggering lambda: %v", err)
	}

	if response.StatusCode != 200 {
		log.Fatalf("invoking Lambda: %d", response.StatusCode)
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(response.Body); err != nil {
		log.Fatalf("reading response body: %v", err)
	}

	if err := json.Unmarshal(buf.Bytes(), &responseFromLambda); err != nil {
		log.Fatalf("unmarshalling response: %v", err)
	}
}

func (s *steps) aGreetingIsReturned() error {
	assert.Equal(s.t, 200, responseFromLambda.StatusCode)
	assert.Equal(s.t, "application/json", responseFromLambda.Headers["Content-Type"])
	assert.Equal(s.t, `{"message":"Hello World!"}`, responseFromLambda.Body)

	return nil
}
