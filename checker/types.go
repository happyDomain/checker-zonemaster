package checker

import (
	"encoding/json"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

// ObservationKeyZonemaster is the observation key for Zonemaster test data.
const ObservationKeyZonemaster sdk.ObservationKey = "zonemaster"

// ── JSON-RPC structures ────────────────────────────────────────────────────────

type zmJSONRPCRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	ID      int    `json:"id"`
}

type zmJSONRPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	ID int `json:"id"`
}

// ── Zonemaster API parameter types ────────────────────────────────────────────

type zmStartTestParams struct {
	Domain  string `json:"domain"`
	Profile string `json:"profile,omitempty"`
	IPv4    bool   `json:"ipv4,omitempty"`
	IPv6    bool   `json:"ipv6,omitempty"`
}

type zmProgressParams struct {
	TestID string `json:"test_id"`
}

type zmGetResultsParams struct {
	ID       string `json:"id"`
	Language string `json:"language"`
}

// ── Observation data types ─────────────────────────────────────────────────────

// ZonemasterTestResult is a single result entry returned by the Zonemaster API.
type ZonemasterTestResult struct {
	Module   string `json:"module"`
	Message  string `json:"message"`
	Level    string `json:"level"`
	Testcase string `json:"testcase,omitempty"`
}

// ZonemasterData holds the full Zonemaster test output stored as an observation.
type ZonemasterData struct {
	CreatedAt            string                 `json:"created_at"`
	HashID               string                 `json:"hash_id"`
	Language             string                 `json:"language,omitempty"`
	Params               map[string]any         `json:"params"`
	Results              []ZonemasterTestResult `json:"results"`
	TestcaseDescriptions map[string]string      `json:"testcase_descriptions,omitempty"`
}
