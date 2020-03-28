// +build exclude

package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"sync"

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
				panic(err)
			}
			batch := &sqs.SendMessageBatchInput{
				QueueUrl: urlResult.QueueUrl,
			}

			buff := make([]byte, 16)
			for i := 0; i < 10; i++ {
				rand.Read(buff)
				batch.Entries = append(batch.Entries, &sqs.SendMessageBatchRequestEntry{
					Id:          aws.String(hex.EncodeToString(buff)),
					MessageBody: aws.String(fmt.Sprintf("hey %d", i)),
				})
			}
			_, err = svc.SendMessageBatch(batch)
			if err != nil {
				panic(err)
			}
			wg.Done()
			fmt.Printf("completed batch %d\n", batchNo)
		}(ses, t)
	}
	wg.Wait()
}
