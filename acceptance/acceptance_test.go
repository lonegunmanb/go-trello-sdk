// Package acceptance contains read-only acceptance tests that run against the
// live Trello REST API. They are gated behind the ``TRELLO_API_KEY`` and
// ``TRELLO_API_TOKEN`` environment variables — when either is missing, the
// tests are skipped so ``go test ./...`` remains green for contributors who
// don't have credentials.
//
// All operations exercised by these tests are read-only:
//
//   - GET /members/me
//   - GET /members/me/boards
//   - GET /boards/{id}
//   - GET /boards/{id}/lists
//
// No write methods are invoked, so running the suite cannot modify any
// existing Trello data.
package acceptance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/lonegunmanb/go-trello-sdk/trello"
)

const callTimeout = 30 * time.Second

// newClient constructs a Trello client using credentials from the
// environment, or skips the test if they are not available.
//
// We deliberately use the raw client methods (those that return
// ``*http.Response``) rather than the typed ``…WithResponse`` helpers because
// the upstream Trello OpenAPI spec marks several response fields (e.g.
// ``dateLastActivity``) as ``format: date`` while the live API returns full
// RFC3339 date-times. The generated ``Parse…Response`` functions therefore
// fail with ``parsing time "…": extra text: "T…Z"``. Decoding the body
// ourselves into the small structs we actually care about avoids the issue
// entirely and keeps these acceptance tests resilient to spec drift.
func newClient(t *testing.T) *trello.ClientWithResponses {
	t.Helper()
	key := os.Getenv("TRELLO_API_KEY")
	token := os.Getenv("TRELLO_API_TOKEN")
	if key == "" || token == "" {
		t.Skip("TRELLO_API_KEY and TRELLO_API_TOKEN must both be set to run acceptance tests")
	}
	c, err := trello.New(trello.WithCredentials(key, token))
	if err != nil {
		t.Fatalf("trello.New: %v", err)
	}
	return c
}

// readBody closes the response body and returns its bytes.
func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return b
}

// TestGetMe calls GET /members/me and verifies that the response carries an
// id field. This is the canonical "auth works" smoke test.
func TestGetMe(t *testing.T) {
	c := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()

	resp, err := c.GetMembersId(ctx, "me", &trello.GetMembersIdParams{})
	if err != nil {
		t.Fatalf("GetMembersId: %v", err)
	}
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /members/me returned %d: %s", resp.StatusCode, string(body))
	}

	var member struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &member); err != nil {
		t.Fatalf("decode body: %v\nbody: %s", err, string(body))
	}
	if member.ID == "" {
		t.Fatalf("member.id is empty: %s", string(body))
	}
	t.Logf("authenticated as %s (%s)", member.Username, member.ID)
}

// TestGetMyBoards calls GET /members/me/boards and verifies it returns a JSON
// array. The list is allowed to be empty.
func TestGetMyBoards(t *testing.T) {
	c := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()

	resp, err := c.GetMembersIdBoards(ctx, "me", &trello.GetMembersIdBoardsParams{})
	if err != nil {
		t.Fatalf("GetMembersIdBoards: %v", err)
	}
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /members/me/boards returned %d: %s", resp.StatusCode, string(body))
	}

	var boards []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Closed bool   `json:"closed"`
	}
	if err := json.Unmarshal(body, &boards); err != nil {
		t.Fatalf("decode body: %v\nbody: %s", err, string(body))
	}
	t.Logf("found %d board(s)", len(boards))
	for _, b := range boards {
		if b.ID == "" {
			t.Errorf("board has empty id: %+v", b)
		}
	}
}

// TestGetBoardDetails picks the first available board and reads its details
// and lists. If the test account has no boards we skip rather than fail.
func TestGetBoardDetails(t *testing.T) {
	c := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()

	listResp, err := c.GetMembersIdBoards(ctx, "me", &trello.GetMembersIdBoardsParams{})
	if err != nil {
		t.Fatalf("list boards: %v", err)
	}
	listBody := readBody(t, listResp)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list boards returned %d: %s", listResp.StatusCode, string(listBody))
	}
	var boards []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(listBody, &boards); err != nil {
		t.Fatalf("decode boards: %v", err)
	}
	if len(boards) == 0 {
		t.Skip("test account has no boards; skipping board-detail test")
	}

	target := boards[0]
	t.Logf("inspecting board %q (%s)", target.Name, target.ID)

	boardResp, err := c.GetBoardsId(ctx, target.ID, &trello.GetBoardsIdParams{})
	if err != nil {
		t.Fatalf("GetBoardsId: %v", err)
	}
	boardBody := readBody(t, boardResp)
	if boardResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /boards/%s returned %d: %s", target.ID, boardResp.StatusCode, string(boardBody))
	}
	var board struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(boardBody, &board); err != nil {
		t.Fatalf("decode board: %v", err)
	}
	if board.ID != target.ID {
		t.Errorf("board id mismatch: got %q want %q", board.ID, target.ID)
	}

	listsResp, err := c.GetBoardsIdLists(ctx, target.ID, &trello.GetBoardsIdListsParams{})
	if err != nil {
		t.Fatalf("GetBoardsIdLists: %v", err)
	}
	listsBody := readBody(t, listsResp)
	if listsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /boards/%s/lists returned %d: %s", target.ID, listsResp.StatusCode, string(listsBody))
	}
	var lists []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(listsBody, &lists); err != nil {
		t.Fatalf("decode lists: %v", err)
	}
	t.Logf("board %q has %d list(s)", target.Name, len(lists))
}
