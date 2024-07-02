package tests

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var opts = godog.Options{
	Output:      colors.Colored(os.Stdout),
	Concurrency: 4,
}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opts)
}

func TestFeatures(t *testing.T) {
	o := opts
	o.TestingT = t

	status := godog.TestSuite{
		Name:                 "godogs",
		Options:              &o,
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
	}.Run()

	if status == 2 {
		t.SkipNow()
	}

	if status != 0 {
		t.Fatalf("zero status code expected, %d received", status)
	}
}

func iDownloadTheFile() error {
	return godog.ErrPending
}

func iHaveReceivedADownloadEventForASmallFile() error {
	return godog.ErrPending
}

func theFileShouldBeDownloadedAsASingleFragment() error {
	return godog.ErrPending
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^I download the file$`, iDownloadTheFile)
	ctx.Step(`^I have received a download event for a small file$`, iHaveReceivedADownloadEventForASmallFile)
	ctx.Step(`^the file should be downloaded as a single fragment$`, theFileShouldBeDownloadedAsASingleFragment)
}

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() { fmt.Println("Get the party started!") })
}
