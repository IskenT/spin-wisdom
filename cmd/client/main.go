package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/IskenT/spin-wisdom/internal/utils/powsolver"
)

type JSONQuote struct {
	Quote  string `json:"quote"`
	Author string `json:"author"`
}

// connectToServer...
func connectToServer(address string) (net.Conn, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("error connecting to server: %w", err)
	}
	return conn, nil
}

// readLine...
func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("error reading line: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// sendSolution...
func sendSolution(conn net.Conn, solution int) error {
	_, err := fmt.Fprintf(conn, "%d\n", solution)
	if err != nil {
		return fmt.Errorf("error sending solution: %w", err)
	}
	return nil
}

// receiveQuote...
func receiveQuote(reader *bufio.Reader) (JSONQuote, error) {
	res, err := reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return JSONQuote{}, fmt.Errorf("error reading response: %w", err)
	}

	if len(res) == 0 {
		return JSONQuote{}, fmt.Errorf("received empty response from server")
	}

	var quote JSONQuote
	if err := json.Unmarshal(res, &quote); err != nil {
		return JSONQuote{}, fmt.Errorf("error unmarshalling quote: %w", err)
	}

	return quote, nil
}

func main() {
	conn, err := connectToServer("localhost:8083")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Println("Error closing connection:", err)
		}
	}()

	reader := bufio.NewReader(conn)

	// Read
	difficultyStr, err := readLine(reader)
	if err != nil {
		log.Fatal(err)
	}

	challenge, err := readLine(reader)
	if err != nil {
		log.Fatal(err)
	}

	// Converter
	difficulty, err := strconv.Atoi(difficultyStr)
	if err != nil {
		log.Fatal("Error parsing difficulty:", err)
	}

	solution := powsolver.FindSolution(challenge, difficulty)

	if err := sendSolution(conn, solution); err != nil {
		log.Fatal(err)
	}

	quote, err := receiveQuote(reader)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Quote: %s\n", quote.Quote)
	log.Printf("Author: %s\n", quote.Author)
}
