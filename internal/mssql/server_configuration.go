// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
)

// showAdvancedOptions is the option that makes the advanced options visible to sp_configure.
const showAdvancedOptions = "show advanced options"

// ServerConfiguration is one row of sys.configurations (the options of sp_configure).
type ServerConfiguration struct {
	Name string
	// Value is the configured value; ValueInUse is the one the running server uses. They differ
	// until RECONFIGURE (or, for an option that is not dynamic, a restart) has been done.
	Value      int64
	ValueInUse int64
	Minimum    int64
	Maximum    int64
	IsDynamic  bool
	IsAdvanced bool
}

// RestartRequired reports whether the configured value only takes effect after a restart.
func (s ServerConfiguration) RestartRequired() bool {
	return !s.IsDynamic && s.Value != s.ValueInUse
}

// GetServerConfiguration returns an option of sp_configure, or nil when the server has no such option.
func (c *Client) GetServerConfiguration(ctx context.Context, name string) (*ServerConfiguration, error) {
	query := `
		SELECT
			name,
			CAST(value AS bigint),
			CAST(value_in_use AS bigint),
			CAST(minimum AS bigint),
			CAST(maximum AS bigint),
			is_dynamic,
			is_advanced
		FROM sys.configurations
		WHERE name = @p1`

	var cfg ServerConfiguration
	err := c.QueryRowContext(ctx, query, name).Scan(&cfg.Name, &cfg.Value, &cfg.ValueInUse, &cfg.Minimum, &cfg.Maximum, &cfg.IsDynamic, &cfg.IsAdvanced)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server configuration: %w", err)
	}
	return &cfg, nil
}

// configurationStep is one statement of a configuration change.
type configurationStep struct {
	Statement string
	Args      []interface{}
}

const (
	spConfigure = "EXEC master.dbo.sp_configure @configname = @p1, @configvalue = @p2"
	reconfigure = "RECONFIGURE"
)

// planConfigurationChange returns the statements that set an option. An advanced option can only be
// changed while "show advanced options" is on, so when it is off it is switched on for the change and
// switched off again afterwards; the option the caller asked for is never "show advanced options" itself
// in that case. It validates the value against the range the server reports.
func planConfigurationChange(cfg ServerConfiguration, value int64, showAdvancedInUse bool) ([]configurationStep, error) {
	if value < cfg.Minimum || value > cfg.Maximum {
		return nil, fmt.Errorf("the value %d is out of the range of %q: %d to %d", value, cfg.Name, cfg.Minimum, cfg.Maximum)
	}

	var steps []configurationStep
	needsShow := cfg.IsAdvanced && !showAdvancedInUse && cfg.Name != showAdvancedOptions
	if needsShow {
		steps = append(steps, configurationStep{spConfigure, []interface{}{showAdvancedOptions, int64(1)}}, configurationStep{Statement: reconfigure})
	}
	steps = append(steps, configurationStep{spConfigure, []interface{}{cfg.Name, value}}, configurationStep{Statement: reconfigure})
	if needsShow {
		steps = append(steps, configurationStep{spConfigure, []interface{}{showAdvancedOptions, int64(0)}}, configurationStep{Statement: reconfigure})
	}
	return steps, nil
}

// SetServerConfiguration sets an option of sp_configure and applies it with RECONFIGURE.
func (c *Client) SetServerConfiguration(ctx context.Context, name string, value int64) error {
	cfg, err := c.GetServerConfiguration(ctx, name)
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("the server has no configuration option %q", name)
	}

	showAdvanced, err := c.GetServerConfiguration(ctx, showAdvancedOptions)
	if err != nil {
		return err
	}

	steps, err := planConfigurationChange(*cfg, value, showAdvanced != nil && showAdvanced.ValueInUse == 1)
	if err != nil {
		return err
	}

	for i, step := range steps {
		if _, err := c.ExecContext(ctx, step.Statement, step.Args...); err != nil {
			// Leave "show advanced options" as it was when a later step fails.
			if i > 0 && cfg.IsAdvanced && showAdvanced != nil && showAdvanced.ValueInUse == 0 && name != showAdvancedOptions {
				_, _ = c.ExecContext(ctx, spConfigure, showAdvancedOptions, int64(0))
				_, _ = c.ExecContext(ctx, reconfigure)
			}
			return fmt.Errorf("failed to set %q: %w", name, err)
		}
	}
	return nil
}
