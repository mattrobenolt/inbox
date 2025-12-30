package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmailapi "google.golang.org/api/gmail/v1"

	"go.withmatt.com/inbox/internal/config"
)

const (
	//nolint:gosec // OAuth client credentials are embedded intentionally.
	clientID = "812893543388-tml92q3ok88o0cgf7o9dinojselcqttn.apps.googleusercontent.com"
	//nolint:gosec // OAuth client credentials are embedded intentionally.
	clientSecret = "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"
	callbackPath = "/oauth2callback"
)

func Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmailapi.GmailModifyScope},
	}
}

func GetClient(ctx context.Context, tokenPath string, email string) (*http.Client, error) {
	return getClient(ctx, Config(), tokenPath, email, printfLogger)
}

func GetClientQuiet(ctx context.Context, tokenPath string, email string) (*http.Client, error) {
	return getClient(ctx, Config(), tokenPath, email, nil)
}

type loggerFunc func(format string, args ...any)

func printfLogger(format string, args ...any) {
	fmt.Printf(format, args...)
}

func getClient(
	ctx context.Context,
	oauthCfg *oauth2.Config,
	tokenPath string,
	email string,
	logf loggerFunc,
) (*http.Client, error) {
	if err := config.EnsureTokensDir(); err != nil {
		return nil, fmt.Errorf("unable to create tokens directory: %w", err)
	}

	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		if logf != nil {
			logf("No token found for %s, starting authentication...\n", email)
		}
		tok, err = getTokenFromWeb(oauthCfg, email, logf)
		if err != nil {
			return nil, err
		}
		saveToken(tokenPath, tok, logf)
	}

	tokenSource := oauthCfg.TokenSource(ctx, tok)
	newTok, err := tokenSource.Token()
	if err != nil {
		if logf != nil {
			logf("Token refresh failed for %s, re-authenticating...\n", email)
		}
		tok, err = getTokenFromWeb(oauthCfg, email, logf)
		if err != nil {
			return nil, err
		}
		saveToken(tokenPath, tok, logf)
		tokenSource = oauthCfg.TokenSource(ctx, tok)
	} else if newTok.AccessToken != tok.AccessToken {
		saveToken(tokenPath, newTok, logf)
	}

	return oauth2.NewClient(ctx, tokenSource), nil
}

func getTokenFromWeb(config *oauth2.Config, email string, logf loggerFunc) (*oauth2.Token, error) {
	if config == nil {
		return nil, errors.New("missing oauth config")
	}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
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

	if logf != nil {
		logf("\n=== Authentication Required for %s ===\n", email)
	}
	if err := browser.OpenURL(authURL); err != nil {
		if logf != nil {
			logf("Open this URL to authorize:\n%v\n", authURL)
		}
	} else if logf != nil {
		logf("If your browser does not open, visit:\n%v\n", authURL)
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

func saveToken(path string, token *oauth2.Token, logf loggerFunc) {
	if logf != nil {
		logf("Saving credential file to: %s\n", path)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		if logf != nil {
			logf("Unable to cache oauth token: %v\n", err)
		}
		return
	}
	defer f.Close()
	if err := json.MarshalWrite(f, token); err != nil {
		if logf != nil {
			logf("Unable to cache oauth token: %v\n", err)
		}
	}
}

func randomState() (string, error) {
	const size = 16
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("unable to generate oauth state: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
