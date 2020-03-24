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
	"blaster/lib"
	"time"

	"github.com/spf13/cobra"
)

var queueName, region, httpHandlerURL string
var maxNumberOfMesages, waitTimeSeconds, retryDelaySeconds int64
var retryCount int

// sqsCmd represents the sqs command
var sqsCmd = &cobra.Command{
	Use:   "sqs",
	Short: "Start a message pump for an AWS sqs backend",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := lib.SQSConfiguration{
			QueueName:           queueName,
			MaxNumberOfMessages: maxNumberOfMesages,
			WaitTime:            waitTimeSeconds,
			Region:              region,
		}

		sqs, err := lib.NewSQSService(&config)
		if err != nil {
			panic(err)
		}

		dispatcher := lib.NewHttpDispatcher(httpHandlerURL)
		mp := lib.NewMessagePump(sqs, dispatcher, retryCount, time.Second*time.Duration(retryDelaySeconds))
		return lib.StartTheSystem(mp)
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
	sqsCmd.Flags().StringVarP(&httpHandlerURL, "target", "t", "", "target http handler url")
	sqsCmd.Flags().Int64VarP(&maxNumberOfMesages, "max-number-of-messages", "m", 1, "max number of messages to receive in a single poll")
	sqsCmd.Flags().Int64VarP(&waitTimeSeconds, "wait-time-seconds", "w", 1, "wait time between polls")
	sqsCmd.Flags().IntVarP(&retryCount, "retry-count", "c", 0, "number of retry attempts")
	sqsCmd.Flags().Int64VarP(&retryDelaySeconds, "retry-delay-seconds", "d", 1, "delay between retry attempts")
	sqsCmd.MarkFlagRequired("queue-name")
	sqsCmd.MarkFlagRequired("region")
	sqsCmd.MarkFlagRequired("target")
}
