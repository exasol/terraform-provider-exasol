package provider

import (
	"context"
	"database/sql"

	"terraform-provider-exasol/internal/exasolclient"

	"github.com/exasol/exasol-driver-go"
)

// Re-export the concrete type so the rest of the provider can keep using provider.Client.
type Client = exasolclient.Client

// NewClient builds the correct Exasol DSN and opens the connection.
// It now always includes the `encryption` flag, and lets the caller
// control whether the server certificate is validated.
func NewClient(ctx context.Context, c *ProviderConfig) (*Client, error) {

	dsn := exasol.NewConfig(c.User, c.Password).
		Host(c.Host).
		Port(int(c.Port)).
		ValidateServerCertificate(c.ValidateServerCertificate).
		String()

	db, err := sql.Open("exasol", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &Client{DB: db}, nil
}
