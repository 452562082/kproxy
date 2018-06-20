package main

import (
	"kuaishangtong/common/kafka"
	"kuaishangtong/common/utils/log"
)

type proxy struct {
	reqConsumer	*kafka.KafkaClusterConsumer
	resProducer	*kafka.KafkaSyncProducer
	taskQueue	*TaskQueue
}

func NewProxy(gkConsumer *kafka.KafkaClusterConsumer,
	gkProducer *kafka.KafkaSyncProducer,
	taskQueue *TaskQueue) (*proxy, error) {

	p := &proxy{
		reqConsumer:	gkConsumer,
		resProducer:	gkProducer,
		taskQueue: 		taskQueue,
	}
	return p, nil
}

func (p *proxy) Loop() {
	go p.consumerRecvLoop()
	select{}
}

func (p *proxy) consumerRecvLoop() {
	for {
		select {
		case msg, ok := <-p.reqConsumer.Messages():
			if ok && msg != nil {
				p.taskQueue.AddTask(msg)
			}
		case err := <-p.reqConsumer.Errors():
			log.Error(err)

		case notice, ok := <-p.reqConsumer.Notifications():
			if ok {
				if patitions := notice.Current[defaultConfig.Kafka.KafkaConsumerTopics[0]]; len(patitions) > 0 {
					log.Noticef("proxy consumer topic %s, patitions: %+v",
						defaultConfig.Kafka.KafkaConsumerTopics[0],
						notice.Current[defaultConfig.Kafka.KafkaConsumerTopics[0]])
				}
			}
		}
	}
	p.reqConsumer.Close()
}