package tcp_transport

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IskenT/spin-wisdom/internal/service/quotes/model"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type mockConn struct {
	net.Conn
	readData  string
	writeData []byte
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	copy(b, m.readData)
	return len(m.readData), nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	m.writeData = append(m.writeData, b...)
	return len(b), nil
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func TestHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPow := NewMockPowService(ctrl)
	mockQuote := NewMockQuoteService(ctrl)
	difficulty := 4

	tests := []struct {
		name           string
		challenge      string
		response       string
		quote          model.Quote
		validateResult bool
		expectError    bool
	}{
		{
			name:      "successful_case",
			challenge: "1234",
			response:  "5678\n",
			quote: model.Quote{
				Quote:  "Test quote",
				Author: "Test author",
			},
			validateResult: true,
			expectError:    false,
		},
		{
			name:           "invalid_pow_solution",
			challenge:      "1234",
			response:       "wrong\n",
			validateResult: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &mockConn{
				readData: tt.response,
			}

			mockPow.EXPECT().
				GenerateChallenge(gomock.Any()).
				Return(tt.challenge, nil)

			mockPow.EXPECT().
				ValidateChallenge(
					gomock.Any(),
					difficulty,
					tt.challenge,
					strings.TrimSpace(tt.response),
				).
				Return(tt.validateResult)

			if tt.validateResult {
				mockQuote.EXPECT().
					GetRandomQuote(gomock.Any()).
					Return(tt.quote)
			}

			handler := NewHandler(mockPow, mockQuote, difficulty)

			handler.Handle(context.Background(), mockConn)

			if !tt.expectError {
				expectedInitialWrite := strconv.Itoa(difficulty) + "\n" + tt.challenge + "\n"
				assert.True(t, strings.HasPrefix(string(mockConn.writeData), expectedInitialWrite))

				if tt.validateResult {
					expectedQuote, _ := json.Marshal(tt.quote)
					assert.True(t, strings.Contains(string(mockConn.writeData), string(expectedQuote)))
				}
			}
		})
	}
}

func TestHandler_Handle_GenerateChallengeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPow := NewMockPowService(ctrl)
	mockQuote := NewMockQuoteService(ctrl)
	mockConn := &mockConn{}
	difficulty := 4

	mockPow.EXPECT().
		GenerateChallenge(gomock.Any()).
		Return("", assert.AnError)

	handler := NewHandler(mockPow, mockQuote, difficulty)
	handler.Handle(context.Background(), mockConn)

	assert.Empty(t, mockConn.writeData)
}
