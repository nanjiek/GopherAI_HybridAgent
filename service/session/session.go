package session

import (
	"github.com/nanjiek/GopherAI_HybridAgent/common/aihelper"
	"github.com/nanjiek/GopherAI_HybridAgent/common/code"
	messagedao "github.com/nanjiek/GopherAI_HybridAgent/dao/message"
	sessiondao "github.com/nanjiek/GopherAI_HybridAgent/dao/session"
	"github.com/nanjiek/GopherAI_HybridAgent/model"
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ctx = context.Background()

func GetUserSessionsByUserName(userName string) ([]model.SessionInfo, error) {
	sessions, err := sessiondao.GetSessionsByUserName(userName)
	if err != nil {
		return nil, err
	}

	sessionInfos := make([]model.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			title = s.ID
		}
		sessionInfos = append(sessionInfos, model.SessionInfo{
			SessionID: s.ID,
			Title:     title,
		})
	}
	return sessionInfos, nil
}

func CreateSessionAndSendMessage(userName string, userQuestion string, modelType string) (string, string, code.Code) {
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    userQuestion,
	}
	createdSession, err := sessiondao.CreateSession(newSession)
	if err != nil {
		log.Println("CreateSessionAndSendMessage CreateSession error:", err)
		return "", "", code.CodeServerBusy
	}

	helper, status := getOrCreateHelperWithHistory(userName, createdSession.ID, modelType)
	if status != code.CodeSuccess {
		return "", "", status
	}

	aiResponse, err := helper.GenerateResponse(userName, ctx, userQuestion)
	if err != nil {
		log.Println("CreateSessionAndSendMessage GenerateResponse error:", err)
		return "", "", code.AIModelFail
	}

	return createdSession.ID, aiResponse.Content, code.CodeSuccess
}

func CreateStreamSessionOnly(userName string, userQuestion string) (string, code.Code) {
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    userQuestion,
	}
	createdSession, err := sessiondao.CreateSession(newSession)
	if err != nil {
		log.Println("CreateStreamSessionOnly CreateSession error:", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

func StreamMessageToExistingSession(parentCtx context.Context, userName string, sessionID string, userQuestion string, modelType string, writer http.ResponseWriter) code.Code {
	streamCtx, cancel := context.WithTimeout(parentCtx, 3*time.Minute)
	defer cancel()

	flusher, ok := writer.(http.Flusher)
	if !ok {
		log.Println("StreamMessageToExistingSession: streaming unsupported")
		return code.CodeServerBusy
	}

	helper, status := getOrCreateHelperWithHistory(userName, sessionID, modelType)
	if status != code.CodeSuccess {
		return status
	}

	var writeMu sync.Mutex
	writeEvent := func(eventName, data string) error {
		writeMu.Lock()
		defer writeMu.Unlock()

		payload := "data: " + data + "\n\n"
		if eventName != "" {
			payload = "event: " + eventName + "\n" + payload
		}
		_, writeErr := writer.Write([]byte(payload))
		if writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	}

	heartbeatDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-streamCtx.Done():
				close(heartbeatDone)
				return
			case <-ticker.C:
				if err := writeEvent("heartbeat", "ping"); err != nil {
					cancel()
					close(heartbeatDone)
					return
				}
			}
		}
	}()

	cb := func(msg string) {
		select {
		case <-streamCtx.Done():
			return
		default:
		}
		if err := writeEvent("", msg); err != nil {
			log.Println("[SSE] Write error:", err)
			cancel()
		}
	}

	_, err := helper.StreamResponse(userName, streamCtx, cb, userQuestion)
	if err != nil {
		log.Println("StreamMessageToExistingSession StreamResponse error:", err)
		return code.AIModelFail
	}

	if err = writeEvent("done", "[DONE]"); err != nil {
		log.Println("StreamMessageToExistingSession write DONE error:", err)
		return code.AIModelFail
	}
	cancel()
	<-heartbeatDone

	return code.CodeSuccess
}

func CreateStreamSessionAndSendMessage(parentCtx context.Context, userName string, userQuestion string, modelType string, writer http.ResponseWriter) (string, code.Code) {
	sessionID, status := CreateStreamSessionOnly(userName, userQuestion)
	if status != code.CodeSuccess {
		return "", status
	}

	status = StreamMessageToExistingSession(parentCtx, userName, sessionID, userQuestion, modelType, writer)
	if status != code.CodeSuccess {
		return sessionID, status
	}

	return sessionID, code.CodeSuccess
}

func ChatSend(userName string, sessionID string, userQuestion string, modelType string) (string, code.Code) {
	if !isSessionOwner(userName, sessionID) {
		return "", code.CodeForbidden
	}

	helper, status := getOrCreateHelperWithHistory(userName, sessionID, modelType)
	if status != code.CodeSuccess {
		return "", status
	}

	aiResponse, err := helper.GenerateResponse(userName, ctx, userQuestion)
	if err != nil {
		log.Println("ChatSend GenerateResponse error:", err)
		return "", code.AIModelFail
	}

	return aiResponse.Content, code.CodeSuccess
}

func GetChatHistory(userName string, sessionID string) ([]model.History, code.Code) {
	if !isSessionOwner(userName, sessionID) {
		return nil, code.CodeForbidden
	}

	messages, err := messagedao.GetMessagesBySessionID(sessionID)
	if err != nil {
		log.Println("GetChatHistory GetMessagesBySessionID error:", err)
		return nil, code.CodeServerBusy
	}

	history := make([]model.History, 0, len(messages))
	for _, msg := range messages {
		history = append(history, model.History{
			IsUser:  msg.IsUser,
			Content: msg.Content,
		})
	}

	return history, code.CodeSuccess
}

func ChatStreamSend(parentCtx context.Context, userName string, sessionID string, userQuestion string, modelType string, writer http.ResponseWriter) code.Code {
	if !isSessionOwner(userName, sessionID) {
		return code.CodeForbidden
	}
	return StreamMessageToExistingSession(parentCtx, userName, sessionID, userQuestion, modelType, writer)
}

func getOrCreateHelperWithHistory(userName, sessionID, modelType string) (*aihelper.AIHelper, code.Code) {
	manager := aihelper.GetGlobalManager()
	if helper, exists := manager.GetAIHelper(userName, sessionID); exists {
		return helper, code.CodeSuccess
	}

	cfg := map[string]interface{}{
		"username": userName,
	}
	helper, err := manager.GetOrCreateAIHelper(userName, sessionID, modelType, cfg)
	if err != nil {
		log.Println("getOrCreateHelperWithHistory GetOrCreateAIHelper error:", err)
		return nil, code.AIModelFail
	}

	messages, err := messagedao.GetMessagesBySessionID(sessionID)
	if err != nil {
		log.Println("getOrCreateHelperWithHistory GetMessagesBySessionID error:", err)
		return nil, code.CodeServerBusy
	}
	for i := range messages {
		m := &messages[i]
		helper.AddMessage(m.Content, m.UserName, m.IsUser, false)
	}

	return helper, code.CodeSuccess
}

func isSessionOwner(userName, sessionID string) bool {
	s, err := sessiondao.GetSessionByID(sessionID)
	if err != nil {
		return false
	}
	return s.UserName == userName
}
