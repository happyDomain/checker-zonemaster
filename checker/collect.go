package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

func (p *zonemasterProvider) Collect(ctx context.Context, opts sdk.CheckerOptions) (any, error) {
	domainName, ok := opts["domainName"].(string)
	if !ok || domainName == "" {
		return nil, fmt.Errorf("domainName is required")
	}
	domainName = strings.TrimSuffix(domainName, ".")

	apiURL, ok := opts["zonemasterAPIURL"].(string)
	if !ok || apiURL == "" {
		apiURL = "https://zonemaster.net/api"
	}
	apiURL = strings.TrimSuffix(apiURL, "/")

	language := "en"
	if lang, ok := opts["language"].(string); ok && lang != "" {
		language = lang
	}

	profile := "default"
	if prof, ok := opts["profile"].(string); ok && prof != "" {
		profile = prof
	}

	// Step 1: start the test.
	startResult, err := zmCallJSONRPC(ctx, apiURL, "start_domain_test", zmStartTestParams{
		Domain:  domainName,
		Profile: profile,
		IPv4:    true,
		IPv6:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start test: %w", err)
	}

	var testID string
	if err = json.Unmarshal(startResult, &testID); err != nil {
		return nil, fmt.Errorf("failed to parse test ID: %w", err)
	}
	if testID == "" {
		return nil, fmt.Errorf("received empty test ID")
	}

	// Step 2: poll for completion.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("test cancelled (test ID: %s): %w", testID, ctx.Err())
		case <-ticker.C:
			progressResult, err := zmCallJSONRPC(ctx, apiURL, "test_progress", zmProgressParams{TestID: testID})
			if err != nil {
				return nil, fmt.Errorf("failed to check progress: %w", err)
			}

			var progress float64
			if err := json.Unmarshal(progressResult, &progress); err != nil {
				return nil, fmt.Errorf("failed to parse progress: %w", err)
			}

			if progress >= 100 {
				goto testComplete
			}
		}
	}

testComplete:
	// Step 3: fetch results.
	rawResults, err := zmCallJSONRPC(ctx, apiURL, "get_test_results", zmGetResultsParams{
		ID:       testID,
		Language: language,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}

	var data ZonemasterData
	if err := json.Unmarshal(rawResults, &data); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}
	data.Language = language

	return &data, nil
}

// zmCallJSONRPC performs a single JSON-RPC 2.0 call and returns the raw result.
func zmCallJSONRPC(ctx context.Context, apiURL, method string, params any) (json.RawMessage, error) {
	body, err := json.Marshal(zmJSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(b))
	}

	var rpcResp zmJSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("API error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}
