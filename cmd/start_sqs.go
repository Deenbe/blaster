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
	"github.com/spf13/cobra"
	"blaster/lib"
	log "github.com/sirupsen/logrus"
)

var queueName, region, httpHandlerURL string
var maxNumberOfMesages, waitTimeSeconds int64

// startSqsCmd represents the startSqs command
var startSQSCmd = &cobra.Command{
	Use:   "start-sqs",
	Short: "Start a message pump for an AWS sqs backend",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
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
		mp := lib.NewMessagePump(sqs, dispatcher)
		mp.Start()
		log.Info("Message pump started")
		err = <-mp.Done
	},
}

func init() {
	rootCmd.AddCommand(startSQSCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startSqsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startSqsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	startSQSCmd.Flags().StringVarP(&queueName, "queue-name", "q", "", "queue name")
	startSQSCmd.Flags().StringVarP(&region, "region", "r", "", "queue region")
	startSQSCmd.Flags().Int64VarP(&maxNumberOfMesages, "max-number-of-messages", "m", 1, "max number of messages to receive in a single poll")
	startSQSCmd.Flags().Int64VarP(&waitTimeSeconds, "wait-time-seconds", "w", 1, "wait time between polls")
	startSQSCmd.Flags().StringVarP(&httpHandlerURL, "target", "t", "", "target http handler url")
	startSQSCmd.MarkFlagRequired("queue-name")
	startSQSCmd.MarkFlagRequired("region")
	startSQSCmd.MarkFlagRequired("target")
}
