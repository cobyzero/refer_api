package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrInvalidCredentials is returned when Kratos rejects an identifier+password.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrAlreadyExists is returned when an account with the identifier exists.
var ErrAlreadyExists = errors.New("email already exists")

// ErrInvalidCode is returned for wrong/expired email OTP codes.
var ErrInvalidCode = errors.New("invalid or expired verification code")

// ErrIdentityNotFound is returned when no Kratos identity matches.
var ErrIdentityNotFound = errors.New("identity not found")

// ErrEmailServiceUnavailable is returned when Kratos could not send the
// email (bad/missing SMTP credentials, courier outage). Callers map it to
// 503 so Flutter shows "inténtalo más tarde" instead of a generic 500.
var ErrEmailServiceUnavailable = errors.New("email service unavailable")

// IsMailerErr reports whether err looks like a mailer/send failure.
func IsMailerErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrEmailServiceUnavailable) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "courier") ||
		strings.Contains(msg, "smtp") ||
		strings.Contains(msg, "mailer") ||
		strings.Contains(msg, "email service unavailable") ||
		strings.Contains(msg, "535") ||
		strings.Contains(msg, "error sending")
}

// Identity is the subset of a Kratos identity the backend needs.
type Identity struct {
	ID       string
	Email    string
	Name     string
	Verified bool
}

// Client is a minimal Ory Kratos HTTP client (public self-service flows +
// admin API) using only the standard library.
type Client struct {
	publicURL  string
	adminURL   string
	httpClient *http.Client
}

// NewClient builds a Kratos client from cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		publicURL:  cfg.PublicURL,
		adminURL:   cfg.AdminURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) do(base, method, path string, query url.Values, body interface{}) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}
	u := base + path
	if query != nil && len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, data, nil
}

func (c *Client) pub(method, path string, query url.Values, body interface{}) (int, []byte, error) {
	return c.do(c.publicURL, method, path, query, body)
}

func (c *Client) adm(method, path string, query url.Values, body interface{}) (int, []byte, error) {
	return c.do(c.adminURL, method, path, query, body)
}

// kratosErrorMessage extracts a human message from Kratos error payloads:
// {"error":{"message"|"reason"}}, {"ui":{"messages":[{"text"}]}} or flow
// re-renders {"ui":{"nodes":[{"messages":[{"text"}]}]}} (e.g. weak password,
// duplicate identifier). Falls back to the raw body.
func kratosErrorMessage(status int, data []byte) string {
	var payload struct {
		Err struct {
			Message string `json:"message"`
			Reason  string `json:"reason"`
		} `json:"error"`
		Message string `json:"message"`
		UI      struct {
			Messages []struct {
				Text string `json:"text"`
			} `json:"messages"`
			Nodes []struct {
				Messages []struct {
					Text string `json:"text"`
					Type string `json:"type"`
				} `json:"messages"`
			} `json:"nodes"`
		} `json:"ui"`
	}
	msg := ""
	if err := json.Unmarshal(data, &payload); err == nil {
		msg = payload.Err.Message
		if msg == "" {
			msg = payload.Err.Reason
		}
		if msg == "" && len(payload.UI.Messages) > 0 {
			msg = payload.UI.Messages[0].Text
		}
		if msg == "" {
			var parts []string
			for _, n := range payload.UI.Nodes {
				for _, m := range n.Messages {
					if m.Type == "error" && m.Text != "" {
						parts = append(parts, m.Text)
					}
				}
			}
			msg = strings.Join(parts, "; ")
		}
		if msg == "" {
			msg = payload.Message
		}
	}
	if msg == "" {
		msg = strings.TrimSpace(string(data))
		if msg == "" {
			msg = "kratos request failed"
		}
	}
	return fmt.Sprintf("kratos (status %d): %s", status, msg)
}

// flowID extracts {"id": "..."} from a flow-creation response.
func flowID(data []byte) (string, error) {
	var f struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &f); err != nil || f.ID == "" {
		return "", fmt.Errorf("kratos flow without id: %s", string(data))
	}
	return f.ID, nil
}

// CreateFlow starts a self-service API flow: registration, login,
// verification or recovery. Returns the flow id for ?flow= submits.
func (c *Client) CreateFlow(kind string) (string, error) {
	status, data, err := c.pub(http.MethodGet, "/self-service/"+kind+"/api", nil, nil)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", errors.New(kratosErrorMessage(status, data))
	}
	return flowID(data)
}

// identityFromPayload scans login/registration/admin shapes for the identity:
// {"identity":{...}}, {"session":{"identity":{...}}} or the identity itself.
func identityFromPayload(data []byte) (*Identity, error) {
	var top struct {
		Identity  *rawIdentity `json:"identity"`
		Session   *struct {
			Identity *rawIdentity `json:"identity"`
		} `json:"session"`
		ID     string `json:"id"`
		Traits *struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"traits"`
	}
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("kratos identity parse: %w", err)
	}
	var raw *rawIdentity
	switch {
	case top.Identity != nil && top.Identity.ID != "":
		raw = top.Identity
	case top.Session != nil && top.Session.Identity != nil && top.Session.Identity.ID != "":
		raw = top.Session.Identity
	case top.ID != "" && top.Traits != nil:
		raw = &rawIdentity{ID: top.ID, Traits: struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}{Email: top.Traits.Email, Name: top.Traits.Name}}
	default:
		return nil, fmt.Errorf("kratos response without identity: %s", string(data))
	}
	return raw.toIdentity(), nil
}

type rawIdentity struct {
	ID     string `json:"id"`
	Traits struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"traits"`
	VerifiableAddresses []struct {
		Value    string `json:"value"`
		Verified bool   `json:"verified"`
		Via      string `json:"via"`
	} `json:"verifiable_addresses"`
}

func (r *rawIdentity) toIdentity() *Identity {
	id := &Identity{ID: r.ID, Email: r.Traits.Email, Name: r.Traits.Name}
	for _, a := range r.VerifiableAddresses {
		if strings.EqualFold(a.Value, r.Traits.Email) || a.Via == "email" {
			if a.Verified {
				id.Verified = true
			}
		}
	}
	if id.Email == "" {
		for _, a := range r.VerifiableAddresses {
			if a.Via == "email" && a.Value != "" {
				id.Email = a.Value
				break
			}
		}
	}
	return id
}

// Register creates the identity with password and triggers the verification
// email (code). On success returns the identity (usually unverified yet).
func (c *Client) Register(email, password, name string) (*Identity, error) {
	flow, err := c.CreateFlow("registration")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("flow", flow)
	status, data, err := c.pub(http.MethodPost, "/self-service/registration", q, map[string]interface{}{
		"method":   "password",
		"password": password,
		"traits":   map[string]string{"email": email, "name": name},
	})
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		msg := kratosErrorMessage(status, data)
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "already") || strings.Contains(lower, "exists") || strings.Contains(lower, "duplicate") || strings.Contains(lower, "identifier") {
			return nil, fmt.Errorf("%w: %s", ErrAlreadyExists, msg)
		}
		if IsMailerErr(errors.New(msg)) || status >= 500 {
			return nil, fmt.Errorf("%w: %s", ErrEmailServiceUnavailable, msg)
		}
		return nil, errors.New(msg)
	}
	id, err := identityFromPayload(data)
	if err != nil {
		// Registration accepted but shape unknown: resolve via admin.
		adminID, aErr := c.AdminGetUserByEmail(email)
		if aErr != nil {
			return nil, err
		}
		return c.AdminGetIdentity(adminID)
	}
	return id, nil
}

// Login checks identifier+password. Returns the identity (verified flag from
// the identity record, refreshed via admin for a single source of truth).
func (c *Client) Login(email, password string) (*Identity, error) {
	flow, err := c.CreateFlow("login")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("flow", flow)
	status, data, err := c.pub(http.MethodPost, "/self-service/login", q, map[string]interface{}{
		"method":     "password",
		"identifier": email,
		"password":   password,
	})
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		msg := kratosErrorMessage(status, data)
		if status == 400 || status == 401 || status == 403 || status == 422 {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCredentials, msg)
		}
		return nil, errors.New(msg)
	}
	id, err := identityFromPayload(data)
	if err != nil {
		adminID, aErr := c.AdminGetUserByEmail(email)
		if aErr != nil {
			return nil, err
		}
		return c.AdminGetIdentity(adminID)
	}
	// Refresh verified flag from the record (single source of truth).
	if fresh, err := c.AdminGetIdentity(id.ID); err == nil {
		return fresh, nil
	}
	return id, nil
}

// RequestVerificationCode sends a fresh 6-digit verification email.
func (c *Client) RequestVerificationCode(email string) error {
	flow, err := c.CreateFlow("verification")
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Set("flow", flow)
	status, data, err := c.pub(http.MethodPost, "/self-service/verification", q, map[string]string{
		"email": email,
	})
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		msg := kratosErrorMessage(status, data)
		if IsMailerErr(errors.New(msg)) || status >= 500 {
			return fmt.Errorf("%w: %s", ErrEmailServiceUnavailable, msg)
		}
		return errors.New(msg)
	}
	return nil
}

// SubmitVerificationCode confirms the email with the 6-digit code and
// returns the verified identity.
func (c *Client) SubmitVerificationCode(email, code string) (*Identity, error) {
	flow, err := c.CreateFlow("verification")
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("flow", flow)
	status, data, err := c.pub(http.MethodPost, "/self-service/verification", q, map[string]string{
		"email":  email,
		"code":   code,
		"method": "code",
	})
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		msg := kratosErrorMessage(status, data)
		if status == 400 || status == 401 || status == 403 || status == 410 || status == 422 {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCode, msg)
		}
		if IsMailerErr(errors.New(msg)) || status >= 500 {
			return nil, fmt.Errorf("%w: %s", ErrEmailServiceUnavailable, msg)
		}
		return nil, errors.New(msg)
	}
	adminID, err := c.AdminGetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	id, err := c.AdminGetIdentity(adminID)
	if err != nil {
		return nil, err
	}
	if !id.Verified {
		return nil, fmt.Errorf("%w: address still unverified", ErrInvalidCode)
	}
	return id, nil
}

// AdminGetUserByEmail resolves the identity id for an email (admin API).
func (c *Client) AdminGetUserByEmail(email string) (string, error) {
	q := url.Values{}
	q.Set("credentials_identifier", email)
	q.Set("per_page", "1")
	status, data, err := c.adm(http.MethodGet, "/admin/identities", q, nil)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", errors.New(kratosErrorMessage(status, data))
	}
	var list []rawIdentity
	if err := json.Unmarshal(data, &list); err != nil || len(list) == 0 || list[0].ID == "" {
		return "", fmt.Errorf("%w for email %s", ErrIdentityNotFound, email)
	}
	return list[0].ID, nil
}

// AdminGetIdentity returns the full identity record by id.
func (c *Client) AdminGetIdentity(id string) (*Identity, error) {
	status, data, err := c.adm(http.MethodGet, "/admin/identities/"+id, nil, nil)
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, fmt.Errorf("%w: %s", ErrIdentityNotFound, id)
	}
	if status < 200 || status >= 300 {
		return nil, errors.New(kratosErrorMessage(status, data))
	}
	var raw rawIdentity
	if err := json.Unmarshal(data, &raw); err != nil || raw.ID == "" {
		if parsed, perr := identityFromPayload(data); perr == nil {
			return parsed, nil
		}
		return nil, fmt.Errorf("kratos identity parse: %s", string(data))
	}
	return raw.toIdentity(), nil
}

// AdminCreateUser creates an identity (admin API). When verified=true the
// email is marked verified without sending mail (operator-provisioned path).
func (c *Client) AdminCreateUser(email, password, name string, verified bool) (string, error) {
	addrs := []map[string]interface{}{}
	if verified {
		addrs = append(addrs, map[string]interface{}{
			"value": email, "verified": true, "via": "email", "status": "completed",
		})
	}
	body := map[string]interface{}{
		"schema_id": "default",
		"traits":    map[string]string{"email": email, "name": name},
		"credentials": map[string]interface{}{
			"password": map[string]interface{}{
				"config": map[string]string{"password": password},
			},
		},
	}
	if len(addrs) > 0 {
		body["verifiable_addresses"] = addrs
	}
	status, data, err := c.adm(http.MethodPost, "/admin/identities", nil, body)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		msg := kratosErrorMessage(status, data)
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "already") || strings.Contains(lower, "exists") || strings.Contains(lower, "duplicate") {
			return "", fmt.Errorf("%w: %s", ErrAlreadyExists, msg)
		}
		return "", errors.New(msg)
	}
	id, err := identityFromPayload(data)
	if err != nil {
		adminID, aErr := c.AdminGetUserByEmail(email)
		if aErr != nil {
			return "", err
		}
		return adminID, nil
	}
	return id.ID, nil
}

// AdminUpdatePassword sets a new password (admin API, hashed server-side).
func (c *Client) AdminUpdatePassword(identityID, newPassword string) error {
	status, data, err := c.adm(http.MethodGet, "/admin/identities/"+identityID, nil, nil)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return errors.New(kratosErrorMessage(status, data))
	}
	var current map[string]interface{}
	if err := json.Unmarshal(data, &current); err != nil {
		return fmt.Errorf("kratos identity parse: %w", err)
	}
	creds, _ := current["credentials"].(map[string]interface{})
	if creds == nil {
		creds = map[string]interface{}{}
	}
	creds["password"] = map[string]interface{}{
		"config": map[string]string{"password": newPassword},
	}
	current["credentials"] = creds
	status, data, err = c.adm(http.MethodPut, "/admin/identities/"+identityID, nil, current)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return errors.New(kratosErrorMessage(status, data))
	}
	return nil
}

// AdminDeleteUser removes an identity (used to avoid orphans when the
// public.users mirror insert fails after identity creation).
func (c *Client) AdminDeleteUser(identityID string) error {
	status, data, err := c.adm(http.MethodDelete, "/admin/identities/"+identityID, nil, nil)
	if err != nil {
		return err
	}
	if status == 404 {
		return nil
	}
	if status < 200 || status >= 300 {
		return errors.New(kratosErrorMessage(status, data))
	}
	return nil
}

// EnsureUser returns the identity id for email, creating a verified identity
// when missing. Used for operator-provisioned accounts (seed admin) so they
// can sign in immediately without an OTP round-trip.
func (c *Client) EnsureUser(email, password, name string) (string, error) {
	if id, err := c.AdminGetUserByEmail(email); err == nil && id != "" {
		return id, nil
	}
	return c.AdminCreateUser(email, password, name, true)
}

// WaitReady polls the admin health endpoint until Kratos serves (migrations
// applied by kratos-migrate) or the timeout elapses.
func (c *Client) WaitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, data, err := c.adm(http.MethodGet, "/health/ready", nil, nil)
		if err == nil && status == 200 && strings.Contains(string(data), "ok") {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("kratos not ready at %s: %w", c.adminURL, err)
			}
			return fmt.Errorf("kratos not ready at %s (status %d): %s", c.adminURL, status, string(data))
		}
		time.Sleep(2 * time.Second)
	}
}
