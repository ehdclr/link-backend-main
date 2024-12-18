package nats

import (
	"fmt"
	"link/pkg/common"

	"github.com/nats-io/nats.go"
)

//! NATS publisher 패키지

type NatsPublisher struct {
	conn *nats.Conn
}

func NewPublisher(conn *nats.Conn) *NatsPublisher {
	return &NatsPublisher{conn: conn}
}

func (p *NatsPublisher) PublishEvent(subject string, data []byte) error {
	if err := p.conn.Publish(subject, data); err != nil {
		fmt.Printf("NATS 이벤트 발행 오류[TOPIC: %s]: %v ", subject, err)
		return common.NewError(500, "NATS 이벤트 발행 오류", err)
	}
	return nil
}
