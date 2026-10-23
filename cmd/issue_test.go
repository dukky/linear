package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/dukky/linear/internal/client"
)

func TestBuildRelationInputs(t *testing.T) {
	ids := map[string]string{
		"ENG-2": "uuid-2",
		"ENG-3": "uuid-3",
		"ENG-4": "uuid-4",
		"ENG-5": "uuid-5",
	}
	resolve := func(id string) (string, error) {
		if uuid, ok := ids[id]; ok {
			return uuid, nil
		}
		return "", errors.New("issue not found: " + id)
	}
	req := func(issueID, relatedIssueID, typ, target string) relationRequest {
		return relationRequest{
			Input:  client.CreateIssueRelationInput{IssueID: issueID, RelatedIssueID: relatedIssueID, Type: typ},
			Target: target,
		}
	}

	tests := []struct {
		name      string
		blocks    []string
		blockedBy []string
		related   []string
		want      []relationRequest
		wantErr   string
	}{
		{name: "none", want: []relationRequest{}},
		{
			name:   "blocks: current issue blocks target",
			blocks: []string{"ENG-2"},
			want:   []relationRequest{req("uuid-1", "uuid-2", "blocks", "ENG-2")},
		},
		{
			name:      "blocked-by: target blocks current issue",
			blockedBy: []string{"ENG-2"},
			want:      []relationRequest{req("uuid-2", "uuid-1", "blocks", "ENG-2")},
		},
		{
			name:    "related",
			related: []string{"ENG-2"},
			want:    []relationRequest{req("uuid-1", "uuid-2", "related", "ENG-2")},
		},
		{
			name:      "ordering: blocks, then blocked-by, then related, each in flag order",
			related:   []string{"ENG-5"},
			blockedBy: []string{"ENG-4"},
			blocks:    []string{"ENG-3", "ENG-2"},
			want: []relationRequest{
				req("uuid-1", "uuid-3", "blocks", "ENG-3"),
				req("uuid-1", "uuid-2", "blocks", "ENG-2"),
				req("uuid-4", "uuid-1", "blocks", "ENG-4"),
				req("uuid-1", "uuid-5", "related", "ENG-5"),
			},
		},
		{
			name:    "resolver error in blocks",
			blocks:  []string{"ENG-2", "ENG-404"},
			wantErr: "resolving --blocks issue ENG-404: issue not found: ENG-404",
		},
		{
			name:      "resolver error in blocked-by",
			blockedBy: []string{"ENG-404"},
			wantErr:   "resolving --blocked-by issue ENG-404: issue not found: ENG-404",
		},
		{
			name:    "resolver error in related",
			related: []string{"ENG-404"},
			wantErr: "resolving --related issue ENG-404: issue not found: ENG-404",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildRelationInputs("uuid-1", tt.blocks, tt.blockedBy, tt.related, resolve)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				if got != nil {
					t.Fatalf("expected no inputs on error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

// graphQLRequest is a GraphQL request as decoded by linearStub.
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

// operationName returns the first root field selected by a GraphQL document,
// e.g. "issue", "teams", "issueUpdate" or "issueRelationCreate".
func operationName(query string) string {
	brace := strings.Index(query, "{")
	if brace == -1 {
		return ""
	}
	rest := strings.TrimLeftFunc(query[brace+1:], unicode.IsSpace)
	end := strings.IndexFunc(rest, func(r rune) bool {
		return r == '(' || r == '{' || unicode.IsSpace(r)
	})
	if end == -1 {
		return rest
	}
	return rest[:end]
}

// linearStub fakes the Linear GraphQL API for cmd tests. It records every
// decoded request in order and replies by operation name.
type linearStub struct {
	t        *testing.T
	handlers map[string]func(vars map[string]interface{}) string
	requests []graphQLRequest
}

// stubLinearAPI routes every request made through http.DefaultTransport (which
// client.NewClient uses) to a linearStub, so command code can be exercised
// without touching the network. Each handler returns the JSON "data"
// payload for its operation; an operation without a handler fails the test.
// LINEAR_API_KEY is set so NewClient never reads real credentials, and the
// original transport is restored on cleanup.
func stubLinearAPI(t *testing.T, handlers map[string]func(vars map[string]interface{}) string) *linearStub {
	t.Helper()
	t.Setenv("LINEAR_API_KEY", "test-key")

	stub := &linearStub{t: t, handlers: handlers}

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	http.DefaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		stub.requests = append(stub.requests, req)

		data := `{}`
		name := operationName(req.Query)
		if handler, ok := stub.handlers[name]; ok {
			data = handler(req.Variables)
		} else {
			t.Errorf("unexpected operation %q in query: %s", name, req.Query)
		}

		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/json")
		rec.WriteString(`{"data":` + data + `}`)
		return rec.Result(), nil
	})

	return stub
}

// requestsFor returns the recorded requests for one operation, in order.
func (s *linearStub) requestsFor(name string) []graphQLRequest {
	var out []graphQLRequest
	for _, req := range s.requests {
		if operationName(req.Query) == name {
			out = append(out, req)
		}
	}
	return out
}

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// fakeIssueByIdentifier answers issue(id:) lookups for ENG-<n> with UUID
// uuid-<n> and title "Title <n>".
func fakeIssueByIdentifier(vars map[string]interface{}) string {
	id, _ := vars["id"].(string)
	n := strings.TrimPrefix(id, "ENG-")
	return fmt.Sprintf(`{"issue":{"id":"uuid-%s","identifier":"ENG-%s","title":"Title %s","url":"https://linear.app/test/issue/ENG-%s"}}`, n, n, n, n)
}

// fakeRelationCreate echoes a successful issueRelationCreate for its input,
// mapping uuid-<n> back to ENG-<n>.
func fakeRelationCreate(vars map[string]interface{}) string {
	input, _ := vars["input"].(map[string]interface{})
	issueID, _ := input["issueId"].(string)
	relatedID, _ := input["relatedIssueId"].(string)
	ident := func(uuid string) string { return "ENG-" + strings.TrimPrefix(uuid, "uuid-") }
	return fmt.Sprintf(`{"issueRelationCreate":{"success":true,"issueRelation":{"id":"rel-1","type":%q,"issue":{"id":%q,"identifier":%q,"title":"x"},"relatedIssue":{"id":%q,"identifier":%q,"title":"y"}}}}`,
		input["type"], issueID, ident(issueID), relatedID, ident(relatedID))
}

func fakeIssueUpdate(vars map[string]interface{}) string {
	return `{"issueUpdate":{"success":true,"issue":{"id":"uuid-1","identifier":"ENG-1","title":"Title 1","url":"https://linear.app/test/issue/ENG-1"}}}`
}

// issueUpdateFlagNames are the flags registered on issueUpdateCmd.
var issueUpdateFlagNames = []string{"title", "description", "priority", "project", "assignee", "state", "parent", "blocks", "blocked-by", "related"}

// setIssueUpdateFlags parses args into issueUpdateCmd's flags after resetting
// every update flag (values and Changed), and resets them again on cleanup so
// state does not leak between tests sharing the package-level command.
func setIssueUpdateFlags(t *testing.T, args ...string) {
	t.Helper()
	reset := func() {
		issueUpdateTitle, issueUpdateDesc, issueUpdateProject = "", "", ""
		issueUpdateAssignee, issueUpdateState, issueUpdateParent = "", "", ""
		issueUpdatePriority = 0
		issueUpdateBlocks, issueUpdateBlockedBy, issueUpdateRelated = nil, nil, nil
		for _, name := range issueUpdateFlagNames {
			flag := issueUpdateCmd.Flags().Lookup(name)
			if flag == nil {
				t.Fatalf("issue update has no --%s flag", name)
			}
			flag.Changed = false
		}
	}
	originalJSON := jsonOutput
	reset()
	jsonOutput = false
	t.Cleanup(func() {
		reset()
		jsonOutput = originalJSON
	})
	if err := issueUpdateCmd.Flags().Parse(args); err != nil {
		t.Fatalf("parsing flags %v: %v", args, err)
	}
}

func TestIssueCreate_ParentResolvedToUUID(t *testing.T) {
	originalTitle, originalTeam, originalParent, originalJSON := issueTitle, issueTeamID, issueParent, jsonOutput
	t.Cleanup(func() {
		issueTitle, issueTeamID, issueParent, jsonOutput = originalTitle, originalTeam, originalParent, originalJSON
	})
	issueTitle, issueTeamID, issueParent, jsonOutput = "Sub-task", "ENG", "ENG-10", false

	stub := stubLinearAPI(t, map[string]func(map[string]interface{}) string{
		"teams": func(map[string]interface{}) string {
			return `{"teams":{"nodes":[{"id":"team-uuid","key":"ENG","name":"Engineering"}]}}`
		},
		"issue": fakeIssueByIdentifier,
		"issueCreate": func(map[string]interface{}) string {
			return `{"issueCreate":{"success":true,"issue":{"id":"uuid-11","identifier":"ENG-11","title":"Sub-task","url":"https://linear.app/test/issue/ENG-11"}}}`
		},
	})

	out := captureStdout(t, func() {
		issueCreateCmd.Run(issueCreateCmd, nil)
	})

	lookups := stub.requestsFor("issue")
	if len(lookups) != 1 || lookups[0].Variables["id"] != "ENG-10" {
		t.Fatalf("expected one parent lookup for ENG-10, got %+v", lookups)
	}
	creates := stub.requestsFor("issueCreate")
	if len(creates) != 1 {
		t.Fatalf("expected one issueCreate, got %d; output: %q", len(creates), out)
	}
	createInput, _ := creates[0].Variables["input"].(map[string]interface{})
	if createInput["parentId"] != "uuid-10" {
		t.Fatalf("expected issueCreate input parentId %q (the parent's UUID, not its identifier), got %v", "uuid-10", createInput["parentId"])
	}
	if !strings.Contains(out, "ID:    ENG-11") {
		t.Fatalf("expected created issue in output, got %q", out)
	}
	if !strings.Contains(out, "Parent: ENG-10 Title 10") {
		t.Fatalf("expected parent line in output, got %q", out)
	}
}

func TestIssueUpdate_RelationsOnlySkipsUpdate(t *testing.T) {
	setIssueUpdateFlags(t, "--blocks", "ENG-2")
	stub := stubLinearAPI(t, map[string]func(map[string]interface{}) string{
		"issue":               fakeIssueByIdentifier,
		"issueUpdate":         fakeIssueUpdate,
		"issueRelationCreate": fakeRelationCreate,
	})

	var err error
	out := captureStdout(t, func() {
		err = runIssueUpdate(issueUpdateCmd, []string{"ENG-1"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updates := stub.requestsFor("issueUpdate"); len(updates) != 0 {
		t.Fatalf("expected no issueUpdate mutation for a relations-only update, got %d: %+v", len(updates), updates)
	}

	relations := stub.requestsFor("issueRelationCreate")
	if len(relations) != 1 {
		t.Fatalf("expected exactly one issueRelationCreate, got %d", len(relations))
	}
	want := map[string]interface{}{"issueId": "uuid-1", "relatedIssueId": "uuid-2", "type": "blocks"}
	if got := relations[0].Variables["input"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("issueRelationCreate input: got %v, want %v", got, want)
	}

	currentFetches := 0
	for _, req := range stub.requestsFor("issue") {
		if req.Variables["id"] == "ENG-1" {
			currentFetches++
		}
	}
	if currentFetches != 1 {
		t.Fatalf("expected the current issue to be fetched once, got %d", currentFetches)
	}

	if !strings.Contains(out, "Relation created: ENG-1 blocks ENG-2") {
		t.Fatalf("expected relation line in output, got %q", out)
	}
}

func TestIssueUpdate_ParentOnlySendsUpdate(t *testing.T) {
	setIssueUpdateFlags(t, "--parent", "ENG-3")
	stub := stubLinearAPI(t, map[string]func(map[string]interface{}) string{
		"issue":               fakeIssueByIdentifier,
		"issueUpdate":         fakeIssueUpdate,
		"issueRelationCreate": fakeRelationCreate,
	})

	var err error
	out := captureStdout(t, func() {
		err = runIssueUpdate(issueUpdateCmd, []string{"ENG-1"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updates := stub.requestsFor("issueUpdate")
	if len(updates) != 1 {
		t.Fatalf("expected exactly one issueUpdate, got %d", len(updates))
	}
	if updates[0].Variables["id"] != "ENG-1" {
		t.Fatalf("expected issueUpdate for ENG-1, got %v", updates[0].Variables["id"])
	}
	want := map[string]interface{}{"parentId": "uuid-3"}
	if got := updates[0].Variables["input"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("issueUpdate input: got %v, want %v (parentId only)", got, want)
	}

	if relations := stub.requestsFor("issueRelationCreate"); len(relations) != 0 {
		t.Fatalf("expected no issueRelationCreate, got %d", len(relations))
	}

	if !strings.Contains(out, "Parent: ENG-3 Title 3") {
		t.Fatalf("expected parent line in output, got %q", out)
	}
}

func TestIssueUpdate_RelationErrorNamesIdentifier(t *testing.T) {
	setIssueUpdateFlags(t, "--blocks", "ENG-2")
	stub := stubLinearAPI(t, map[string]func(map[string]interface{}) string{
		"issue":       fakeIssueByIdentifier,
		"issueUpdate": fakeIssueUpdate,
		"issueRelationCreate": func(map[string]interface{}) string {
			return `{"issueRelationCreate":{"success":false,"issueRelation":null}}`
		},
	})

	var err error
	captureStdout(t, func() {
		err = runIssueUpdate(issueUpdateCmd, []string{"ENG-1"})
	})

	if len(stub.requestsFor("issueRelationCreate")) != 1 {
		t.Fatalf("expected the relation mutation to be attempted once")
	}
	if err == nil {
		t.Fatal("expected an error when issueRelationCreate reports success: false")
	}
	const want = "Error: Failed to create blocks relation with ENG-2"
	if err.Error() != want {
		t.Fatalf("error: got %q, want %q", err.Error(), want)
	}
	if strings.Contains(err.Error(), "uuid-") {
		t.Fatalf("error must name the identifier the user typed, not a UUID: %q", err.Error())
	}
}

func TestIssueUpdate_JSONRefetchesRelations(t *testing.T) {
	setIssueUpdateFlags(t, "--related", "ENG-2")
	jsonOutput = true // setIssueUpdateFlags restores the original on cleanup

	// Until the relation exists, ENG-1 is returned without it; afterwards the
	// lookup includes it. Only a re-fetch after issueRelationCreate can print
	// the new relation.
	relationCreated := false
	stub := stubLinearAPI(t, map[string]func(map[string]interface{}) string{
		"issue": func(vars map[string]interface{}) string {
			if vars["id"] == "ENG-1" && relationCreated {
				return `{"issue":{"id":"uuid-1","identifier":"ENG-1","title":"Title 1","url":"https://linear.app/test/issue/ENG-1",` +
					`"relations":{"nodes":[{"id":"rel-1","type":"related",` +
					`"issue":{"id":"uuid-1","identifier":"ENG-1","title":"Title 1"},` +
					`"relatedIssue":{"id":"uuid-2","identifier":"ENG-2","title":"Title 2"}}]},` +
					`"inverseRelations":{"nodes":[]}}}`
			}
			return fakeIssueByIdentifier(vars)
		},
		"issueUpdate": fakeIssueUpdate,
		"issueRelationCreate": func(vars map[string]interface{}) string {
			relationCreated = true
			return fakeRelationCreate(vars)
		},
	})

	var err error
	out := captureStdout(t, func() {
		err = runIssueUpdate(issueUpdateCmd, []string{"ENG-1"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	relationIndex := -1
	refetched := false
	for i, req := range stub.requests {
		switch operationName(req.Query) {
		case "issueRelationCreate":
			relationIndex = i
		case "issue":
			if relationIndex != -1 && req.Variables["id"] == "ENG-1" {
				refetched = true
			}
		}
	}
	if relationIndex == -1 {
		t.Fatal("expected an issueRelationCreate mutation")
	}
	if !refetched {
		t.Fatalf("expected ENG-1 to be re-fetched after issueRelationCreate; requests: %d", len(stub.requests))
	}

	var printed client.Issue
	if err := json.Unmarshal([]byte(out), &printed); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out, err)
	}
	if printed.Relations == nil || len(printed.Relations.Nodes) != 1 {
		t.Fatalf("expected the printed issue to carry the new relation in relations.nodes, got output: %s", out)
	}
	rel := printed.Relations.Nodes[0]
	if rel.Type != "related" || rel.RelatedIssue == nil || rel.RelatedIssue.Identifier != "ENG-2" {
		t.Fatalf("expected a related relation to ENG-2, got %+v", rel)
	}
}

func TestFormatRelations(t *testing.T) {
	ref := func(id, identifier string) *client.IssueRef {
		return &client.IssueRef{ID: id, Identifier: identifier}
	}
	conn := func(rels ...client.IssueRelation) *client.IssueRelationConnection {
		return &client.IssueRelationConnection{Nodes: rels}
	}
	self := ref("uuid-1", "ENG-1")

	tests := []struct {
		name     string
		outgoing *client.IssueRelationConnection
		incoming *client.IssueRelationConnection
		want     []string
	}{
		{name: "nil connections (not selected)"},
		{name: "empty connections", outgoing: conn(), incoming: conn()},
		{
			name:     "outgoing blocks names the related issue",
			outgoing: conn(client.IssueRelation{ID: "r1", Type: "blocks", Issue: self, RelatedIssue: ref("uuid-2", "ENG-2")}),
			want:     []string{"blocks ENG-2"},
		},
		{
			name:     "incoming blocks reads blocked by the issue end",
			incoming: conn(client.IssueRelation{ID: "r1", Type: "blocks", Issue: ref("uuid-3", "ENG-3"), RelatedIssue: self}),
			want:     []string{"blocked by ENG-3"},
		},
		{
			name:     "outgoing related names the related issue",
			outgoing: conn(client.IssueRelation{ID: "r1", Type: "related", Issue: self, RelatedIssue: ref("uuid-4", "ENG-4")}),
			want:     []string{"related ENG-4"},
		},
		{
			name:     "incoming related names the issue end",
			incoming: conn(client.IssueRelation{ID: "r1", Type: "related", Issue: ref("uuid-5", "ENG-5"), RelatedIssue: self}),
			want:     []string{"related ENG-5"},
		},
		{
			name:     "other types keep their direction",
			outgoing: conn(client.IssueRelation{ID: "r1", Type: "duplicate", Issue: self, RelatedIssue: ref("uuid-6", "ENG-6")}),
			incoming: conn(client.IssueRelation{ID: "r2", Type: "similar", Issue: ref("uuid-7", "ENG-7"), RelatedIssue: self}),
			want:     []string{"duplicate ENG-6", "similar from ENG-7"},
		},
		{
			name: "outgoing first, each list in order",
			outgoing: conn(
				client.IssueRelation{ID: "r1", Type: "blocks", Issue: self, RelatedIssue: ref("uuid-2", "ENG-2")},
				client.IssueRelation{ID: "r2", Type: "related", Issue: self, RelatedIssue: ref("uuid-4", "ENG-4")},
			),
			incoming: conn(
				client.IssueRelation{ID: "r3", Type: "blocks", Issue: ref("uuid-3", "ENG-3"), RelatedIssue: self},
				client.IssueRelation{ID: "r4", Type: "related", Issue: ref("uuid-5", "ENG-5"), RelatedIssue: self},
			),
			want: []string{"blocks ENG-2", "related ENG-4", "blocked by ENG-3", "related ENG-5"},
		},
		{
			name:     "nil outgoing with incoming",
			incoming: conn(client.IssueRelation{ID: "r1", Type: "blocks", Issue: ref("uuid-3", "ENG-3"), RelatedIssue: self}),
			want:     []string{"blocked by ENG-3"},
		},
		{
			name:     "same edge in both lists is printed once",
			outgoing: conn(client.IssueRelation{ID: "r1", Type: "blocks", Issue: self, RelatedIssue: ref("uuid-2", "ENG-2")}),
			incoming: conn(client.IssueRelation{ID: "r1", Type: "blocks", Issue: self, RelatedIssue: ref("uuid-2", "ENG-2")}),
			want:     []string{"blocks ENG-2"},
		},
		{
			name:     "missing endpoint renders as ?",
			outgoing: conn(client.IssueRelation{ID: "r1", Type: "blocks", Issue: self}),
			incoming: conn(client.IssueRelation{ID: "r2", Type: "blocks", RelatedIssue: self}),
			want:     []string{"blocks ?", "blocked by ?"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRelations(tt.outgoing, tt.incoming)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}
