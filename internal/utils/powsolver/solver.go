package powsolver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

func FindSolution(challenge string, difficulty int) int {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		solution int
	)

	targetPrefix := strings.Repeat("0", difficulty/4)
	numWorkers := 8
	workChan := make(chan int, numWorkers*2)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for nonce := range workChan {
				hashStr := hashWithNonce(challenge, nonce)
				if strings.HasPrefix(hashStr, targetPrefix) {
					mu.Lock()
					if solution == 0 {
						solution = nonce
					}
					mu.Unlock()
					return
				}
			}
		}()
	}

	for nonce := 0; solution == 0; nonce++ {
		workChan <- nonce
	}

	close(workChan)
	wg.Wait()

	return solution
}

func hashWithNonce(challenge string, nonce int) string {
	data := []byte(fmt.Sprintf("%s%d", challenge, nonce))
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
