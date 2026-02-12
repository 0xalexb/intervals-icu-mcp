// Package tools contains MCP tool registrations for the intervals.icu MCP server.
package tools

import (
	"errors"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	errMissingEventID    = errors.New("event_id is required")
	errMissingDate       = errors.New("date is required")
	errMissingOldest     = errors.New("oldest is required")
	errMissingNewest     = errors.New("newest is required")
	errMissingActivityID = errors.New("activity_id is required")
	errMissingType       = errors.New("type is required")
	errMissingName       = errors.New("name is required")
	errMissingStartDate  = errors.New("start_date_local is required")
	errMissingCategory   = errors.New("category is required")
)

func validateDateFormat(s string) error {
	_, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return fmt.Errorf("invalid date format %q, expected yyyy-MM-dd: %w", s, err)
	}

	return nil
}

// ToolRegistration is a function that registers an MCP tool on the given server.
type ToolRegistration func(server *mcp.Server)
