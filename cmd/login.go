package cmd

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
)


func init() {
	rootCommand.AddCommand(registryLoginCommand)
	registryLoginCommand.Flags().Bool("ecr", false, "If we are using ECR we need a different set of flags, so if this is set, we need to set --username and --password-stdin")
	registryLoginCommand.Flags().String("registry-url", "https://index.docker.io/v1", "The Registry URL, it is defaulted to the docker registry")
	registryLoginCommand.Flags().String("username", "", "The Registry Username")
	registryLoginCommand.Flags().String("password-stdin", "", "The registry password gets passed as stdin, so its not left in the history")
	registryLoginCommand.Flags().String("account-id", "", "Your AWS Account id")
	registryLoginCommand.Flags().String("region", "us-west-1", "Your AWS region")
}

var registryLoginCommand = &cobra.Command{
	Use:          "login",
	Short:        "Generate and save the registry authentication file",
	SilenceUsage: true,
	RunE:         generateRegistryAuthFile,
}

func generateRegistryAuthFile(command *cobra.Command, _ []string) error {

	command.RunE = func(cmd *cobra.Command, args []string) error {
		ecrEnabled, _ := command.Flags().GetBool("ecr")
		if ecrEnabled {

			accountID, _ := command.Flags().GetString("account-id")
			region, _ := command.Flags().GetString("region")

			if len(accountID) == 0 {
				return errors.New("you must provide an --account-id value when using --ecr")
			}
			fileBytes, err := generateECRRegistryAuth(accountID, region)
			if err != nil {
				return err
			}

			writeErr := writeFileToOFCTmp(fileBytes)

			return writeErr

		} else {
			username, _ := command.Flags().GetString("username")
			password, _ := command.Flags().GetString("password-stdin")
			registryURL, _ := command.Flags().GetString("registry-url")

			if len(username) == 0 || len(password) == 0 {
				return errors.New("both --username and --password-stdin must be used, and provided, for us to generate a valid file")
			}

			fileBytes, err := generateRegistryAuth(registryURL, username, password)
			if err != nil {
				return err
			}

			writeErr := writeFileToOFCTmp(fileBytes)

			return writeErr
		}
	}
	return nil
}

func generateRegistryAuth(registryURL, username, password string) ([]byte, error) {
	encodedString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	data := RegistryAuths{
		AuthConfigs: map[string]Auth{
			registryURL : {Base64AuthString:encodedString},
		},
	}

	file, err := json.MarshalIndent(data, "", " ")

	return file, err
}

func generateECRRegistryAuth(accountID, region string) ([]byte, error) {
	data := ECRRegistryAuth{
		CredsStore: "ecr-registryLogin",
		CredHelpers: map[string]string {
			fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", accountID, region): "ecr-registryLogin",
		},

	}

	file, err := json.MarshalIndent(data, "", " ")

	return file, err
}

func writeFileToOFCTmp(fileBytes []byte) error {
	path := "./credentials"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0744)
		if err != nil {
			return err
		}
	}

	writeErr := ioutil.WriteFile(filepath.Join(path, "config.json"), fileBytes, 0744)

	return writeErr

}

type Auth struct {
	Base64AuthString string `json:"auth"`
}

type RegistryAuths struct {
	AuthConfigs map[string]Auth `json:"auths"`
}

type ECRRegistryAuth struct {
	CredsStore  string      `json:"credsStore"`
	CredHelpers map[string]string `json:"credHelpers"`
}
