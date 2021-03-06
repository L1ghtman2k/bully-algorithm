package bully

import (
	"encoding/gob"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockSock is a testing `struct` used for writing & reading tests.
type mockSock struct {
	expectedMsg []byte
}

// newMockSock is a testing function returning a new `newMockSock`.
func newMockSock() *mockSock {
	return &mockSock{expectedMsg: make([]byte, 0)}
}

// Write is a testing function used to implement `io.Writer` interface.
func (ms *mockSock) Write(p []byte) (n int, err error) {
	for n = 0; n < len(p); n++ {
		ms.expectedMsg = append(ms.expectedMsg, p[n])
	}
	return n, nil
}

// Read is a testing function used to implement `io.Reader` interface.
func (ms *mockSock) Read(p []byte) (n int, err error) {
	copy(p, ms.expectedMsg)
	return len(ms.expectedMsg), nil
}

// mockPeerMap is a testing function used to generate populated a
// `*bully.PeerMap` containing `nb` elements with a maximum of 5 elements.
func mockPeerMap(nb int) *PeerMap {
	pm := NewPeerMap()
	mockPeers := []*Peer{
		&Peer{ID: "mock-1", addr: "40.87.127.215", sock: nil},
		&Peer{ID: "mock-2", addr: "84.72.203.27", sock: nil},
		&Peer{ID: "mock-3", addr: "232.65.164.182", sock: nil},
		&Peer{ID: "mock-4", addr: "135.68.39.183", sock: nil},
		&Peer{ID: "mock-5", addr: "65.74.170.184", sock: nil},
	}

	for i := 0; i < nb && i < 5; i++ {
		pm.Add(mockPeers[i].ID, mockPeers[i].addr, nil)
	}
	return pm
}

// -----------------------------------------------------------------------------

func TestPeerMap_NewPeerMap(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{"regular"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotNil(t, NewPeerMap())
		})
	}
}

func TestPeerMap_Add(t *testing.T) {
	testCases := []struct {
		name      string
		mockPeers []*Peer
	}{
		{
			"regular_single",
			[]*Peer{
				&Peer{ID: "single", addr: "127.0.0.1", sock: gob.NewEncoder(nil)},
			},
		},
		{
			"regular_multiple",
			[]*Peer{
				&Peer{ID: "multiple-1", addr: "40.87.127.215", sock: gob.NewEncoder(nil)},
				&Peer{ID: "multiple-2", addr: "84.72.203.27", sock: gob.NewEncoder(nil)},
				&Peer{ID: "multiple-3", addr: "232.65.164.182", sock: gob.NewEncoder(nil)},
			},
		},
		{"none_added", []*Peer{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			res := NewPeerMap()
			for _, mockPeer := range tc.mockPeers {
				res.Add(mockPeer.ID, mockPeer.addr, nil)
			}
			assert.Equal(t, len(tc.mockPeers), len(res.peers))

			expectedPeerList := make([]*Peer, 0)
			for _, expectedPeer := range res.peers {
				expectedPeerList = append(expectedPeerList, expectedPeer)
			}
			assert.ElementsMatch(t, expectedPeerList, tc.mockPeers)
		})
	}
}

func TestPeerMap_Delete(t *testing.T) {
	testCases := []struct {
		name         string
		mockIDs      []string
		expectedSize int
	}{
		{"delete_single", []string{"mock-2"}, 4},
		{"delete_multiple", []string{"mock-1", "mock-5"}, 3},
		{"delete_none", []string{}, 5},
		{"not_found", []string{"badPeerID"}, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm := mockPeerMap(5)
			for _, peerID := range tc.mockIDs {
				pm.Delete(peerID)
			}
			assert.Equal(t, len(pm.peers), tc.expectedSize)
		})
	}
}

func TestPeerMap_Find(t *testing.T) {
	testCases := []struct {
		name               string
		mockID             string
		mockPeerMapSize    int
		expectedAssertFunc func(assert.TestingT, bool, ...interface{}) bool
	}{
		{"found", "mock-2", 2, assert.True},
		{"not_found", "badID", 2, assert.False},
		{"empty", "mock-2", 0, assert.False},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm := mockPeerMap(tc.mockPeerMapSize)
			tc.expectedAssertFunc(t, pm.Find(tc.mockID))
		})
	}
}

func TestPeerMap_Write(t *testing.T) {
	testCases := []struct {
		name               string
		mockPeerID         string
		mockWriteID        string
		mockPeerMap        bool
		mockMessage        interface{}
		expectedMessage    string
		expectedAssertFunc func(assert.TestingT, interface{}, ...interface{}) bool
	}{
		{"regular_string", "0", "0", true, "ok", "ok", assert.Nil},
		{"empty_map", "1", "1", false, "ok", "", assert.NotNil},
		{"empty_message", "0", "0", true, "", "", assert.Nil},
		{"peerNotFound", "1", "50", true, "ok", "", assert.NotNil},
		{"badMessage", "0", "0", true, interface{}(nil), "ok", assert.NotNil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm := NewPeerMap()
			ms := newMockSock()

			if tc.mockPeerMap == true {
				pm.Add(tc.mockPeerID, "127.0.0.1", ms)
			}

			err := pm.Write(tc.mockWriteID, tc.mockMessage)
			tc.expectedAssertFunc(t, err)

			if err == nil {
				var res string
				dec := gob.NewDecoder(ms)
				assert.Nil(t, dec.Decode(&res))
				assert.Equal(t, tc.expectedMessage, res)
			}
		})
	}
}

func TestPeerMap_PeerData(t *testing.T) {
	testCases := []struct {
		name             string
		mockPeerMapSize  int
		expectedPeerInfo []struct {
			ID   string
			Addr string
		}
	}{
		{
			"single",
			1,
			[]struct {
				ID   string
				Addr string
			}{{"mock-1", "40.87.127.215"}},
		},
		{
			"multiple",
			2,
			[]struct {
				ID   string
				Addr string
			}{
				{"mock-1", "40.87.127.215"},
				{"mock-2", "84.72.203.27"},
			},
		},
		{
			"empty",
			0,
			[]struct {
				ID   string
				Addr string
			}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm := mockPeerMap(tc.mockPeerMapSize)
			assert.Equal(t, len(tc.expectedPeerInfo), len(pm.PeerData()))
			assert.ElementsMatch(t, tc.expectedPeerInfo, pm.PeerData())
		})
	}
}
