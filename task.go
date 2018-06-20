package main

import (
	"github.com/Shopify/sarama"
	"encoding/json"
	"kuaishangtong/common/utils/httplib"
	"time"
	"fmt"
	"kuaishangtong/common/utils/log"
	"kuaishangtong/common/utils"
	//"strconv"
	"mime/multipart"
	"bytes"
	"os"
	"io"
	"net/http"
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
	//msg 	*sarama.ConsumerMessage
}

const MAX_QUEUE_SIZE = 500
type TaskQueue struct {
	proxy		*proxy
	queueSize	int
	maxWaitTime	int
	taskqueue	chan *Task
}

func NewTask(attachQueue *TaskQueue) *Task {
	return &Task {
		attachQueue:	attachQueue,
	}
}

func (t *Task) MsgJson2Req(msg Message) (*http.Request, error) {
	var req *http.Request
	switch msg.Body.Type {
	case "json":
		httpreq, err := httplib.NewRequest(msg.Url, msg.Method).JSONBody(msg.Body.JsonBody)
		if err != nil {
			return nil, err
		}
		req = httpreq.GetRequest()
	case "string":
		req = httplib.NewRequest(msg.Url, msg.Method).Body(msg.Body.StringBody).GetRequest()
	case "form_data_body":
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		for k, v := range msg.Body.FormDataBody.Text {
			err := w.WriteField(k, v.(string))
			if err != nil {
				return nil, err
			}
		}
		for k, v := range msg.Body.FormDataBody.File {
			file, err := os.Open(v.(string))
			if err != nil {
				return nil, fmt.Errorf("open file %v error: %v",v.(string), err)
			}
			defer file.Close()

			fw, err := w.CreateFormFile(k, v.(string))
			if err != nil {
				return nil, err
			}

			if _, err = io.Copy(fw, file); err != nil {
				return nil, err
			}
		}
		w.Close()
		req, err := http.NewRequest(msg.Method, msg.Url, &b)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", w.FormDataContentType())
	case "form_urlencoded":
		//req = httplib.NewRequest(msg.Url, msg.Method).Body(msg.Body.FormUrlEncoded)
		for k, v := range msg.Body.FormUrlEncoded {
			req.Form.Add(k, v.(string))
		}
	}

	//for k, v := range msg.Header {
	//	switch v.(type){
	//	case string:
	//		req.Header.Set(k, v.(string))
	//	case float64:
	//		req.Header.Set(k, strconv.FormatFloat(v.(float64),'E', -1 ,64))
	//	case int:
	//		req.Header.Set(k, strconv.Itoa(v.(int)))
	//	}
	//}

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
	 go this.callService(msg)
}

func (this *TaskQueue) callService(msg *sarama.ConsumerMessage) {
	var msg_json Message
	err := json.Unmarshal(msg.Value, &msg_json)
	if err != nil {
		log.Error(err)
		return
	}

	task, err := this.getTask()
	if err != nil {
		this.replyRes(msg_json.RequestId, "500")
		return
	}
	defer this.putTask(task)

	req, err := task.MsgJson2Req(msg_json)
	if err != nil {
		log.Error(err)
		return
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Errorf("server error, response status code: %d", res.StatusCode)
		return
	}

	respBuf := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(respBuf)

	data, err := utils.ReadAllToByteBuffer(res.Body, respBuf)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info(string(data))
	this.replyRes(msg_json.RequestId, string(data))
}

func (this *TaskQueue) replyRes(key, val string) {
	err := this.proxy.resProducer.SendStringMessage(key, val)
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("reply res key: %v, value: %v\n", key, val)
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