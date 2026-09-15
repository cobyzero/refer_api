package supabase

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

// ErrEmailNotConfirmed is returned when Supabase requires email verification
// (AUTOCONFIRM=false) and the account has not verified its OTP yet.
// Callers use errors.Is(err, ErrEmailNotConfirmed) to map it to 403/201
// with needsVerification instead of a generic 500/401.
var ErrEmailNotConfirmed = errors.New("email not confirmed")

// VerifyTypeSignup is the GoTrue verification type for the email OTP sent
// at signup. Kept as a constant so handlers/services never hardcode it.
const VerifyTypeSignup = "signup"

// ErrEmailServiceUnavailable is returned when GoTrue could not send the
// email (bad/missing SMTP credentials, mailer outage, rate limit on send).
// Callers map it to 503 so Flutter shows "inténtalo más tarde" instead of a
// generic 500. It is distinct from ErrEmailNotConfirmed (account pending
// verification), which maps to 201/403.
var ErrEmailServiceUnavailable = errors.New("email service unavailable")

// IsMailerErr reports whether err means the email could not be sent.
// Matches GoTrue mailer failures like "Error sending confirmation email"
// (e.g. SMTP 535 "Missing password" when SMTP_PASS is empty) as well as
// recovery/resend send failures. It intentionally does NOT match
// "email not confirmed/verified", which is handled by IsNotConfirmedErr.
func IsMailerErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrEmailServiceUnavailable) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "error sending") ||
		strings.Contains(msg, "email service unavailable") ||
		strings.Contains(msg, "smtp") ||
		strings.Contains(msg, "mailer") ||
		strings.Contains(msg, "535")
}

// asMailerError wraps err with ErrEmailServiceUnavailable when its message
// looks like a mailer/send failure, so callers can use errors.Is.
func asMailerError(err error) error {
	if err == nil {
		return nil
	}
	if IsMailerErr(err) {
		return fmt.Errorf("%w: %s", ErrEmailServiceUnavailable, err.Error())
	}
	return err
}

// IsNotConfirmedErr reports whether err (usually from parseError) means the
// account exists but its email is still unverified.
// NOTE: it intentionally matches only the "email not confirmed/verified"
// phrases. Broader substrings like "confirmation" would also match mailer
// failures such as "Error sending confirmation email" (e.g. bad SMTP
// credentials), which must surface as real errors instead.
func IsNotConfirmedErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrEmailNotConfirmed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "email not confirmed") ||
		strings.Contains(msg, "email not verified")
}

// Session is the Supabase Auth session returned to the caller.
// Go adapts it to the legacy Flutter contract {accessToken, refreshToken}.
type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	UserID       string `json:"-"`
	Email        string `json:"-"`
}

// supabaseUser mirrors the subset of auth.users we need.
type supabaseUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// tokenResponse mirrors /auth/v1/token and /auth/v1/signup responses.
type tokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	User         supabaseUser `json:"user"`
}

// Client is a minimal Supabase Auth REST client (GoTrue compatible).
// It uses only the standard library so no new dependencies are required.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient builds a Supabase Auth client from cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(method, path string, query url.Values, body interface{}, useServiceRole bool, accessToken string) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}
	u := c.cfg.URL + path
	if query != nil && len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if useServiceRole {
		req.Header.Set("apikey", c.cfg.ServiceRoleKey)
		req.Header.Set("Authorization", "Bearer "+c.cfg.ServiceRoleKey)
	} else {
		req.Header.Set("apikey", c.cfg.AnonKey)
		if accessToken != "" {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}
	}
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

func parseError(status int, data []byte) error {
	var payload struct {
		Msg   string `json:"msg"`
		Error string `json:"error"`
		// GoTrue uses {msg} or {error_description}
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
	}
	_ = json.Unmarshal(data, &payload)
	msg := payload.Msg
	if msg == "" {
		msg = payload.ErrorDescription
	}
	if msg == "" {
		msg = payload.Message
	}
	if msg == "" {
		msg = payload.Error
	}
	if msg == "" {
		msg = string(data)
		if msg == "" {
			msg = "supabase auth request failed"
		}
	}
	return fmt.Errorf("supabase auth (status %d): %s", status, msg)
}

// SignUp registers a user in Supabase Auth (auth.users).
// When email confirmation is ON (AUTOCONFIRM=false), Supabase returns a user
// with no session and the follow-up SignIn fails with "Email not confirmed".
// In that case SignUp returns Session{UserID, Email} + ErrEmailNotConfirmed
// (via errors.Is) so the caller can create the public.users mirror row and
// answer 201 {needsVerification:true} instead of a generic error.
func (c *Client) SignUp(email, password, name string) (Session, error) {
	status, data, err := c.do(http.MethodPost, "/auth/v1/signup", nil, map[string]interface{}{
		"email":    email,
		"password": password,
		"data":     map[string]string{"name": name},
	}, false, "")
	if err != nil {
		return Session{}, err
	}
	if status < 200 || status >= 300 {
		return Session{}, asMailerError(parseError(status, data))
	}
	var tr tokenResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return Session{}, err
	}
	pending := Session{UserID: tr.User.ID, Email: tr.User.Email}
	if pending.Email == "" {
		pending.Email = email
	}
	// When email confirmation is ON, Supabase returns a user with no session.
	// Fall back to an immediate password grant so Go can still return
	// {accessToken, refreshToken} and keep the Flutter contract unchanged.
	// When the account needs verification, SignIn fails with "Email not
	// confirmed" -> surface ErrEmailNotConfirmed (keeping UserID/Email).
	if tr.AccessToken == "" {
		if tr.User.ID == "" {
			// Confirmation-required responses come in two shapes depending on
			// the GoTrue version: {user:{id,email}} or the bare user
			// {id,email,...}. Accept both.
			var wrapped struct {
				User supabaseUser `json:"user"`
			}
			if err := json.Unmarshal(data, &wrapped); err == nil && wrapped.User.ID != "" {
				pending.UserID = wrapped.User.ID
				if wrapped.User.Email != "" {
					pending.Email = wrapped.User.Email
				}
			} else {
				var bare supabaseUser
				if err := json.Unmarshal(data, &bare); err == nil && bare.ID != "" {
					pending.UserID = bare.ID
					if bare.Email != "" {
						pending.Email = bare.Email
					}
				}
			}
			if pending.UserID == "" {
				return Session{}, fmt.Errorf("supabase signup requires email confirmation; session is empty")
			}
		}
		if sess, signInErr := c.SignIn(email, password); signInErr == nil {
			return sess, nil
		} else if IsNotConfirmedErr(signInErr) {
			return pending, ErrEmailNotConfirmed
		} else {
			return Session{}, signInErr
		}
	}
	return Session{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		TokenType:    tr.TokenType,
		ExpiresIn:    tr.ExpiresIn,
		UserID:       tr.User.ID,
		Email:        tr.User.Email,
	}, nil
}

// SignIn validates credentials against Supabase Auth and returns a session.
func (c *Client) SignIn(email, password string) (Session, error) {
	q := url.Values{}
	q.Set("grant_type", "password")
	status, data, err := c.do(http.MethodPost, "/auth/v1/token", q, map[string]string{
		"email":    email,
		"password": password,
	}, false, "")
	if err != nil {
		return Session{}, err
	}
	if status < 200 || status >= 300 {
		return Session{}, parseError(status, data)
	}
	var tr tokenResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return Session{}, err
	}
	return Session{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		TokenType:    tr.TokenType,
		ExpiresIn:    tr.ExpiresIn,
		UserID:       tr.User.ID,
		Email:        tr.User.Email,
	}, nil
}

// Refresh exchanges a Supabase refresh token for a new session.
func (c *Client) Refresh(refreshToken string) (Session, error) {
	q := url.Values{}
	q.Set("grant_type", "refresh_token")
	status, data, err := c.do(http.MethodPost, "/auth/v1/token", q, map[string]string{
		"refresh_token": refreshToken,
	}, false, "")
	if err != nil {
		return Session{}, err
	}
	if status < 200 || status >= 300 {
		return Session{}, parseError(status, data)
	}
	var tr tokenResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return Session{}, err
	}
	return Session{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		TokenType:    tr.TokenType,
		ExpiresIn:    tr.ExpiresIn,
		UserID:       tr.User.ID,
		Email:        tr.User.Email,
	}, nil
}

// Logout revokes the Supabase session identified by accessToken.
func (c *Client) Logout(accessToken string) error {
	status, data, err := c.do(http.MethodPost, "/auth/v1/logout", nil, nil, false, accessToken)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return parseError(status, data)
	}
	return nil
}

// Recover triggers the Supabase password-recovery email (link based).
// The Flutter {email} -> {message} contract is preserved by Go.
func (c *Client) Recover(email string) error {
	status, data, err := c.do(http.MethodPost, "/auth/v1/recover", nil, map[string]string{
		"email": email,
	}, false, "")
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return asMailerError(parseError(status, data))
	}
	return nil
}

// VerifyOtp confirms an email OTP (6-digit code from the signup email).
// On success GoTrue returns a full session {access_token, refresh_token}.
// type is usually VerifyTypeSignup ("signup") for account verification.
func (c *Client) VerifyOtp(email, token, verifyType string) (Session, error) {
	if verifyType == "" {
		verifyType = VerifyTypeSignup
	}
	status, data, err := c.do(http.MethodPost, "/auth/v1/verify", nil, map[string]string{
		"email": email,
		"token": token,
		"type":  verifyType,
	}, false, "")
	if err != nil {
		return Session{}, err
	}
	if status < 200 || status >= 300 {
		return Session{}, parseError(status, data)
	}
	var tr tokenResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return Session{}, err
	}
	if tr.AccessToken == "" {
		return Session{}, fmt.Errorf("supabase verify did not return a session")
	}
	return Session{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		TokenType:    tr.TokenType,
		ExpiresIn:    tr.ExpiresIn,
		UserID:       tr.User.ID,
		Email:        tr.User.Email,
	}, nil
}

// ResendSignup re-sends the signup confirmation OTP email (rate-limited by
// GoTrue). Used by POST /api/auth/resend-verification when the user did not
// receive the code or it expired.
func (c *Client) ResendSignup(email string) error {
	status, data, err := c.do(http.MethodPost, "/auth/v1/resend", nil, map[string]string{
		"type":  VerifyTypeSignup,
		"email": email,
	}, false, "")
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return asMailerError(parseError(status, data))
	}
	return nil
}
// AdminConfirmEmail marks auth.users.id as email-confirmed (service role).
// Used when the operator disables signup emails (smtp_enabled=false o
// signup_email_enabled=false): el registro no debe exigir OTP y la cuenta
// queda activa de inmediato sin depender del SMTP.
func (c *Client) AdminConfirmEmail(authUserID string) error {
	if !c.cfg.HasAdminKey() {
		return fmt.Errorf("supabase service role key is not configured")
	}
	status, data, err := c.do(http.MethodPut, "/auth/v1/admin/users/"+authUserID, nil, map[string]interface{}{
		"email_confirm": true,
	}, true, "")
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return parseError(status, data)
	}
	return nil
}

// AdminUpdatePassword sets a new password for auth.users.id via service role.
func (c *Client) AdminUpdatePassword(authUserID, newPassword string) error {
	if !c.cfg.HasAdminKey() {
		return fmt.Errorf("supabase service role key is not configured")
	}
	status, data, err := c.do(http.MethodPut, "/auth/v1/admin/users/"+authUserID, nil, map[string]string{
		"password": newPassword,
	}, true, "")
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return parseError(status, data)
	}
	return nil
}

// AdminDeleteUser removes a Supabase Auth user (used to avoid orphan auth.users
// when the public.users insert fails after a successful signup).
func (c *Client) AdminDeleteUser(authUserID string) error {
	if !c.cfg.HasAdminKey() {
		return fmt.Errorf("supabase service role key is not configured")
	}
	status, data, err := c.do(http.MethodDelete, "/auth/v1/admin/users/"+authUserID, nil, nil, true, "")
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return parseError(status, data)
	}
	return nil
}

// AdminGetUserByEmail looks up an auth user by email (service role).
// Used during migration to map legacy users to auth.users.id.
func (c *Client) AdminGetUserByEmail(email string) (string, error) {
	users, err := c.AdminListUsers()
	if err != nil {
		return "", err
	}
	for _, u := range users {
		if u.Email == email {
			return u.ID, nil
		}
	}
	return "", fmt.Errorf("supabase user not found for email %s", email)
}

// AuthUser is the exported subset of auth.users used by admin tooling.
type AuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// AdminListUsers returns all auth users (service role), following pagination.
// Caps at 100 pages as a safety net; document if you outgrow it.
func (c *Client) AdminListUsers() ([]AuthUser, error) {
	if !c.cfg.HasAdminKey() {
		return nil, fmt.Errorf("supabase service role key is not configured")
	}
	var all []AuthUser
	for page := 1; page <= 100; page++ {
		q := url.Values{}
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("per_page", "100")
		status, data, err := c.do(http.MethodGet, "/auth/v1/admin/users", q, nil, true, "")
		if err != nil {
			return nil, err
		}
		if status < 200 || status >= 300 {
			return nil, parseError(status, data)
		}
		// GoTrue returns {"users": [...]}; tolerate a bare array for safety.
		var obj struct {
			Users []AuthUser `json:"users"`
		}
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, err
		}
		if obj.Users == nil {
			var arr []AuthUser
			if err := json.Unmarshal(data, &arr); err != nil {
				return nil, fmt.Errorf("unexpected admin users shape: %s", string(data))
			}
			obj.Users = arr
		}
		if len(obj.Users) == 0 {
			break
		}
		all = append(all, obj.Users...)
		if len(obj.Users) < 100 {
			break
		}
	}
	return all, nil
}

// AdminCreateUser creates an auth.users row (service role) and returns its id.
// userID is attempted as the new id so public.users FKs keep working without
// remap; when the server ignores it, the caller must remap (see migrator).
// Passwords CANNOT be migrated (legacy bcrypt vs GoTrue) — pass a random one
// and send a recovery email so the user sets their own.
func (c *Client) AdminCreateUser(email, password, name, userID string, emailConfirm bool) (string, error) {
	if !c.cfg.HasAdminKey() {
		return "", fmt.Errorf("supabase service role key is not configured")
	}
	body := map[string]interface{}{
		"email":         email,
		"password":      password,
		"email_confirm": emailConfirm,
		"user_metadata": map[string]string{"name": name},
	}
	if userID != "" {
		body["id"] = userID
	}
	status, data, err := c.do(http.MethodPost, "/auth/v1/admin/users", nil, body, true, "")
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", parseError(status, data)
	}
	var created struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		// Some versions nest under user.
		User AuthUser `json:"user"`
	}
	if err := json.Unmarshal(data, &created); err != nil {
		return "", err
	}
	if created.ID == "" {
		created.ID = created.User.ID
	}
	if created.ID == "" {
		return "", fmt.Errorf("admin create did not return a user id: %s", string(data))
	}
	return created.ID, nil
}
