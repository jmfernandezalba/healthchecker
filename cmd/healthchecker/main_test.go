package main

import (
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"
)

func assertEqual(t *testing.T, actual interface{}, expected interface{}) {

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%+v != %+v", actual, expected)
	}
}

func TestGetConfigFilepathFromArgsOk(t *testing.T) {

	//Arrange
	expectedFilename := "exampleFilepath"
	//-Prepare
	os.Args = []string{"healthchecker", expectedFilename}

	//Act
	configFilepath, argsError := getConfigFilepathFromArgs()

	//Assert
	assertEqual(t, configFilepath, expectedFilename)
	if argsError != nil {
		t.Errorf("error is not nil")
	}
}

func TestGetConfigFilepathFromArgsWrong(t *testing.T) {

	//Arrange
	//-Prepare
	os.Args = []string{"healthchecker"}

	//Act
	configFilepath, argsError := getConfigFilepathFromArgs()

	//Assert
	assertEqual(t, configFilepath, "")
	if argsError == nil {
		t.Errorf("error is nil")
	}
}

func TestGetConfigOk(t *testing.T) {

	//Arrange
	expectedConfig := &healthCheckConfig{
		Interval: 5000,
		Notify: request{
			Endpoint: "https://interview-notifier-svc.spotahome.net/api/v1/notification",
			Method:   "POST",
			Header: map[string]string{
				"authorization": "Bearer XXXXXXXXXXXXXXXXXXXX",
			},
			Body: "{\"service\": \"{{.FailingService}}\", \"description\": \"{{.FailingServiceDescription}}\"}",
		},
		Checks: []check{
			{
				Service: "myapp",
				Request: request{
					Endpoint: "http://myapp.com/check",
					Method:   "GET",
				},
				Response: response{
					Ok: ok{
						Code: 200,
					},
				},
			},
			{
				Service: "otherapp-euwest1",
				Request: request{
					Endpoint: "https://euwest1.otherapp.io/checks",
					Method:   "POST",
					Body:     "{\"checker\": \"spotahome\"}",
				},
				Response: response{
					Ok: ok{
						Header: header{
							Key:   "healthcheck",
							Value: "ok",
						},
					},
				},
			},
			{
				Service: "important-service",
				Request: request{
					Endpoint: "http://awesome-teapot.io:18976/healthz/live",
					Method:   "GET",
				},
				Response: response{
					Ok: ok{
						Code: 418,
					},
				},
			},
		},
	}

	//Act
	config, fileError := getConfig("testdata/unitTestingHealthCheckConfig.yaml")

	//Assert
	assertEqual(t, config, expectedConfig)
	if fileError != nil {
		t.Errorf("error is not nil")
	}
}

func TestGetConfigNonExistentFile(t *testing.T) {

	//Arrange
	filename := "testdata/nonExistentHealthCheckConfig.yaml"

	//Act
	config, fileError := getConfig(filename)

	//Assert
	if config != nil {
		t.Errorf("config is not nil")
	}
	if fileError == nil {
		t.Errorf("error is nil")
	}
}

func TestGetConfigMalformedFile(t *testing.T) {

	//Arrange
	filename := "testdata/malformedHealthCheckConfig.yaml"

	//Act
	config, fileError := getConfig(filename)

	//Assert
	if config != nil {
		t.Errorf("config is not nil")
	}
	if fileError == nil {
		t.Errorf("error is nil")
	}
}

func TestCallServiceOk(t *testing.T) {

	//Arrange
	modelRequest := &request{
		Endpoint: "https://www.google.com",
	}
	expectedResponse := ok{
		Code: 200,
	}

	//Act
	actualResponse, err := callService(modelRequest, "")

	//Assert
	assertEqual(t, actualResponse.Ok, expectedResponse)
	if err != nil {
		t.Errorf("error is not nil")
	}
}

func TestResponsesMatchOk(t *testing.T) {

	//Arrange
	responseA := &response{
		Ok: ok{
			Code: 200,
		},
	}
	responseB := &response{
		Ok: ok{
			Code: 200,
		},
	}

	//Act
	match := responsesMatch(responseA, responseB)

	//Assert
	assertEqual(t, match, true)
}

func TestReplaceVariablesOk(t *testing.T) {

	//Arrange
	text := "template with {{.FailingService}} in it"
	data := errorReport{
		FailingService: "example",
	}

	expectedResult := "template with example in it"

	//Act
	actualResult := replaceVariables(text, data)

	//Assert
	assertEqual(t, actualResult, expectedResult)
}

func TestReplaceVariablesWrong(t *testing.T) {

	//Arrange
	text := "template with {{.FailingServiceWrong}} in it"
	data := errorReport{
		FailingService: "example",
	}

	expectedResult := "template with {{.FailingServiceWrong}} in it"

	//Act
	actualResult := replaceVariables(text, data)

	//Assert
	assertEqual(t, actualResult, expectedResult)
}

func TestNotifyErrorOk(t *testing.T) {

	//Arrange
	notification := &request{
		Endpoint: "https://www.google.com",
	}
	service := &check{
		Service: "myservice",
		Request: request{
			Endpoint: "https://www.google.com",
		},
	}
	actualResponse := &response{
		Ok: ok{
			Code: 200,
		},
	}

	//Act
	notifyError(notification, service, actualResponse, errors.New("testing"))
}

func TestCheckServiceRoutineOk(t *testing.T) {

	//Arrange
	notification := request{
		Endpoint: "https://www.google.com",
	}
	service := check{
		Service: "myservice",
		Request: request{
			Endpoint: "https://www.google.com",
		},
	}
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	//Act
	checkServiceRoutine(service, notification, &waitGroup)
}

func TestCheckServiceRoutineError(t *testing.T) {

	//Arrange
	notification := request{
		Endpoint: "https://www.google.com",
	}
	service := check{
		Service: "myservice",
		Request: request{
			Endpoint: "https://euwest1.otherapp.io/checks",
		},
	}
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	//Act
	checkServiceRoutine(service, notification, &waitGroup)
}

func TestCheckServiceRoutineWrongResponse(t *testing.T) {

	//Arrange
	notification := request{
		Endpoint: "https://www.google.com",
	}
	service := check{
		Service: "myservice",
		Request: request{
			Endpoint: "https://www.google.com",
		},
		Response: response{
			Ok: ok{
				Code: 300,
			},
		},
	}
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	//Act
	checkServiceRoutine(service, notification, &waitGroup)
}
