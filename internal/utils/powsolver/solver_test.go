package powsolver

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFindSolution(t *testing.T) {
	tests := []struct {
		name             string
		challenge        string
		difficulty       int
		validateSolution func(t *testing.T, challenge string, difficulty int, solution int)
		timeout          time.Duration
	}{
		{
			name:       "difficulty_4",
			challenge:  "test123",
			difficulty: 4,
			validateSolution: func(t *testing.T, challenge string, difficulty int, solution int) {
				hash := hashWithNonce(challenge, solution)
				assert.True(t, strings.HasPrefix(hash, "0"))
			},
			timeout: 2 * time.Second,
		},
		{
			name:       "difficulty_8",
			challenge:  "test456",
			difficulty: 8,
			validateSolution: func(t *testing.T, challenge string, difficulty int, solution int) {
				hash := hashWithNonce(challenge, solution)
				assert.True(t, strings.HasPrefix(hash, "00"))
			},
			timeout: 5 * time.Second,
		},
		{
			name:       "zero_difficulty",
			challenge:  "test789",
			difficulty: 0,
			validateSolution: func(t *testing.T, challenge string, difficulty int, solution int) {
				assert.NotEqual(t, 0, solution)
			},
			timeout: 1 * time.Second,
		},
		{
			name:       "empty_challenge",
			challenge:  "",
			difficulty: 4,
			validateSolution: func(t *testing.T, challenge string, difficulty int, solution int) {
				hash := hashWithNonce(challenge, solution)
				assert.True(t, strings.HasPrefix(hash, "0"))
			},
			timeout: 2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := make(chan int, 1)

			go func() {
				solution := FindSolution(tt.challenge, tt.difficulty)
				done <- solution
			}()

			select {
			case solution := <-done:
				tt.validateSolution(t, tt.challenge, tt.difficulty, solution)
			case <-time.After(tt.timeout):
				t.Fatalf("timeout after %v", tt.timeout)
			}
		})
	}
}

func TestHashWithNonce(t *testing.T) {
	tests := []struct {
		name      string
		challenge string
		nonce     int
		wantLen   int
	}{
		{
			name:      "normal_case",
			challenge: "test123",
			nonce:     42,
			wantLen:   64,
		},
		{
			name:      "empty_challenge",
			challenge: "",
			nonce:     0,
			wantLen:   64,
		},
		{
			name:      "large_nonce",
			challenge: "test123",
			nonce:     999999,
			wantLen:   64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashWithNonce(tt.challenge, tt.nonce)

			assert.Equal(t, tt.wantLen, len(result))

			_, err := hex.DecodeString(result)
			assert.NoError(t, err, "hash should be valid hex string")

			result2 := hashWithNonce(tt.challenge, tt.nonce)
			assert.Equal(t, result, result2, "hash function should be deterministic")
		})
	}
}

func TestConcurrentOperation(t *testing.T) {
	const numConcurrent = 3
	var wg sync.WaitGroup
	results := make([]int, numConcurrent)

	challenge := "testconcurrent"
	difficulty := 4

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = FindSolution(challenge, difficulty)
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		for i, solution := range results {
			hash := hashWithNonce(challenge, solution)
			assert.True(t, strings.HasPrefix(hash, "0"),
				"solution %d with nonce %d did not produce valid hash: %s", i, solution, hash)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for concurrent solutions")
	}
}

func BenchmarkFindSolution(b *testing.B) {
	difficulties := []int{4, 8, 12}
	challenge := "benchtest"

	for _, diff := range difficulties {
		b.Run(fmt.Sprintf("difficulty_%d", diff), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				FindSolution(challenge, diff)
			}
		})
	}
}
