package kafka

import (
	"blaster/core"
	"context"
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

const DataItemMessage string = "message"

type KafkaTransporter struct {
	messages chan []*core.Message
	session  sarama.ConsumerGroupSession
}

func (t *KafkaTransporter) Messages() <-chan []*core.Message {
	return t.messages
}

func (t *KafkaTransporter) Delete(m *core.Message) error {
	msg := m.Data[DataItemMessage].(*sarama.ConsumerMessage)
	// TODO: Consider reading the metadata string from the config
	t.session.MarkMessage(msg, "")
	return nil
}

func (t *KafkaTransporter) Poison(m *core.Message) error {
	return nil
}

func (t *KafkaTransporter) Close() {
	close(t.messages)
}

type PartionHandler struct {
	Transporter    *KafkaTransporter
	MessagePump    *core.MessagePump
	HandlerManager *core.HandlerManager
	Started        bool
}

func (h *PartionHandler) Start(ctx context.Context) {
	h.HandlerManager.Start(ctx)
	h.MessagePump.Start(ctx)
}

type SaramaConsumerGroupHandler struct {
	PartionHandlers map[int32]*PartionHandler
	Mutex           sync.Mutex
	Binding         *KafkaBinder
	Context         context.Context
	done            chan struct{}
	logFields       log.Fields
}

func (h *SaramaConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	config := h.Binding.Config
	for _, partions := range session.Claims() {
		for _, p := range partions {
			port := core.GetFreePort()
			handlerURL := fmt.Sprintf("http://localhost:%d/", port)
			dispatcher := core.NewHttpDispatcher(handlerURL)
			qsvc := &KafkaTransporter{
				messages: make(chan []*core.Message),
				session:  session,
			}
			pump := core.NewMessagePump(qsvc, dispatcher, config.RetryCount, config.RetryDelay, config.MaxHandlers)
			hm := core.NewHandlerManager(config.HandlerCommand, config.HandlerArgs, handlerURL, config.StartupDelaySeconds)
			ph := &PartionHandler{
				Transporter:    qsvc,
				MessagePump:    pump,
				HandlerManager: hm,
			}

			ph.Start(session.Context())
			h.PartionHandlers[p] = ph
		}
	}
	return nil
}

func (h *SaramaConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	for _, v := range h.PartionHandlers {
		if !v.Started {
			v.Transporter.Close()
		}
		err := v.HandlerManager.Awaiter.Err()
		log.WithFields(h.logFields).WithFields(log.Fields{"generationId": session.GenerationID(), "err": err}).Info("handler manager exited")
		err = v.MessagePump.Awaiter.Err()
		log.WithFields(h.logFields).WithFields(log.Fields{"generationId": session.GenerationID(), "err": err}).Info("message pump exited")
	}
	close(h.done)
	log.WithFields(h.logFields).WithFields(log.Fields{"generationId": session.GenerationID()}).Info("consumer group handler is cleaned up")
	return nil
}

func (h *SaramaConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.Mutex.Lock()
	p, ok := h.PartionHandlers[claim.Partition()]
	if !ok {
		return errors.WithStack(errors.New("unable to consume a claim with an unclaimed partion"))
	}
	p.Started = true
	h.Mutex.Unlock()

	// Loop until the messages channel is open or
	// one of partion handler components exit.
	// When a partion handler component exit, we return
	// from the method and leave Sarama to cancel the
	// session. Since session's context is associated
	// with both MessagePump and HandlerManager, this
	// should gracefully shutdown all components.
ReceiveLoop:
	for {
		var m *core.Message
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				break ReceiveLoop
			}
			m = &core.Message{
				MessageID:  string(msg.Key),
				Body:       string(msg.Value),
				Properties: make(map[string]interface{}),
				Data:       make(map[string]interface{}),
			}
			m.Properties["timestamp"] = msg.Timestamp
			m.Properties["partitionId"] = msg.Partition
			m.Properties["offset"] = msg.Offset
			m.Data["message"] = msg
		case <-p.HandlerManager.Awaiter.Done():
			break ReceiveLoop
		case <-p.MessagePump.Awaiter.Done():
			break ReceiveLoop
		}

		select {
		case p.Transporter.messages <- []*core.Message{m}:
		case <-p.HandlerManager.Awaiter.Done():
			break ReceiveLoop
		case <-p.MessagePump.Awaiter.Done():
			break ReceiveLoop
		}
	}
	p.Transporter.Close()
	return nil
}

func (h *SaramaConsumerGroupHandler) Done() <-chan struct{} {
	return h.done
}

type KafkaConfig struct {
	Topic           string
	Group           string
	BufferSize      int
	BrokerAddresses []string
	StartFromOldest bool
}

type KafkaBinder struct {
	Group         sarama.ConsumerGroup
	KafkaConfig   *KafkaConfig
	Config        *core.Config
	awaiter       *core.Awaiter
	awaitNotifier *core.AwaitNotifier
	logFields     log.Fields
}

func (b *KafkaBinder) Start(ctx context.Context) {
	go func() {
		defer b.Group.Close()

		// Iterate over consumer sessions until we have an error
		// or the context is cancelled.
		for {
			topics := []string{b.KafkaConfig.Topic}
			handler := &SaramaConsumerGroupHandler{
				Binding:         b,
				Context:         ctx,
				PartionHandlers: make(map[int32]*PartionHandler),
				done:            make(chan struct{}),
				logFields:       log.Fields{"module": "consumer_group_handler"},
			}

			err := b.Group.Consume(ctx, topics, handler)
			if err != nil {
				b.awaitNotifier.Notify(err)
				return
			}

			// Wait for the cleanup of current handler before starting a new one.
			// TODO: Consider if this can be done in the background.
			<-handler.Done()

			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			select {
			case <-ctx.Done():
				b.awaitNotifier.Notify(nil)
				return
			default:
			}
		}
	}()
}

func (b *KafkaBinder) Awaiter() *core.Awaiter {
	return b.awaiter
}

func NewKafkaBinder(kafkaConfig *KafkaConfig, coreConfig *core.Config) (*KafkaBinder, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_4_0_0 // specify appropriate version
	config.Consumer.Return.Errors = true
	if kafkaConfig.StartFromOldest {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	if kafkaConfig.BufferSize != 0 {
		config.ChannelBufferSize = kafkaConfig.BufferSize
	}

	g, err := sarama.NewConsumerGroup(kafkaConfig.BrokerAddresses, kafkaConfig.Group, config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	logFields := log.Fields{"module": "kafka_binding"}
	// Track errors
	go func() {
		for err := range g.Errors() {
			log.WithFields(log.Fields{"err": err}).Info("error in consumer group")
		}
	}()

	awaiter, awaitNotifier := core.NewAwaiter()
	return &KafkaBinder{
		Group:         g,
		KafkaConfig:   kafkaConfig,
		Config:        coreConfig,
		awaiter:       awaiter,
		awaitNotifier: awaitNotifier,
		logFields:     logFields,
	}, nil
}
