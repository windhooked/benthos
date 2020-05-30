package lambda

import (
	"fmt"
	"os"
	"time"

	"github.com/windhooked/benthos/v3/lib/config"
	"github.com/windhooked/benthos/v3/lib/serverless"
	"github.com/windhooked/benthos/v3/lib/util/text"
	"github.com/aws/aws-lambda-go/lambda"
	"gopkg.in/yaml.v3"
)

var handler *serverless.Handler

// Run executes Benthos as an AWS Lambda function. Configuration can be stored
// within the environment variable BENTHOS_CONFIG.
func Run() {
	// A list of default config paths to check for if not explicitly defined
	defaultPaths := []string{
		"/benthos.yaml",
		"/etc/benthos/config.yaml",
		"/etc/benthos.yaml",
	}

	conf := config.New()
	conf.Output.Type = serverless.ServerlessResponseType

	if confStr := os.Getenv("BENTHOS_CONFIG"); len(confStr) > 0 {
		confBytes := text.ReplaceEnvVariables([]byte(confStr))
		if err := yaml.Unmarshal(confBytes, &conf); err != nil {
			fmt.Fprintf(os.Stderr, "Configuration file read error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Iterate default config paths
		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				if _, err = config.Read(path, true, &conf); err != nil {
					fmt.Fprintf(os.Stderr, "Configuration file read error: %v\n", err)
					os.Exit(1)
				}
				break
			}
		}
	}

	var err error
	if handler, err = serverless.NewHandler(conf); err != nil {
		fmt.Fprintf(os.Stderr, "Initialisation error: %v\n", err)
		os.Exit(1)
	}

	lambda.Start(handler.Handle)
	if err = handler.Close(time.Second * 30); err != nil {
		fmt.Fprintf(os.Stderr, "Shut down error: %v\n", err)
		os.Exit(1)
	}
}
