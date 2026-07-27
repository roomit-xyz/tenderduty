package tenderduty

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// GnoStatusResult represents the /status RPC response from a TM2 node.
type GnoStatusResult struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  struct {
		NodeInfo struct {
			Network string `json:"network"`
		} `json:"node_info"`
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
			CatchingUp        bool   `json:"catching_up"`
		} `json:"sync_info"`
	} `json:"result"`
}

// GnoValidatorsResult represents the /validators RPC response.
type GnoValidatorsResult struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      interface{}    `json:"id"`
	Result  struct {
		Validators []GnoValidator `json:"validators"`
	} `json:"result"`
}

// GnoValidator represents a single validator from /validators.
type GnoValidator struct {
	Address          string `json:"address"`
	PubKey           struct {
		Type  string `json:"@type"`
		Value string `json:"value"`
	} `json:"pub_key"`
	VotingPower      string `json:"voting_power"`
	ProposerPriority string `json:"proposer_priority"`
}

// GnoABCIResult represents an abci_query RPC response.
type GnoABCIResult struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  *struct {
		Response struct {
			ResponseBase struct {
				Data  string          `json:"Data"`
				Log   string          `json:"Log"`
				Error json.RawMessage `json:"Error"`
			} `json:"ResponseBase"`
		} `json:"response"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"error"`
}

// gnoHTTPGet performs a GET request with a timeout.
func gnoHTTPGet(url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GnoGetStatus fetches /status from a TM2 RPC node.
func GnoGetStatus(rpcURL string) (chainID string, height string, catchingUp bool, err error) {
	url := strings.TrimRight(rpcURL, "/") + "/status?"
	body, err := gnoHTTPGet(url, 10*time.Second)
	if err != nil {
		return "", "", false, fmt.Errorf("status request failed: %w", err)
	}
	var result GnoStatusResult
	if err = json.Unmarshal(body, &result); err != nil {
		return "", "", false, fmt.Errorf("status unmarshal failed: %w", err)
	}
	return result.Result.NodeInfo.Network,
		result.Result.SyncInfo.LatestBlockHeight,
		result.Result.SyncInfo.CatchingUp,
		nil
}

// GnoGetValidators fetches the active validator set.
func GnoGetValidators(rpcURL string) ([]GnoValidator, error) {
	url := strings.TrimRight(rpcURL, "/") + "/validators?per_page=100"
	body, err := gnoHTTPGet(url, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("validators request failed: %w", err)
	}
	var result GnoValidatorsResult
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("validators unmarshal failed: %w", err)
	}
	return result.Result.Validators, nil
}

// GnoIsValidatorActive checks if the given address is in the active validator set.
func GnoIsValidatorActive(rpcURL string, address string) (bool, error) {
	vals, err := GnoGetValidators(rpcURL)
	if err != nil {
		return false, err
	}
	addrLower := strings.ToLower(address)
	for _, v := range vals {
		if strings.ToLower(v.Address) == addrLower {
			return true, nil
		}
	}
	return false, nil
}

var valoperMonikerRex = regexp.MustCompile(`\("([^"]*)" string\)`)
var valoperAddrRex = regexp.MustCompile(`"(g1[a-z0-9]+)" \.uverse\.address`)

// GnoResolveConsensusAddr resolves a valoper address to its consensus address
// via the valopers realm vm/qeval query. Returns empty string if not found,
// never panics.
func GnoResolveConsensusAddr(rpcURL, realmPath, accountAddr string) (string, error) {
	if realmPath == "" {
		realmPath = "gno.land/r/gnops/valopers"
	}
	expr := fmt.Sprintf(`%s.GetByAddr("%s")`, realmPath, accountAddr)
	dataHex := hex.EncodeToString([]byte(expr))
	qurl := fmt.Sprintf(`%s/abci_query?path=%%22vm/qeval%%22&data=0x%s`,
		strings.TrimRight(rpcURL, "/"), dataHex)
	body, err := gnoHTTPGet(qurl, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("qeval request failed: %w", err)
	}
	var result GnoABCIResult
	if err = json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("qeval unmarshal failed: %w", err)
	}
	if result.Error != nil {
		if strings.Contains(result.Error.Message, "valoper does not exist") {
			return "", errors.New("validator not registered in valopers realm")
		}
		return "", fmt.Errorf("qeval error: %s", result.Error.Message)
	}
	if result.Result == nil {
		return "", errors.New("qeval: nil result")
	}
	dataB64 := result.Result.Response.ResponseBase.Data
	if dataB64 == "" {
		return "", errors.New("qeval: empty response data")
	}
	decoded, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return "", fmt.Errorf("qeval base64 decode failed: %w", err)
	}
	matches := valoperAddrRex.FindAllStringSubmatch(string(decoded), -1)
	if len(matches) >= 2 && len(matches[1]) >= 2 {
		return matches[1][1], nil
	}
	return "", errors.New("consensus address not found in valopers response")
}

// GnoGetMoniker resolves a validator address to its moniker using the valopers realm.
// Returns "unknown" gracefully for unregistered validators.
func GnoGetMoniker(rpcURL, realmPath, address string) (string, error) {
	if realmPath == "" {
		realmPath = "gno.land/r/gnops/valopers"
	}
	expr := fmt.Sprintf(`%s.GetByAddr("%s")`, realmPath, address)
	dataHex := hex.EncodeToString([]byte(expr))
	url := fmt.Sprintf(`%s/abci_query?path=%%22vm/qeval%%22&data=0x%s`,
		strings.TrimRight(rpcURL, "/"), dataHex)
	body, err := gnoHTTPGet(url, 10*time.Second)
	if err != nil {
		return "unknown", nil // connection error is not critical for moniker
	}
	var result GnoABCIResult
	if err = json.Unmarshal(body, &result); err != nil {
		return "unknown", nil
	}
	if result.Error != nil {
		// Graceful: validator not registered in valopers realm
		return "unknown", nil
	}
	if result.Result == nil {
		return "unknown", nil
	}
	dataB64 := result.Result.Response.ResponseBase.Data
	if dataB64 == "" {
		logMsg := result.Result.Response.ResponseBase.Log
		if logMsg != "" && strings.Contains(logMsg, "valoper does not exist") {
			return "unknown", nil
		}
		return "unknown", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return "unknown", nil
	}
	matches := valoperMonikerRex.FindAllStringSubmatch(string(decoded), -1)
	if len(matches) > 0 && len(matches[0]) > 1 {
		return matches[0][1], nil
	}
	return "unknown", nil
}

// GnoGetValInfo populates ValInfo for a Gno.land chain using HTTP calls.
func (cc *ChainConfig) GnoGetValInfo(first bool) error {
	if cc.valInfo == nil {
		cc.valInfo = &ValInfo{}
	}

	rpcURL := cc.gnoRPCUrl()
	if rpcURL == "" {
		return errors.New("no RPC URL available")
	}

	realmPath := cc.GnoValopersRealm

	// 1. Resolve consensus address - check active set directly first
	if cc.gnoConsensusAddr == "" {
		directMatch, _ := GnoIsValidatorActive(rpcURL, cc.ValAddress)
		if directMatch {
			cc.gnoConsensusAddr = cc.ValAddress
			if first {
				l(fmt.Sprintf("⚙️ consensus address resolved from active set: %s", cc.ValAddress))
			}
		} else {
			// Try valopers realm as fallback
			consAddr, err := GnoResolveConsensusAddr(rpcURL, realmPath, cc.ValAddress)
			if err != nil {
				if first {
					l(fmt.Sprintf("⚙️ consensus resolution skipped: %s", err))
				}
			} else {
				cc.gnoConsensusAddr = consAddr
				if first {
					l(fmt.Sprintf("⚙️ resolved %s → consensus %s", cc.ValAddress, consAddr))
				}
			}
		}
	}

	// 2. Check if validator is in the active set
	lookupAddr := cc.gnoConsensusAddr
	if lookupAddr == "" {
		lookupAddr = cc.ValAddress
	}
	bonded, err := GnoIsValidatorActive(rpcURL, lookupAddr)
	if err != nil {
		if first {
			l(fmt.Sprintf("⚠️ could not check active set for %s: %s", lookupAddr, err))
		}
	} else {
		cc.valInfo.Bonded = bonded
	}

	// 3. Resolve moniker - graceful, never logs panic
	cc.valInfo.Moniker = cc.ValAddress[:minInt(len(cc.ValAddress), 20)] + ".."
	if first {
		moniker, _ := GnoGetMoniker(rpcURL, realmPath, cc.ValAddress)
		if moniker != "unknown" {
			cc.valInfo.Moniker = moniker
		} else {
			l(fmt.Sprintf("⚙️ moniker not available for %s (not in valopers realm)", cc.ValAddress))
		}
	}

	// 4. Gno.land has no slashing module
	cc.valInfo.Jailed = false
	cc.valInfo.Tombstoned = false

	// 5. Store consensus address for block signature matching
	if len(cc.valInfo.Conspub) == 0 && cc.gnoConsensusAddr != "" {
		vals, verr := GnoGetValidators(rpcURL)
		if verr == nil {
			addrLower := strings.ToLower(cc.gnoConsensusAddr)
			for _, v := range vals {
				if strings.ToLower(v.Address) == addrLower {
					if pkBytes, pkErr := base64.StdEncoding.DecodeString(v.PubKey.Value); pkErr == nil {
					cc.valInfo.Conspub = pkBytes
				} else {
					cc.valInfo.Conspub = []byte(strings.ToUpper(v.Address))
				}
					cc.valInfo.Valcons = v.Address
					break
				}
			}
		}
	}

	if first && cc.valInfo.Bonded {
		l(fmt.Sprintf("⚙️ found %s (%s) in active gno.land validator set", cc.ValAddress, cc.valInfo.Moniker))
	} else if first && !cc.valInfo.Bonded {
		l(fmt.Sprintf("❌ %s (%s) is NOT in active gno.land validator set", cc.ValAddress, cc.valInfo.Moniker))
	}

	return nil
}

// gnoRPCUrl returns the first available RPC URL.
func (cc *ChainConfig) gnoRPCUrl() string {
	for _, node := range cc.Nodes {
		if !node.down {
			return node.Url
		}
	}
	if len(cc.Nodes) > 0 {
		return cc.Nodes[0].Url
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// gnoAddresser is used for address/bech32 operations (present for compatibility).
