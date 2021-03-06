package bully

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockBully is a testing function returning a mock `*bully.Bully`.
func mockBully(ID, coordinator, addr string) *Bully {
	return &Bully{
		ID:           ID,
		addr:         addr,
		coordinator:  coordinator,
		peers:        NewPeerMap(),
		mu:           &sync.RWMutex{},
		electionChan: make(chan Message, 1),
		receiveChan:  make(chan Message),
	}
}

// mockSocket is a `struct` only used for testing purposes.
type mockSocket struct {
	*net.TCPListener

	msgChan chan Message
}

// newMockSocket is a testing function returning a new `*bully.mockSocket` or an
// `error` if something bad occurs.
func newMockSocket(addr string) (*mockSocket, error) {
	laddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return nil, fmt.Errorf("newMockSocket: %v", err)
	}
	tcpListener, err := net.ListenTCP("tcp4", laddr)
	if err != nil {
		return nil, fmt.Errorf("newMockSocket: %v", err)
	}
	return &mockSocket{TCPListener: tcpListener, msgChan: make(chan Message, 1)}, nil
}

// accept is a testing function mimicking the behaviour of a `bully.Bully`.
func (ms *mockSocket) accept() {
	var msg Message

	conn, err := ms.AcceptTCP()
	if err != nil {
		log.Printf("mockSocket: %v", err)
		return
	}
	dec := gob.NewDecoder(conn)
	for {
		if err := dec.Decode(&msg); err == nil {
			ms.msgChan <- msg
		}
	}
}

// -----------------------------------------------------------------------------

func TestBully_NewBully(t *testing.T) {
	testCases := []struct {
		name                    string
		mockID                  string
		mockAddr                string
		mockProto               string
		mockPeers               map[string]string
		expectedAssertBullyFunc func(assert.TestingT, interface{}, ...interface{}) bool
		expectedAssertErrorFunc func(assert.TestingT, interface{}, ...interface{}) bool
	}{
		{
			"regular", "1",
			"127.0.0.1:8000",
			"tcp4",
			nil,
			assert.NotNil,
			assert.Nil,
		},
		{
			"badProto",
			"1",
			"127.0.0.1:8001",
			"tcp22",
			nil,
			assert.Nil,
			assert.NotNil,
		},
		{
			"badAddr",
			"1",
			"errorAddr:8002",
			"tcp4",
			nil,
			assert.Nil,
			assert.NotNil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := NewBully(tc.mockID, tc.mockAddr, tc.mockProto, tc.mockPeers)
			tc.expectedAssertBullyFunc(t, res)
			tc.expectedAssertErrorFunc(t, err)
		})
	}
}

func TestBully_Listen(t *testing.T) {
	testCases := []struct {
		name               string
		mockProto          string
		mockAddr           string
		expectedAssertFunc func(assert.TestingT, interface{}, ...interface{}) bool
	}{
		{
			"regular",
			"tcp4",
			"127.0.0.1:8100",
			assert.Nil,
		},
		{
			"badProto",
			"tcp22",
			"127.0.0.1:8101",
			assert.NotNil,
		},
		{
			"badAddr",
			"tcp6",
			"mockBadAddr:8102",
			assert.NotNil,
		},
		{
			"reserved_ports",
			"tcp4",
			"127.0.0.1:8",
			assert.NotNil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := mockBully("1", "1", "127.0.0.1")
			tc.expectedAssertFunc(t, b.Listen(tc.mockProto, tc.mockAddr))
		})
	}
}

func TestBully_connect(t *testing.T) {
	testCases := []struct {
		name               string
		mockProto          string
		mockAddr           string
		expectedAssertFunc func(assert.TestingT, interface{}, ...interface{}) bool
	}{
		{
			"regular",
			"tcp4",
			"127.0.0.1:8200",
			assert.Nil,
		},
		{
			"badProto",
			"tcp22",
			"127.0.0.1:8200",
			assert.NotNil,
		},
		{
			"badAddr",
			"tcp6",
			"127.0.0.1:9999",
			assert.NotNil,
		},
	}

	ms, err := newMockSocket("127.0.0.1:8200")
	assert.Nil(t, err)
	defer func() { _ = ms.Close() }()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := mockBully("1", "1", "127.0.0.1")
			tc.expectedAssertFunc(t, b.connect(tc.mockProto, tc.mockAddr, "1"))
		})
	}
}

func TestBully_Connect(t *testing.T) {
	testCases := []struct {
		name           string
		mockProto      string
		mockSocketAddr string
		mockPeers      map[string]string
	}{
		{
			"regular",
			"tcp4",
			"127.0.0.1:8300",
			map[string]string{
				"1": "127.0.0.1:8301",
				"2": "127.0.0.1:8302",
				"3": "127.0.0.1:8303",
			},
		},
		{
			"samePeers",
			"tcp4",
			"127.0.0.1:8310",
			map[string]string{
				"1": "127.0.0.1:8311",
				"2": "127.0.0.1:8311",
				"3": "127.0.0.1:8311",
			},
		},
		{
			"emptyMap",
			"tcp4",
			"127.0.0.1:8320",
			map[string]string{},
		},
		{
			"badProto",
			"tcp22",
			"127.0.0.1:8330",
			map[string]string{
				"1": "127.0.0.1:8331",
				"2": "127.0.0.1:8332",
				"3": "127.0.0.1:8333",
			},
		},
		{
			"badPeer",
			"tcp22",
			"127.0.0.1:8340",
			map[string]string{
				"1": "127.0.0.1:8341",
				"2": "notWorkingAddr",
				"3": "127.0.0.1:8343",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ms, err := newMockSocket(tc.mockSocketAddr)
			assert.Nil(t, err)
			defer func() { _ = ms.Close() }()

			b := mockBully("1", "1", "127.0.0.1")
			assert.NotPanics(t, func() { b.Connect(tc.mockProto, tc.mockPeers) })
		})
	}
}

func TestBully_SetCoordinator(t *testing.T) {
	testCases := []struct {
		name                string
		mockID              string
		mockPeerID          string
		expectedCoordinator string
	}{
		{"greater", "A", "B", "B"},
		{"less", "Zawarudo", "A", "Zawarudo"},
		{"equal", "same-id", "same-id", "same-id"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := mockBully(tc.mockID, tc.mockID, "127.0.0.1")
			b.SetCoordinator(tc.mockPeerID)
			assert.Equal(t, tc.expectedCoordinator, b.coordinator)
		})
	}
}

func TestBully_Coordinator(t *testing.T) {
	testCases := []struct {
		name                string
		mockCoordinator     string
		expectedCoordinator string
	}{
		{"regular", "A", "A"},
		{"empty_coordinator", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := mockBully("mockID", tc.mockCoordinator, "127.0.0.1")
			assert.Equal(t, tc.expectedCoordinator, b.Coordinator())
		})
	}
}

func TestBully_Send(t *testing.T) {
	testCases := []struct {
		name               string
		mockAddr           string
		mockTo             string
		mockPeerAddr       string
		mockPeers          map[string]string
		expectedAssertFunc func(assert.TestingT, interface{}, ...interface{}) bool
	}{
		{
			"regular",
			"127.0.0.1:8410",
			"2",
			"127.0.0.1:8412",
			map[string]string{
				"1": "127.0.0.1:8411",
				"2": "127.0.0.1:8400",
				"3": "127.0.0.1:8413",
			},
			assert.Nil,
		},
		{
			"peerNotFound",
			"127.0.0.1:8420",
			"2",
			"127.0.0.1:8400",
			map[string]string{
				"1": "127.0.0.1:8421",
				"3": "127.0.0.1:8423",
			},
			assert.Nil,
		},
		{
			"noRemoteHost",
			"127.0.0.1:8430",
			"2",
			"127.0.0.1:8439",
			map[string]string{
				"1": "127.0.0.1:8431",
				"3": "127.0.0.1:8433",
			},
			assert.NotNil,
		},
	}

	ms, err := newMockSocket("127.0.0.1:8400")
	assert.Nil(t, err)
	defer func() { _ = ms.Close() }()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := NewBully("mockId", tc.mockAddr, "tcp4", tc.mockPeers)
			assert.Nil(t, err)
			defer func() { _ = b.Close() }()

			tc.expectedAssertFunc(t, b.Send(tc.mockTo, tc.mockPeerAddr, 0))
		})
	}
}
func TestBully_Elect(t *testing.T) {
	testCases := []struct {
		name                string
		mockID              string
		mockCoordinator     string
		mockAddr            string
		mockPeers           map[string]string
		expectedMessageType int
		expectedCoordinator string
	}{
		{
			"peerCoordinator",
			"1",
			"2",
			"127.0.0.1:8511",
			map[string]string{
				"2": "127.0.0.1:8512",
			},
			ELECTION,
			"2",
		},
		{
			"selfCoordinator",
			"5",
			"2",
			"127.0.0.1:8521",
			map[string]string{
				"1": "127.0.0.1:8511",
				"2": "127.0.0.1:8512",
				"3": "127.0.0.1:8513",
			},
			COORDINATOR,
			"5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ms, err := newMockSocket("127.0.0.1:8512")
			go ms.accept()
			assert.Nil(t, err)
			defer func() { _ = ms.Close() }()

			b := mockBully(tc.mockID, tc.mockCoordinator, tc.mockAddr)
			defer func() { _ = b.Close() }()
			b.Connect("tcp4", tc.mockPeers)

			if tc.expectedMessageType == ELECTION {
				b.electionChan <- Message{}
			}
			b.Elect()

			select {
			case msg := <-ms.msgChan:
				assert.Equal(t, tc.expectedMessageType, msg.Type)
				break
			case <-time.After(3 * time.Second):
				t.Fail()
			}
			assert.Equal(t, tc.expectedCoordinator, b.coordinator)
		})
	}
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	os.Exit(m.Run())
}
