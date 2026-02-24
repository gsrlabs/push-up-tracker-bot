package hendler

import "sync"

type InputManager struct {
	storage sync.Map
}

func NewInputManager() *InputManager {
	return &InputManager{}
}

type PendingInput struct {
	InputType   inputType
	MessageID   int
	CancelMsgID int
}

func (m *InputManager) Set(chatID int64, input PendingInput) {
	m.storage.Store(chatID, input)
}

func (m *InputManager) Get(chatID int64) (PendingInput, bool) {
	val, ok := m.storage.Load(chatID)
	if !ok {
		return PendingInput{}, false
	}
	return val.(PendingInput), true
}

func (m *InputManager) Delete(chatID int64) {
	m.storage.Delete(chatID)
}