package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"

	"go.withmatt.com/inbox/internal/config"
	"go.withmatt.com/inbox/internal/log"
)

const (
	//nolint:gosec // OAuth client credentials are embedded intentionally.
	clientID = "812893543388-tml92q3ok88o0cgf7o9dinojselcqttn.apps.googleusercontent.com"
	//nolint:gosec // OAuth client credentials are embedded intentionally.
	clientSecret   = "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"
	callbackPath   = "/oauth2callback"
	keyringService = "go.withmatt.com/inbox"
)

func Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailModifyScope, gmail.MailGoogleComScope},
	}
}

func GetClient(ctx context.Context, email string) (*http.Client, error) {
	return getClient(ctx, Config(), email)
}

func GetClientQuiet(ctx context.Context, email string) (*http.Client, error) {
	return getClient(ctx, Config(), email)
}

func getClient(
	ctx context.Context,
	oauthCfg *oauth2.Config,
	email string,
) (*http.Client, error) {
	if strings.TrimSpace(email) == "" {
		return nil, errors.New("missing email for oauth")
	}

	tok, err := tokenFromKeyring(email)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			tok, err = tokenFromLegacyFile(email)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					log.Printf("Unable to read legacy token for %s: %v", email, err)
				}
				tok = nil
			} else if err := saveTokenToKeyring(email, tok); err != nil {
				log.Printf("Unable to cache oauth token in keyring: %v", err)
			} else if err := removeLegacyTokenFile(email); err != nil {
				log.Printf("Unable to remove legacy token for %s: %v", email, err)
			}
		} else {
			return nil, fmt.Errorf("unable to load oauth token from keyring: %w", err)
		}
	}

	if tok == nil {
		log.Printf("No token found for %s, starting authentication...", email)
		tok, err = getTokenFromWeb(oauthCfg, email)
		if err != nil {
			return nil, err
		}
		if err := saveTokenToKeyring(email, tok); err != nil {
			log.Printf("Unable to cache oauth token in keyring: %v", err)
		}
	}

	tokenSource := oauthCfg.TokenSource(ctx, tok)
	newTok, err := tokenSource.Token()
	if err != nil {
		log.Printf("Token refresh failed for %s, re-authenticating...", email)
		tok, err = getTokenFromWeb(oauthCfg, email)
		if err != nil {
			return nil, err
		}
		if err := saveTokenToKeyring(email, tok); err != nil {
			log.Printf("Unable to cache oauth token in keyring: %v", err)
		}
		tokenSource = oauthCfg.TokenSource(ctx, tok)
	} else if newTok.AccessToken != tok.AccessToken {
		if err := saveTokenToKeyring(email, newTok); err != nil {
			log.Printf("Unable to cache oauth token in keyring: %v", err)
		}
	}

	return oauth2.NewClient(ctx, tokenSource), nil
}

func getTokenFromWeb(config *oauth2.Config, email string) (*oauth2.Token, error) {
	if config == nil {
		return nil, errors.New("missing oauth config")
	}

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("unable to start oauth callback server: %w", err)
	}
	defer listener.Close()

	callbackURL := fmt.Sprintf("http://%s%s", listener.Addr().String(), callbackPath)
	cfg := *config
	cfg.RedirectURL = callbackURL

	state, err := randomState()
	if err != nil {
		return nil, err
	}

	pkceVerifier, pkceChallenge, err := generatePKCE()
	if err != nil {
		return nil, err
	}

	authURL := cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("login_hint", email),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge),
	)

	log.Printf("Authentication required for %s", email)
	if err := browser.OpenURL(authURL); err != nil {
		log.Printf("Open this URL to authorize: %v", authURL)
	} else {
		log.Printf("If your browser does not open, visit: %v", authURL)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter.", http.StatusBadRequest)
			select {
			case errCh <- errors.New("oauth state mismatch"):
			default:
			}
			return
		}
		if errText := r.URL.Query().Get("error"); errText != "" {
			http.Error(w, errText, http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("oauth error: %s", errText):
			default:
			}
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code parameter.", http.StatusBadRequest)
			select {
			case errCh <- errors.New("oauth callback missing code"):
			default:
			}
			return
		}
		_, _ = w.Write([]byte("inbox authentication complete. You can close this window."))
		select {
		case codeCh <- code:
		default:
		}
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	waitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	select {
	case code := <-codeCh:
		_ = server.Shutdown(context.Background())
		tok, err := cfg.Exchange(
			context.Background(),
			code,
			oauth2.SetAuthURLParam("code_verifier", pkceVerifier),
		)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve token: %w", err)
		}
		return tok, nil
	case err := <-errCh:
		_ = server.Shutdown(context.Background())
		return nil, err
	case <-waitCtx.Done():
		_ = server.Shutdown(context.Background())
		return nil, errors.New("timed out waiting for oauth callback")
	}
}

func generatePKCE() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("unable to generate PKCE verifier: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func tokenFromKeyring(email string) (*oauth2.Token, error) {
	value, err := keyring.Get(keyringService, keyringAccount(email))
	if err != nil {
		return nil, err
	}

	var tok oauth2.Token
	if err := json.Unmarshal([]byte(value), &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func saveTokenToKeyring(email string, token *oauth2.Token) error {
	if token == nil {
		return errors.New("missing oauth token")
	}
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	log.Printf("Saving credential to keyring for: %s", email)
	return keyring.Set(keyringService, keyringAccount(email), string(data))
}

func randomState() (string, error) {
	const size = 16
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("unable to generate oauth state: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func DeleteToken(email string) error {
	if strings.TrimSpace(email) == "" {
		return nil
	}
	if err := keyring.Delete(keyringService, keyringAccount(email)); err != nil &&
		!errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("unable to delete token from keyring: %w", err)
	}
	if err := removeLegacyTokenFile(email); err != nil {
		return fmt.Errorf("unable to remove legacy token file: %w", err)
	}
	return nil
}

func keyringAccount(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func tokenFromLegacyFile(email string) (*oauth2.Token, error) {
	tokenPath, err := config.TokenPath(email)
	if err != nil {
		return nil, err
	}
	return tokenFromFile(tokenPath)
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tok := &oauth2.Token{}
	if err := json.UnmarshalRead(f, tok); err != nil {
		return nil, err
	}
	return tok, nil
}

func removeLegacyTokenFile(email string) error {
	tokenPath, err := config.TokenPath(email)
	if err != nil {
		return err
	}
	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
