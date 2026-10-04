package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/llm"
)

func TestAddTurnCapsHistory(t *testing.T) {
	var sess Session
	for i := 0; i < maxTurns*2; i++ {
		answer := llm.AskResult{Relevant: true, Answer: fmt.Sprintf("answer %d", i)}
		sess.AddTurn(fmt.Sprintf("question %d", i), answer)
	}

	if len(sess.Turns) != maxTurns {
		t.Errorf("len(Turns) = %d, want %d (history must be capped)", len(sess.Turns), maxTurns)
	}
	if got := sess.Turns[len(sess.Turns)-1].Query; got != "question 9" {
		t.Errorf("newest turn = %q, want the last question asked", got)
	}
	if got := sess.Turns[0].Query; got != "question 5" {
		t.Errorf("oldest kept turn = %q, want the newest %d retained", got, maxTurns)
	}
	if sess.LastQuery != "question 9" || sess.LastAnswer.Answer != "answer 9" {
		t.Errorf("LastQuery/LastAnswer = %q/%q", sess.LastQuery, sess.LastAnswer.Answer)
	}
}

func TestSessionJSONRoundTrip(t *testing.T) {
	want := Session{LastQuery: "who knows ML"}
	want.AddTurn("who knows ML", llm.AskResult{Relevant: true, Answer: "Piyush does."})

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Session
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip mismatch:\n got %#v\nwant %#v", got, want)
	}
}

// TestStoreRoundTrip exercises Save/Get against a real Redis (the compose
// one on localhost:6379 by default) and checks the sliding TTL. It skips
// rather than fails when Redis isn't reachable, so `go test ./...` still
// works on a machine with no stack running.
func TestStoreRoundTrip(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable at %s, skipping live round-trip: %v", addr, err)
	}
	t.Cleanup(func() { client.Close() })

	id := "go-test-round-trip"
	t.Cleanup(func() { client.Del(ctx, sessionKey(id)) })

	want := Session{LastQuery: "who knows machine learning"}
	want.AddTurn("who knows machine learning", llm.AskResult{
		Relevant: true,
		Answer:   "Piyush Baraskar works on machine learning and backend systems.",
	})

	store := NewStore(client)
	if err := store.Save(ctx, id, &want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, found, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("session not found right after Save")
	}
	if !reflect.DeepEqual(got, &want) {
		t.Errorf("round trip mismatch:\n got %#v\nwant %#v", got, want)
	}

	// Save must have armed the sliding TTL rather than stored it forever.
	ttl, err := client.TTL(ctx, sessionKey(id)).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 || ttl > sessionTTL {
		t.Errorf("TTL = %v, want greater than 0 and at most %v", ttl, sessionTTL)
	}

	// Unknown ids are a clean miss, not an error.
	if _, found, err := store.Get(ctx, id+"-does-not-exist"); err != nil || found {
		t.Errorf("miss path: found=%v err=%v, want false/nil", found, err)
	}
}

// TestTTLIsResetOnOverwrite checks the "30 minutes of inactivity" property:
// saving again must push the expiry back out to the full window.
func TestTTLIsResetOnOverwrite(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable at %s, skipping live TTL test: %v", addr, err)
	}
	t.Cleanup(func() { client.Close() })

	id := "go-test-ttl-reset"
	t.Cleanup(func() { client.Del(ctx, sessionKey(id)) })

	store := NewStore(client)
	first := Session{}
	first.AddTurn("q1", llm.AskResult{Relevant: true, Answer: "a1"})
	if err := store.Save(ctx, id, &first); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Rewind the expiry to prove the next Save moves it back out.
	if err := client.Expire(ctx, sessionKey(id), time.Minute).Err(); err != nil {
		t.Fatalf("Expire: %v", err)
	}
	second := Session{}
	second.AddTurn("q2", llm.AskResult{Relevant: true, Answer: "a2"})
	if err := store.Save(ctx, id, &second); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ttl, err := client.TTL(ctx, sessionKey(id)).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= time.Minute {
		t.Errorf("TTL after second Save = %v, want it reset beyond the rewound minute", ttl)
	}
}
