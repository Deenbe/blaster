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
	"blaster/kafka"

	"github.com/spf13/cobra"
)

var topic, group string
var brokerAddresses []string
var bufferSize int
var startFromOldest bool

// kafkaCmd represents the kafka command
var kafkaCmd = &cobra.Command{
	Use:   "kafka",
	Short: "Start blaster for a kafka backend",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		kafkaConfig := &kafka.KafkaConfig{
			Topic:           topic,
			Group:           group,
			BrokerAddresses: brokerAddresses,
			BufferSize:      bufferSize,
			StartFromOldest: startFromOldest,
		}

		config := GetConfig()
		binding, err := kafka.NewKafkaBinder(kafkaConfig, config)
		if err != nil {
			return err
		}

		core.RunCLIInstance(binding, config)
		return nil
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
	kafkaCmd.Flags().IntVar(&bufferSize, "buffer-size", 0, "number of messages to buffer")
	kafkaCmd.Flags().BoolVar(&startFromOldest, "start-from-oldest", false, "start consuming messages from the oldest offset")

	kafkaCmd.MarkFlagRequired("topic")
	kafkaCmd.MarkFlagRequired("group")
	kafkaCmd.MarkFlagRequired("brokers")
}
