/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"blaster/core"
	"blaster/sqs"

	"github.com/spf13/cobra"
)

var queueName, region string
var waitTimeSeconds int64
var maxNumberOfMesages int64

// sqsCmd represents the sqs command
var sqsCmd = &cobra.Command{
	Use:   "sqs",
	Short: "Start blaster for an AWS sqs backend",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		sqsConfig := &sqs.SQSConfiguration{
			QueueName:           queueName,
			MaxNumberOfMessages: maxNumberOfMesages,
			WaitTime:            waitTimeSeconds,
			Region:              region,
		}

		config := GetConfig()
		binding := sqs.NewSQSBinder(sqsConfig, config)
		return core.RunCLIInstance(binding, config)
	},
}

func init() {
	rootCmd.AddCommand(sqsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startSqsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startSqsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	sqsCmd.Flags().StringVarP(&queueName, "queue-name", "q", "", "queue name")
	sqsCmd.Flags().StringVarP(&region, "region", "r", "", "queue region")
	sqsCmd.Flags().Int64VarP(&waitTimeSeconds, "wait-time-seconds", "w", 1, "wait time between polls")
	sqsCmd.Flags().Int64VarP(&maxNumberOfMesages, "max-number-of-messages", "m", 1, "max number of messages to receive in a single poll")
	sqsCmd.MarkFlagRequired("queue-name")
	sqsCmd.MarkFlagRequired("region")
}
