package paystack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.paystack.co"

type Client struct {
	secretKey   string
	callbackURL string
	httpClient  *http.Client
}

func NewClient(secretKey, callbackURL string) *Client {
	return &Client{
		secretKey:   secretKey,
		callbackURL: strings.TrimSpace(callbackURL),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type apiResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type InitializeResult struct {
	AccessCode       string
	AuthorizationURL string
	Reference        string
}

func (c *Client) InitializeTransaction(ctx context.Context, email string, amountKobo int64, reference string) (InitializeResult, error) {
	body := map[string]any{
		"email":    email,
		"amount":   amountKobo,
		"currency": "NGN",
	}
	if reference != "" {
		body["reference"] = reference
	}
	if c.callbackURL != "" {
		body["callback_url"] = c.callbackURL
	}

	var data struct {
		AccessCode       string `json:"access_code"`
		AuthorizationURL string `json:"authorization_url"`
		Reference        string `json:"reference"`
	}
	if err := c.post(ctx, "/transaction/initialize", body, &data); err != nil {
		return InitializeResult{}, err
	}
	authURL := strings.TrimSpace(data.AuthorizationURL)
	if authURL == "" && data.AccessCode != "" {
		authURL = "https://checkout.paystack.com/" + data.AccessCode
	}
	return InitializeResult{
		AccessCode:       data.AccessCode,
		AuthorizationURL: authURL,
		Reference:        data.Reference,
	}, nil
}

type VerifyResult struct {
	Status string
	Amount int64
}

func (c *Client) VerifyTransaction(ctx context.Context, reference string) (VerifyResult, error) {
	var data struct {
		Status string `json:"status"`
		Amount int64  `json:"amount"`
	}
	if err := c.get(ctx, "/transaction/verify/"+reference, &data); err != nil {
		return VerifyResult{}, err
	}
	return VerifyResult{Status: data.Status, Amount: data.Amount}, nil
}

func (c *Client) post(ctx context.Context, path string, payload any, out any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal paystack body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("paystack request: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read paystack response: %w", err)
	}

	var envelope apiResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode paystack envelope: %w", err)
	}
	if !envelope.Status {
		return fmt.Errorf("paystack error: %s", envelope.Message)
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return fmt.Errorf("decode paystack data: %w", err)
		}
	}
	return nil
}
