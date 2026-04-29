// Package benchmark owns the control-plane benchmark option contract. The
// fields mirror the small but recurring knobs exposed by pgbench, HammerDB,
// and sysbench: scale, client concurrency, run/warmup time, workload family,
// and optional rate limiting.
package benchmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	defaultScale           = 10
	defaultClients         = 1
	defaultDurationSeconds = 60
	defaultWarmupSeconds   = 10
	defaultReadPercent     = 80
)

type Profile string

const (
	ProfileRead        Profile = "read"
	ProfileWrite       Profile = "write"
	ProfileTransaction Profile = "transaction"
	ProfileJoin        Profile = "join"
	ProfileLock        Profile = "lock"
	ProfileMixed       Profile = "mixed"
)

type TransactionMix string

const (
	TransactionMixBalanced   TransactionMix = "balanced"
	TransactionMixReadHeavy  TransactionMix = "read-heavy"
	TransactionMixWriteHeavy TransactionMix = "write-heavy"
)

type StartOptions struct {
	Scale           int
	Clients         int
	DurationSeconds int
	WarmupSeconds   int
	Profile         Profile
	ReadPercent     *int
	TransactionMix  TransactionMix
	TargetTPS       *int
}

type AlterOptions struct {
	Clients   *int
	TargetTPS *int
}

type startPayload struct {
	Scale           *int            `json:"scale"`
	Clients         *int            `json:"clients"`
	DurationSeconds *int            `json:"duration_seconds"`
	WarmupSeconds   *int            `json:"warmup_seconds"`
	Profile         *Profile        `json:"profile"`
	ReadPercent     *int            `json:"read_percent"`
	TransactionMix  *TransactionMix `json:"transaction_mix"`
	TargetTPS       *int            `json:"target_tps"`
}

type alterPayload struct {
	Clients   *int `json:"clients"`
	TargetTPS *int `json:"target_tps"`
}

func DecodeStartOptions(r io.Reader) (StartOptions, error) {
	var payload startPayload
	if err := decodeJSON(r, &payload); err != nil {
		return StartOptions{}, err
	}

	options := StartOptions{
		Scale:           defaultScale,
		Clients:         defaultClients,
		DurationSeconds: defaultDurationSeconds,
		WarmupSeconds:   defaultWarmupSeconds,
		Profile:         ProfileMixed,
	}

	if payload.Scale != nil {
		options.Scale = *payload.Scale
	}
	if payload.Clients != nil {
		options.Clients = *payload.Clients
	}
	if payload.DurationSeconds != nil {
		options.DurationSeconds = *payload.DurationSeconds
	}
	if payload.WarmupSeconds != nil {
		options.WarmupSeconds = *payload.WarmupSeconds
	}
	if payload.Profile != nil {
		options.Profile = *payload.Profile
	}
	if payload.ReadPercent != nil {
		options.ReadPercent = intPtr(*payload.ReadPercent)
	} else if options.Profile == ProfileMixed {
		options.ReadPercent = intPtr(defaultReadPercent)
	}
	if payload.TransactionMix != nil {
		options.TransactionMix = *payload.TransactionMix
	} else if options.Profile == ProfileTransaction {
		options.TransactionMix = TransactionMixBalanced
	}
	if payload.TargetTPS != nil {
		options.TargetTPS = intPtr(*payload.TargetTPS)
	}

	if err := validateStartOptions(options); err != nil {
		return StartOptions{}, err
	}

	return options, nil
}

func DecodeAlterOptions(r io.Reader) (AlterOptions, error) {
	var payload alterPayload
	if err := decodeJSON(r, &payload); err != nil {
		return AlterOptions{}, err
	}

	options := AlterOptions{
		Clients:   payload.Clients,
		TargetTPS: payload.TargetTPS,
	}

	if err := validateAlterOptions(options); err != nil {
		return AlterOptions{}, err
	}

	return options, nil
}

func decodeJSON(r io.Reader, target any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err == nil {
		return fmt.Errorf("decode JSON: unexpected trailing data")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode JSON: %w", err)
	}

	return nil
}

func validateStartOptions(options StartOptions) error {
	if options.Scale < 1 {
		return fmt.Errorf("scale must be at least 1")
	}
	if options.Clients < 1 {
		return fmt.Errorf("clients must be at least 1")
	}
	if options.DurationSeconds < 1 {
		return fmt.Errorf("duration_seconds must be at least 1")
	}
	if options.WarmupSeconds < 0 {
		return fmt.Errorf("warmup_seconds must be at least 0")
	}
	if options.WarmupSeconds >= options.DurationSeconds {
		return fmt.Errorf("warmup_seconds must be less than duration_seconds")
	}
	if !isValidProfile(options.Profile) {
		return fmt.Errorf("profile must be one of read, write, transaction, join, lock, mixed")
	}
	if options.ReadPercent != nil {
		if options.Profile != ProfileMixed {
			return fmt.Errorf("read_percent is only supported for profile %q", ProfileMixed)
		}
		if *options.ReadPercent < 0 || *options.ReadPercent > 100 {
			return fmt.Errorf("read_percent must be between 0 and 100")
		}
	}
	if options.TransactionMix != "" {
		if options.Profile != ProfileTransaction {
			return fmt.Errorf("transaction_mix is only supported for profile %q", ProfileTransaction)
		}
		if !isValidTransactionMix(options.TransactionMix) {
			return fmt.Errorf("transaction_mix must be one of balanced, read-heavy, write-heavy")
		}
	}
	if options.Profile == ProfileLock && options.Clients < 2 {
		return fmt.Errorf("clients must be at least 2 for profile %q", ProfileLock)
	}
	if options.TargetTPS != nil && *options.TargetTPS < 1 {
		return fmt.Errorf("target_tps must be at least 1")
	}

	return nil
}

func isValidProfile(profile Profile) bool {
	switch profile {
	case ProfileRead, ProfileWrite, ProfileTransaction, ProfileJoin, ProfileLock, ProfileMixed:
		return true
	default:
		return false
	}
}

func isValidTransactionMix(mix TransactionMix) bool {
	switch mix {
	case TransactionMixBalanced, TransactionMixReadHeavy, TransactionMixWriteHeavy:
		return true
	default:
		return false
	}
}

func validateAlterOptions(options AlterOptions) error {
	if options.Clients == nil && options.TargetTPS == nil {
		return fmt.Errorf("alter request must include at least one field")
	}
	if options.Clients != nil && *options.Clients < 1 {
		return fmt.Errorf("clients must be at least 1")
	}
	if options.TargetTPS != nil && *options.TargetTPS < 1 {
		return fmt.Errorf("target_tps must be at least 1")
	}

	return nil
}

func intPtr(value int) *int {
	return &value
}
