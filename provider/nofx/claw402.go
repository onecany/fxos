package nofx

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"fxos/httpclient"
	"fxos/mcp"
	"fxos/mcp/payment"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// Claw402DataClient wraps fxos API calls through claw402's x402 payment gateway.
// Instead of calling fxosos.ai directly, it calls claw402.ai/api/v1/fxos/...
// and pays with USDC for each request.
type Claw402DataClient struct {
	claw402URL string
	privateKey *ecdsa.PrivateKey
	httpClient *http.Client
	logger     mcp.Logger
}

// NewClaw402DataClient creates a client that routes fxos requests through claw402.
// privateKeyHex is the wallet private key (0x-prefixed hex string).
func NewClaw402DataClient(claw402URL, privateKeyHex string, logger mcp.Logger) (*Claw402DataClient, error) {
	if claw402URL == "" {
		claw402URL = "https://claw402.ai"
	}
	claw402URL = strings.TrimRight(claw402URL, "/")

	if privateKeyHex == "" {
		privateKeyHex = os.Getenv("CLAW402_WALLET_KEY")
	}
	if privateKeyHex == "" {
		return nil, fmt.Errorf("claw402 wallet private key not set")
	}

	hexKey := strings.TrimPrefix(privateKeyHex, "0x")
	pk, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid claw402 private key: %w", err)
	}

	return &Claw402DataClient{
		claw402URL: claw402URL,
		privateKey: pk,
		httpClient: httpclient.New(30 * time.Second),
		logger:     logger,
	}, nil
}

// endpoint mapping: fxos path → claw402 path
var endpointMap = map[string]string{
	"/api/ai500/list":  "/api/v1/fxos/ai500/list",
	"/api/ai500/stats": "/api/v1/fxos/ai500/stats",
}

// mapEndpoint converts a fxos endpoint to a claw402 endpoint.
// For endpoints not in the static map, applies the general pattern:
// /api/xxx → /api/v1/fxos/xxx
func mapEndpoint(fxosPath string) string {
	if mapped, ok := endpointMap[fxosPath]; ok {
		return mapped
	}
	// General pattern: /api/xxx → /api/v1/fxos/xxx
	if strings.HasPrefix(fxosPath, "/api/") {
		return "/api/v1/fxos/" + strings.TrimPrefix(fxosPath, "/api/")
	}
	return fxosPath
}

// DoRequest makes a GET request through claw402 with x402 payment.
func (c *Claw402DataClient) DoRequest(endpoint string) ([]byte, error) {
	claw402Path := mapEndpoint(endpoint)
	// Strip auth= query params (claw402 uses x402 payment, not auth keys)
	if idx := strings.Index(claw402Path, "?auth="); idx != -1 {
		claw402Path = claw402Path[:idx]
	}
	if idx := strings.Index(claw402Path, "&auth="); idx != -1 {
		claw402Path = claw402Path[:idx]
	}

	fullURL := c.claw402URL + claw402Path

	buildReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Client-ID", "fxos")
		return req, nil
	}

	signFn := payment.MakeClaw402SignFunc(c.privateKey)

	body, err := payment.DoX402Request(
		context.Background(),
		c.httpClient,
		buildReq,
		signFn,
		"claw402-data",
		c.logger,
	)
	if err != nil {
		return nil, fmt.Errorf("claw402 data request failed (%s): %w", claw402Path, err)
	}

	return body, nil
}
