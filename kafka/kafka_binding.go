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

type KafkaService struct {
	messages chan *core.Message
}

func (b *KafkaService) Read() ([]*core.Message, error) {
	m, ok := <-b.messages
	if !ok {
		// Channel is closed when PartionHandler is being
		// cleaned up. Return an empty result so that
		// the MessagePump can unblock and eventually
		// shutdown.
		return []*core.Message{}, nil
	}
	return []*core.Message{m}, nil
}

func (b *KafkaService) Delete(m *core.Message) error {
	// TODO: Provide granular controls to manipulate
	// how offset is committed.
	return nil
}

func (b *KafkaService) Poison(m *core.Message) error {
	return nil
}

func (b *KafkaService) Messages() chan<- *core.Message {
	return b.messages
}

func (b *KafkaService) Close() {
	close(b.messages)
}

func NewKafkaService() *KafkaService {
	return &KafkaService{
		messages: make(chan *core.Message),
	}
}

type PartionHandler struct {
	KafkaService   *KafkaService
	MessagePump    *core.MessagePump
	HandlerManager *core.HandlerManager
}

func (h *PartionHandler) Start(ctx context.Context) {
	h.HandlerManager.Start(ctx)
	h.MessagePump.Start(ctx)
}

type SaramaConsumerGroupHandler struct {
	PartionHandlers map[int32]*PartionHandler
	Mutex           sync.Mutex
	Binding         *KafkaBinding
	Context         context.Context
	done            chan struct{}
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
			qsvc := NewKafkaService()
			pump := core.NewMessagePump(qsvc, dispatcher, config.RetryCount, config.RetryDelay, config.MaxHandlers)
			hm := core.NewHandlerManager(config.HandlerCommand, config.HandlerArgs, handlerURL, config.StartupDelaySeconds)
			ph := &PartionHandler{
				KafkaService:   qsvc,
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
		<-v.HandlerManager.Done()
		<-v.MessagePump.Done()
	}
	close(h.done)
	log.WithFields(log.Fields{"module": "consumer_group_handler", "generationId": session.GenerationID()}).Info("consumer group handler is cleaned up")
	return nil
}

func (h *SaramaConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.Mutex.Lock()
	p, ok := h.PartionHandlers[claim.Partition()]
	if !ok {
		return errors.WithStack(errors.New("unable to consume a claim with an unclaimed partion"))
	}
	h.Mutex.Unlock()

	// Loop until the messages channel is open or
	// one of partion handler components exit.
	// When a partion handler component exit, we return
	// from the method and leave Sarama to cancel the
	// session. Since session's context is associated
	// with both MessagePump and HandlerManager, this
	// should gracefully shutdown all components.
loop:
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				break loop
			}
			m := &core.Message{
				MessageID:  string(msg.Key),
				Body:       string(msg.Value),
				Properties: make(map[string]interface{}),
			}
			m.Properties["timestamp"] = msg.Timestamp
			m.Properties["partitionId"] = msg.Partition
			m.Properties["offset"] = msg.Offset
			p.KafkaService.Messages() <- m
		case <-p.HandlerManager.Done():
			break loop
		case <-p.MessagePump.Done():
			break loop
		}
	}
	p.KafkaService.Close()
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

type KafkaBinding struct {
	Group       sarama.ConsumerGroup
	KafkaConfig *KafkaConfig
	Config      *core.Config
	done        chan error
}

func (b *KafkaBinding) Start(ctx context.Context) {
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
			}

			err := b.Group.Consume(ctx, topics, handler)
			if err != nil {
				b.close(err)
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
				b.close(nil)
				return
			default:
			}
		}
	}()
}

func (b *KafkaBinding) Done() <-chan error {
	return b.done
}

func (b *KafkaBinding) close(err error) {
	b.done <- err
	close(b.done)
}

func NewKafkaBinding(kafkaConfig *KafkaConfig, coreConfig *core.Config) (*KafkaBinding, error) {
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

	// Track errors
	go func() {
		for err := range g.Errors() {
			log.WithFields(log.Fields{"module": "kafka_binding", "err": err}).Info("error in consumer group")
		}
	}()

	return &KafkaBinding{
		Group:       g,
		KafkaConfig: kafkaConfig,
		Config:      coreConfig,
		done:        make(chan error),
	}, nil
}
