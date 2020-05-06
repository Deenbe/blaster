/*
Copyright © 2020 Blaster Contributors

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
	"blaster/kafka"

	"github.com/spf13/cobra"
)

var topic, group string
var brokerAddresses []string
var startFromOldest bool
var preCommitHookPath string
var commitIntervalSeconds int
var commitBatchSize int

// kafkaCmd represents the kafka command
var kafkaCmd = &cobra.Command{
	Use:   "kafka",
	Short: "Start blaster for a kafka backend",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		kafkaConfig := kafka.Config{
			Topic:                 topic,
			Group:                 group,
			BrokerAddresses:       brokerAddresses,
			StartFromOldest:       startFromOldest,
			CommitIntervalSeconds: commitIntervalSeconds,
			CommitBatchSize:       commitBatchSize,
			PreCommitHookPath:     preCommitHookPath,
		}

		config := GetConfig()
		return core.RunCLIInstance(&kafka.KafkaBinderBuilder{}, config, kafkaConfig)
	},
}

func init() {
	rootCmd.AddCommand(kafkaCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// kafkaCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// kafkaCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	kafkaCmd.Flags().StringVar(&topic, "topic", "", "topic name")
	kafkaCmd.Flags().StringVar(&group, "group", "", "group name")
	kafkaCmd.Flags().StringArrayVar(&brokerAddresses, "brokers", []string{}, "broker addresses")
	kafkaCmd.Flags().IntVar(&commitIntervalSeconds, "commit-interval-seconds", kafka.DefaultCommitIntervalSeconds, "delay before automatically committing the offset of the last message")
	kafkaCmd.Flags().IntVar(&commitBatchSize, "commit-batch-size", 0, "number of messages to batch before committing")
	kafkaCmd.Flags().StringVar(&preCommitHookPath, "pre-commit-hook-path", "", "pre commit hook path")
	kafkaCmd.Flags().BoolVar(&startFromOldest, "start-from-oldest", false, "start consuming messages from the oldest offset")

	kafkaCmd.MarkFlagRequired("topic")
	kafkaCmd.MarkFlagRequired("group")
	kafkaCmd.MarkFlagRequired("brokers")
}
