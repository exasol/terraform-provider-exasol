package exasolclient

import "database/sql"

// Client is the minimal interface/resources need.
type Client struct {
	DB *sql.DB
}
