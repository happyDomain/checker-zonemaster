package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

// maxResponseBytes caps the body size we'll read from the Zonemaster API.
// Real result payloads are tens to a few hundred KB; 8 MiB is generous head-
// room and still bounded so a misbehaving or hostile endpoint can't exhaust
// memory.
const maxResponseBytes = 8 << 20

// maxCollectDuration caps the total time spent collecting (start + poll +
// fetch). The caller's context still wins if it has a tighter deadline.
const maxCollectDuration = 15 * time.Minute

// pollInterval is how often we ask the Zonemaster API for test progress.
const pollInterval = 2 * time.Second

// zmHTTPClient is the HTTP client used for all Zonemaster API calls. It has
// per-phase timeouts so a stalling endpoint can never hang us indefinitely
// even if the caller passes a context without a deadline.
var zmHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	},
}

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

	// Cap the total collection time even when the caller's context has no
	// deadline. The caller's deadline still wins if it's tighter.
	ctx, cancel := context.WithTimeout(ctx, maxCollectDuration)
	defer cancel()

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
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

poll:
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
				break poll
			}
		}
	}

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

	resp, err := zmHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	// Cap the body we'll ever read so a misbehaving endpoint can't exhaust
	// memory. +1 lets us detect that the cap was hit.
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)

	if resp.StatusCode != http.StatusOK {
		b, readErr := io.ReadAll(limited)
		if readErr != nil {
			return nil, fmt.Errorf("API returned status %d (failed to read body: %v)", resp.StatusCode, readErr)
		}
		if len(b) > maxResponseBytes {
			b = b[:maxResponseBytes]
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(b))
	}

	body, readErr := io.ReadAll(limited)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response: %w", readErr)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("API response exceeds %d bytes", maxResponseBytes)
	}

	var rpcResp zmJSONRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("API error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}
