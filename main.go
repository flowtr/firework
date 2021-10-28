package main

import (
	"os"
	"strings"
	"time"

	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type VM struct {
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
}

type Step struct {
	Name  string `yaml:"name"`
	VM    string `yaml:"vm"`
	Notes string `yaml:"notes"`
	Cmd   string `yaml:"cmd"`
}

type Config struct {
	VMs  []VM   `yaml:"vms"`
	Step []Step `yaml:"steps"`
}

func main() {
	var config Config
	var yamlFile string

	rootCmd := &cobra.Command{
		Use:   "fdpl",
		Short: "fdpl",
		Long:  `flowtr-deploy`,
		Run: func(cmd *cobra.Command, args []string) {
			if yamlFile == "" {
				logrus.Fatal("Please provide a yaml file")
			}

			fileContents, err := os.ReadFile(yamlFile)
			if err != nil {
				logrus.Fatal(err)
			}

			if err := yaml.Unmarshal([]byte(fileContents), &config); err != nil {
				logrus.Fatal(err)
			}

			for _, vm := range config.VMs {
				// Deploy vm with weaveworks ignite
				subProcess := exec.Command(
					"ignite",
					"create",
					vm.Image,
					"--name",
					vm.Name,
					// TODO: add more config options
					"--cpus",
					"2",
					"--memory",
					"4GB",
					"--size",
					"25GB",
					"--ssh",
				)

				subProcess.Stdout = os.Stdout
				subProcess.Stderr = os.Stderr

				if err := subProcess.Run(); err != nil {
					logrus.Fatal(err)
				}

				// start the vm
				subProcess = exec.Command(
					"ignite",
					"start",
					vm.Name,
				)

				subProcess.Stdout = os.Stdout
				subProcess.Stderr = os.Stderr

				if err := subProcess.Run(); err != nil {
					logrus.Fatal(err)
				}

				// Wait for vm to be ready
				logrus.Infof("Waiting for vm %s to be ready", vm.Name)
				time.Sleep(time.Second * 3)

				for _, step := range config.Step {
					logrus.WithFields(logrus.Fields{
						"step": step.Name,
						"vm":   step.VM,
					}).Info("Running step")

					for _, note := range strings.Split(step.Notes, "\n") {
						logrus.Info(note)
					}

					// get vm with name
					if vm.Name == step.VM {
						logrus.WithFields(logrus.Fields{
							"step":     step.Name,
							"vm":       vm.Name,
							"commands": step.Cmd,
						}).Info("Running commands on vm")

						// run the commands
						for _, command := range strings.Split(step.Cmd, "\n") {
							subProcess = exec.Command(
								"ignite",
								"exec",
								vm.Name,
								command,
							)

							subProcess.Stdout = os.Stdout
							subProcess.Stderr = os.Stderr

							if err := subProcess.Run(); err != nil {
								logrus.Fatal(err)
							}
						}

						logrus.WithFields(logrus.Fields{
							"status": subProcess.ProcessState.ExitCode(),
							"name":   vm.Name,
						}).Info("Deployed commands on vm")
					}
				}
			}
		},
	}

	rootCmd.PersistentFlags().StringVarP(&yamlFile, "config", "c", "", "yaml config file")
	rootCmd.MarkPersistentFlagRequired("config")
	rootCmd.Execute()
}
