package main

import (
	"github.com/Shopify/sarama"
	"encoding/json"
	"kuaishangtong/common/utils/httplib"
	"time"
	"fmt"
	"kuaishangtong/common/utils/log"
)

type FormDataBody struct {
	Text 	map[string]interface{}	`json:"text"`
	File 	map[string]interface{} 	`json:"file"`
}

type Body struct {
	Type 		string 	`json:"type"`
	JsonBody	string 	`json:"json_body"`
	StringBody	string 	`json:"string_body"`
	FormDataBody	FormDataBody	`json:"form_data_body"`
	FormUrlEncoded	map[string]interface{}	`json:"form_urlencoded"`
}

type Message struct {
	RequestId	string	`json:"request_id"`
	Method		string	`json:"method"`
	Url			string	`json:"url"`
	Header 		map[string]interface{}		`json:"header"`
	Body 		Body	`json:"body"`
}

type Task struct {
	attachQueue *TaskQueue
	msg 	*sarama.ConsumerMessage
}

const MAX_QUEUE_SIZE = 500
type TaskQueue struct {
	queueSize	int
	maxWaitTime	int
	taskqueue	chan *Task
}

func NewTask(attachQueue *TaskQueue) *Task {
	return &Task {
		attachQueue:	attachQueue,
	}
}

func (t *Task) Msg2Req(msg *sarama.ConsumerMessage) (*httplib.HTTPRequest, error) {
	var msg_json Message
	err := json.Unmarshal(msg.Value, &msg_json)
	if err != nil {
		return nil, err
	}

	var req *httplib.HTTPRequest
	switch msg_json.Body.Type {
	case "json":
		req = httplib.NewRequest(msg_json.Url, msg_json.Method).Body(msg_json.Body.JsonBody)
	case "string":
		req = httplib.NewRequest(msg_json.Url, msg_json.Method).Body(msg_json.Body.StringBody)
	case "form_data_body":
		req = httplib.NewRequest(msg_json.Url, msg_json.Method).Body(msg_json.Body.FormDataBody)
	case "form_urlencoded":
		req = httplib.NewRequest(msg_json.Url, msg_json.Method).Body(msg_json.Body.FormUrlEncoded)
	}

	for k, v := range msg_json.Header {
		req.Header(k, v.(string))
	}

	return req, nil
}

func NewTaskQueue(queueSize, maxWaitTime int) (*TaskQueue, error) {
	if queueSize <= 0 {
		return nil, fmt.Errorf("queueSize must be greater than 0")
	}

	if queueSize > MAX_QUEUE_SIZE {
		queueSize = MAX_QUEUE_SIZE
	}

	tQueue := &TaskQueue{
		queueSize:		queueSize,
		maxWaitTime:	maxWaitTime,
		taskqueue:		make(chan *Task, queueSize),
	}

	for i := 0;i < queueSize;i++ {
		tQueue.taskqueue <- NewTask(tQueue)
	}
	return tQueue, nil
}

func (this *TaskQueue) AddTask(msg *sarama.ConsumerMessage) {
	 task, err := this.getTask()
	 if err != nil {
		 return
	 }

	 if task != nil {
	 	log.Infof("key:%v, val:%v\n",msg.Key, msg.Value)
	 	req, err := task.Msg2Req(msg)
	 	if err != nil {
		 	log.Error(err)
		 	return
	 	}
	 	log.Info(req)

	 	_, err = req.Response()
	 	if err != nil {
	 		log.Error(err)
	 		return
		}
		//log.Infof(res.Body.Read())
	 }
	 return
}

func (this *TaskQueue) getTask() (*Task, error) {
	var task *Task

	for {
		select {
		case task = <-this.taskqueue:
			return task,nil

		default:
			var waitTime int = 0
			for {
				ticker := time.NewTicker(1 * time.Second)
				select {
				case <-ticker.C:
					waitTime++

					if waitTime == this.maxWaitTime {
						ticker.Stop()
						return nil, fmt.Errorf("out of time")
					}
				case task = <-this.taskqueue:
					ticker.Stop()
					return task, nil
				}
				ticker.Stop()
			}
		}
	}
	return task, nil
}

func (this *TaskQueue) putTask(task *Task) {
	this.taskqueue <- task
}