package kafka

import (
	"context"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type kafkaMessage struct {
	msg    *kafka.Message
	reader Reader
}

func newKafkaMessage(msg *kafka.Message, reader Reader) *kafkaMessage {
	return &kafkaMessage{
		msg:    msg,
		reader: reader,
	}
}

func (kmsg *kafkaMessage) Commit() {
	if kmsg.reader != nil {
		err := kmsg.reader.CommitMessages(context.Background(), *kmsg.msg)
		if err != nil {
			slog.Error("unable to commit message on kafka")
		}
	}
}
