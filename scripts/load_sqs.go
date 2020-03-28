// +build exclude

package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var region string
var queue string

func init() {
	flag.StringVar(&region, "region", "ap-southeast-2", "region")
	flag.StringVar(&queue, "queue", "", "queue name")
	flag.Parse()
}

func main() {
	if queue == "" {
		println("error: specify queue with -queue option")
		os.Exit(1)
	}

	for {
		load()
	}
}

func load() {
	wg := sync.WaitGroup{}
	for t := 0; t < 100; t++ {
		ses := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		}))

		wg.Add(1)
		go func(s *session.Session, batchNo int) {
			svc := sqs.New(s)
			urlResult, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
				QueueName: aws.String(queue),
			})
			if err != nil {
				println(err)
				println("healing...")
				time.Sleep(time.Second * 5)
			}
			batch := &sqs.SendMessageBatchInput{
				QueueUrl: urlResult.QueueUrl,
			}

			buff := make([]byte, 16)
			for i := 0; i < 10; i++ {
				rand.Read(buff)
				data := hex.EncodeToString(buff)
				batch.Entries = append(batch.Entries, &sqs.SendMessageBatchRequestEntry{
					Id:          aws.String(data),
					MessageBody: aws.String(fmt.Sprintf("hey %s", data)),
				})
			}
			_, err = svc.SendMessageBatch(batch)
			if err != nil {
				println(err)
				println("healing...")
				time.Sleep(time.Second * 5)
			}
			wg.Done()
			fmt.Printf("completed batch %d\n", batchNo)
		}(ses, t+1)
	}
	wg.Wait()
}
