package main

import (
	"kuaishangtong/common/utils/log"
	//"kuaishangtong/common/zk"
	//"fmt"
	"kuaishangtong/common/kafka"
)

func main() {
	if !initFlags() {
		return
	}

	err := initConfig(_flags.Config)
	if err != nil {
		log.Fatal(err)
	}

	// log 设置
	logSet := defaultConfig.Log
	log.SetLogFuncCall(logSet.ShowLines)
	log.SetColor(logSet.Coloured)
	log.SetLevel(logSet.Level)
	if logSet.Enable {
		log.SetLogFile(
			logSet.File,
			logSet.Level,
			logSet.Daily,
			logSet.Coloured,
			logSet.ShowLines,
			logSet.MaxDays)
	}

	//zkclient, err := zk.NewGozkClient(
	//	defaultConfig.Zookeeper.ZookeeperHosts,
	//	defaultConfig.Zookeeper.ZookeeperServiceNode,nil, false)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}
	//
	//go func() {
	//	for {
	//		select {
	//		case c := <-zkclient.GetChildren():
	//			fmt.Println(c)
	//		}
	//	}
	//}()
	consumer, err := kafka.NewKafkaClusterConsumer(
		defaultConfig.Kafka.KafkaHosts,
		defaultConfig.Kafka.KafkaConsumerTopics,
		defaultConfig.Kafka.KafkaConsumerGroup)
	if err != nil {
		log.Fatal(err)
	}

	producer, err := kafka.NewKafkaSyncProducer(
		defaultConfig.Kafka.KafkaHosts,
		defaultConfig.Kafka.KafkaProducerTopic)
	if err != nil {
		log.Fatal(err)
	}

	tqueue, err := NewTaskQueue(defaultConfig.MaxQueueSize, defaultConfig.MaxWaitTime)
	if err != nil {
		log.Fatal(err)
	}

	proxy, err := NewProxy(consumer, producer, tqueue)
	if err != nil {
		log.Fatal(err)
	}

	proxy.Loop()
	select{}
}