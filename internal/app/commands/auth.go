package commands

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/baptistax/rdlxd/internal/config"
	"github.com/baptistax/rdlxd/internal/reddit"
)

const defaultRedirectURI = "http://127.0.0.1:53682/callback"

func RunAuth(args []string, stdout io.Writer) error {
	fs := newFlagSet("auth")
	clientID := fs.String("client-id", "", "reddit installed app client id")
	code := fs.String("code", "", "authorization code")
	redirectURI := fs.String("redirect-uri", defaultRedirectURI, "oauth redirect uri")
	verbose := fs.Bool("verbose", false, "show more console details")
	if err := parseFlags(fs, args); err != nil {
		return ErrUsage
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: auth takes no positional arguments", ErrUsage)
	}

	cfg := config.Default()
	if strings.TrimSpace(*clientID) != "" {
		cfg.ClientID = strings.TrimSpace(*clientID)
	}
	if cfg.ClientID == "" && strings.TrimSpace(*code) == "" {
		fmt.Fprintln(stdout, "Create a Reddit installed app, then run:")
		fmt.Fprintln(stdout, "rdlxd auth --client-id <client-id>")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "You can also set RDLXD_CLIENT_ID.")
		fmt.Fprintf(stdout, "Scopes: %s\n", cfg.ScopesString())
		return nil
	}
	flow := reddit.InstalledAppFlow{Config: reddit.OAuthConfig{
		ClientID:    cfg.ClientID,
		RedirectURI: *redirectURI,
		Scopes:      cfg.Scopes,
		UserAgent:   cfg.UserAgent,
	}}
	if strings.TrimSpace(*code) != "" {
		token, err := flow.ExchangeCode(context.Background(), *code)
		if err != nil {
			return err
		}
		if err := (config.FileTokenStore{Path: cfg.TokenPath}).Save(context.Background(), *token); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "OAuth token saved.")
		return nil
	}
	state, err := reddit.NewOAuthState()
	if err != nil {
		return err
	}
	authURL, err := flow.AuthorizationURL(state)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Open this Reddit authorization URL:")
	fmt.Fprintln(stdout, authURL)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "After approving the installed app, copy the code from the redirect URL and run:")
	fmt.Fprintln(stdout, "rdlxd auth --client-id <client-id> --code <code>")
	fmt.Fprintf(stdout, "Scopes: %s\n", cfg.ScopesString())
	if *verbose {
		fmt.Fprintf(stdout, "Config path: %s\n", cfg.ConfigPath)
		fmt.Fprintf(stdout, "Token path: %s\n", cfg.TokenPath)
		fmt.Fprintf(stdout, "Redirect URI: %s\n", *redirectURI)
	}
	return nil
}
