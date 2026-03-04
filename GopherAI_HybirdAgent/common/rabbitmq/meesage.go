package rabbitmq

import (
	"GopherAI/dao/message"
	"GopherAI/model"
	"encoding/json"

	"github.com/streadway/amqp"
)

type MessageMQParam struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	UserName  string `json:"user_name"`
	IsUser    bool   `json:"is_user"`
}

func GenerateMessageMQParam(sessionID string, content string, userName string, isUser bool) []byte {
	param := MessageMQParam{
		SessionID: sessionID,
		Content:   content,
		UserName:  userName,
		IsUser:    isUser,
	}
	data, _ := json.Marshal(param)
	return data
}

func MQMessage(msg *amqp.Delivery) error {
	var param MessageMQParam
	if err := json.Unmarshal(msg.Body, &param); err != nil {
		return err
	}

	newMsg := &model.Message{
		SessionID: param.SessionID,
		Content:   param.Content,
		UserName:  param.UserName,
		IsUser:    param.IsUser,
	}

	_, err := message.CreateMessage(newMsg)
	return err
}
