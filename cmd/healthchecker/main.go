package main

import (
	"bytes"
	"errors"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

type request struct {
	Endpoint string
	Method   string
	Header   map[string]string
	Body     string
}

type header struct {
	Key   string
	Value string
}

type ok struct {
	Code   int
	Header header
}

type response struct {
	Ok ok
}

type check struct {
	Service  string
	Request  request
	Response response
}

type healthCheckConfig struct {
	Interval time.Duration
	Notify   request
	Checks   []check
}

type errorReport struct {
	FailingService            string
	FailingServiceDescription error
}

func getConfigFilepathFromArgs() (string, error) {

	flag.Parse()

	if flag.NArg() != 1 {
		return "", errors.New("missing argument: config filepath")
	}

	return flag.Arg(0), nil
}

func getConfig(filepath string) (*healthCheckConfig, error) {

	var config healthCheckConfig

	yamlFile, readError := ioutil.ReadFile(filepath)
	if readError != nil {
		return nil, readError
	}

	yamlError := yaml.Unmarshal(yamlFile, &config)
	if yamlError != nil {
		return nil, yamlError
	}

	return &config, nil
}

func callService(modelRequest *request, headerName string) (*response, error) {

	httpRequest, _ := http.NewRequest(
		modelRequest.Method,
		modelRequest.Endpoint,
		strings.NewReader(modelRequest.Body))

	for key, value := range modelRequest.Header {
		httpRequest.Header.Set(key, value)
	}

	httpResponse, err := (&http.Client{}).Do(httpRequest)

	var actualResponse response
	if err == nil {
		actualResponse.Ok.Code = httpResponse.StatusCode
		actualResponse.Ok.Header.Key = headerName
		actualResponse.Ok.Header.Value = httpResponse.Header.Get(headerName)
	}

	return &actualResponse, err
}

func responsesMatch(modelResponse *response, actualResponse *response) bool {

	match := true

	if modelResponse.Ok.Code != 0 {
		match = modelResponse.Ok.Code == actualResponse.Ok.Code
	}

	if modelResponse.Ok.Header.Key == actualResponse.Ok.Header.Key {
		match = match && modelResponse.Ok.Header.Value == actualResponse.Ok.Header.Value
	}

	return match
}

func replaceVariables(text string, data interface{}) string {

	templ, _ := template.New("msg").Parse(text)

	var buffer bytes.Buffer
	templateError := templ.Execute(&buffer, data)

	if templateError != nil {
		return text
	}

	return buffer.String()
}

func notifyError(notification *request, service *check, actualResponse *response, err error) {

	log.Printf(
		"ERROR: %s, Expected vs Actual responses:\n\t%+v\n\t%+v\n\tERROR: %s\n",
		service.Service,
		service.Response,
		actualResponse,
		err)

	notification.Body = replaceVariables(notification.Body, errorReport{service.Service, err})

	notifyResponse, notifyError := callService(notification, "")

	if notifyError != nil || notifyResponse.Ok.Code != 200 {
		log.Printf("ERROR: notification error %s\n\t%+v\n", notifyError, notifyResponse)
	}
}

func checkServiceRoutine(service check, notification request, waitGroup *sync.WaitGroup) {

	log.Printf("Calling service: %s...\n", service.Service)

	actualResponse, err := callService(&service.Request, service.Response.Ok.Header.Key)

	if err != nil {
		notifyError(&notification, &service, actualResponse, err)
	} else if !responsesMatch(&service.Response, actualResponse) {
		notifyError(&notification, &service, actualResponse, errors.New("responses do not match"))
	} else {
		log.Printf("SUCCESS: %s\n", service.Service)
	}

	waitGroup.Done()
}

func checkAllServices(config *healthCheckConfig) {

	var waitGroup sync.WaitGroup

	waitGroup.Add(len(config.Checks))

	for _, service := range config.Checks {
		go checkServiceRoutine(service, config.Notify, &waitGroup)
	}

	waitGroup.Wait()
}

func startCheckerRoutine(config *healthCheckConfig) *time.Ticker {

	ticker := time.NewTicker(config.Interval * time.Millisecond)

	go func() {

		for ; true; <-ticker.C {

			log.Print("Starting loop...\n\n")

			checkAllServices(config)

			log.Print("Loop finished.\n\n")
		}
	}()

	return ticker
}

func stopOnInterruptSignal(ticker *time.Ticker) {

	termChannel := make(chan os.Signal, 1)
	signal.Notify(termChannel, os.Interrupt, syscall.SIGTERM)

	<-termChannel

	close(termChannel)

	log.Println("Interrupt signal received.")

	ticker.Stop()
}

func main() {

	configFilepath, argsError := getConfigFilepathFromArgs()
	if argsError != nil {
		log.Fatal(argsError)
	}

	config, fileError := getConfig(configFilepath)
	if fileError != nil {
		log.Fatal(fileError)
	}

	ticker := startCheckerRoutine(config)

	stopOnInterruptSignal(ticker)
}
