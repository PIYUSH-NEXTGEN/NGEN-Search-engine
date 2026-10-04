// Package session keeps a short conversation history per client so a
// follow-up like "tell me more about her" can be answered with context.
// Sessions are JSON in Redis under "session:<session_id>" with a sliding
// TTL — every Save restarts the inactivity clock, so a conversation dies
// 30 minutes after its last turn.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/llm"
)

// sessionTTL resets on every Save, which makes it an inactivity timeout
// rather than an absolute lifetime.
const sessionTTL = 30 * time.Minute

// maxTurns caps what one session can hold, so neither the Redis value nor
// the prompt handed to the model can grow without bound.
const maxTurns = 5

// Session is one client's conversation so far.
type Session struct {
	LastQuery  string        `json:"last_query"`
	LastAnswer llm.AskResult `json:"last_answer"`
	Turns      []llm.Turn    `json:"turns"` // oldest → newest, at most maxTurns
}

// AddTurn records a completed question/answer pair and drops everything
// older than the cap.
func (s *Session) AddTurn(query string, answer llm.AskResult) {
	s.LastQuery = query
	s.LastAnswer = answer
	s.Turns = append(s.Turns, llm.Turn{Query: query, Answer: answer.Answer})
	if len(s.Turns) > maxTurns {
		s.Turns = s.Turns[len(s.Turns)-maxTurns:]
	}
}

// Store reads and writes sessions in Redis. Like the search and answer
// caches it holds a raw *redis.Client rather than a narrower interface.
type Store struct {
	client *redis.Client
}

func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

func sessionKey(sessionID string) string {
	return "session:" + sessionID
}

// Get returns the stored session, or (nil, false) on a miss — the same
// shape the search and answer caches use, so callers can swallow errors
// the same way they do.
func (s *Store) Get(ctx context.Context, sessionID string) (*Session, bool, error) {
	val, err := s.client.Get(ctx, sessionKey(sessionID)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis get: %w", err)
	}
	var sess Session
	if err := json.Unmarshal([]byte(val), &sess); err != nil {
		return nil, false, fmt.Errorf("unmarshaling session: %w", err)
	}
	return &sess, true, nil
}

// Save stores the session and restarts the inactivity TTL.
func (s *Store) Save(ctx context.Context, sessionID string, sess *Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	if err := s.client.Set(ctx, sessionKey(sessionID), data, sessionTTL).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}
