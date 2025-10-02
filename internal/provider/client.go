package provider

import (
	"context"
	"database/sql"
	"strings"

	"terraform-provider-exasol/internal/exasolclient"

	"github.com/exasol/exasol-driver-go"
	"github.com/exasol/exasol-driver-go/pkg/dsn"
)

// Re-export the concrete type so the rest of the provider can keep using provider.Client.
type Client = exasolclient.Client

// NewClient builds the correct Exasol DSN and opens the connection.
// It now always includes the `encryption` flag, and lets the caller
// control whether the server certificate is validated.
func NewClient(ctx context.Context, c *ProviderConfig) (*Client, error) {
	var config *dsn.DSNConfigBuilder

	// Detect if password is a PAT token
	if strings.HasPrefix(c.Password, "exa_pat_") {
		config = exasol.NewConfigWithAccessToken(c.Password) // Use PAT as access token
	} else {
		config = exasol.NewConfig(c.User, c.Password) // Use regular password
	}

	dsnString := config.Host(c.Host).
		Port(int(c.Port)).
		ValidateServerCertificate(c.ValidateServerCertificate).
		String()

	db, err := sql.Open("exasol", dsnString)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &Client{DB: db}, nil
}
