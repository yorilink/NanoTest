package main

import "context"

// Plugin runs after a robot has logged in successfully.
// Add protocol pressure tests by implementing this interface and registering
// them in buildPlugins.
type Plugin interface {
	Name() string
	Run(ctx context.Context, c *Client, accountID int64, login *GateResponse) error
}

func buildPlugins(_ Config) []Plugin {
	return nil
}
