package convstore

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// Store manages conversationId → parentId mappings for MiMo conversation reuse.
// MiMo maintains server-side context when the same conversationId + parentId chain is used.
type Store struct {
	mu    sync.RWMutex
	convs map[string]*convState // key: hash of first user message, value: conversationId + parentId
}

type convState struct {
	ConvID   string // random UUID sent to MiMo (unique, no collision with existing MiMo convs)
	ParentID string // last AI response message ID from MiMo SSE
}

// processSalt 进程级随机盐，避免不同实例间首条消息相同导致的会话串扰
var processSalt = func() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 极端情况下退化为固定值（不影响功能，仅降低隔离性）
		return "fallback-salt-v1"
	}
	return hex.EncodeToString(b)
}()

func New() *Store {
	return &Store{
		convs: make(map[string]*convState),
	}
}

// DeriveKey generates a stable lookup key from the first user message + model.
// clientHint 可由客户端通过 X-Conv-Key 头显式指定会话标识，
// 显式指定时优先使用，避免不同对话首条消息相同导致的上下文串扰。
// This is used ONLY for local lookup, NOT sent to MiMo.
func DeriveKey(firstUserMsg, model, clientHint string) string {
	base := firstUserMsg + "|" + model
	if clientHint != "" {
		base = clientHint + "|" + model
	}
	// 加入进程级 salt，隔离不同实例
	h := sha256.Sum256([]byte(base + "|" + processSalt))
	return fmt.Sprintf("%x", h[:16]) // 32-char hex
}

// GetOrCreate returns the conversationId and parentId for a conversation.
// If no conversation exists for this key, creates a new one with a random UUID.
func (s *Store) GetOrCreate(key string) (convID, parentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cs, ok := s.convs[key]; ok {
		return cs.ConvID, cs.ParentID
	}
	newConvID := randomHex32()
	s.convs[key] = &convState{ConvID: newConvID, ParentID: "0"}
	return newConvID, "0"
}

// SetParentID updates the last AI response message ID for a conversation.
func (s *Store) SetParentID(key, parentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cs, ok := s.convs[key]; ok {
		cs.ParentID = parentID
	}
}

// randomHex32 generates a random 32-char hex string using crypto/rand.
func randomHex32() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
