package benchmark_test

import (
	"encoding/json"
	"strings"
	"testing"

	"pg_gobench/internal/benchmark"
)

func TestStartOptionsMarshalJSONUsesHTTPFieldNames(t *testing.T) {
	payload, err := json.Marshal(benchmark.StartOptions{
		Scale:           12,
		Clients:         4,
		DurationSeconds: 90,
		WarmupSeconds:   15,
		Reset:           true,
		Profile:         benchmark.ProfileMixed,
		ReadPercent:     intPtr(70),
		TargetTPS:       intPtr(220),
	})
	if err != nil {
		t.Fatalf("Marshal StartOptions: %v", err)
	}

	if got := string(payload); got != `{"scale":12,"clients":4,"duration_seconds":90,"warmup_seconds":15,"reset":true,"profile":"mixed","read_percent":70,"transaction_mix":"","target_tps":220}` {
		t.Fatalf("Marshal StartOptions = %s", got)
	}
}

func TestDecodeStartOptionsAppliesDefaultsForMinimalPayload(t *testing.T) {
	options, err := benchmark.DecodeStartOptions(strings.NewReader(`{"scale":12}`))
	if err != nil {
		t.Fatalf("DecodeStartOptions returned error: %v", err)
	}

	if options.Scale != 12 {
		t.Fatalf("Scale = %d, want %d", options.Scale, 12)
	}
	if options.Clients != 1 {
		t.Fatalf("Clients = %d, want %d", options.Clients, 1)
	}
	if options.DurationSeconds != 60 {
		t.Fatalf("DurationSeconds = %d, want %d", options.DurationSeconds, 60)
	}
	if options.WarmupSeconds != 10 {
		t.Fatalf("WarmupSeconds = %d, want %d", options.WarmupSeconds, 10)
	}
	if options.Reset {
		t.Fatal("Reset = true, want false")
	}
	if options.Profile != benchmark.ProfileMixed {
		t.Fatalf("Profile = %q, want %q", options.Profile, benchmark.ProfileMixed)
	}
	if options.ReadPercent == nil {
		t.Fatal("ReadPercent = nil, want default value")
	}
	if *options.ReadPercent != 80 {
		t.Fatalf("ReadPercent = %d, want %d", *options.ReadPercent, 80)
	}
	if options.TargetTPS != nil {
		t.Fatalf("TargetTPS = %v, want nil", *options.TargetTPS)
	}
	if options.TransactionMix != "" {
		t.Fatalf("TransactionMix = %q, want empty", options.TransactionMix)
	}
}

func TestDecodeStartOptionsRejectsUnknownFields(t *testing.T) {
	_, err := benchmark.DecodeStartOptions(strings.NewReader(`{"scale":12,"bogus":true}`))
	if err == nil {
		t.Fatal("DecodeStartOptions returned nil error for unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("DecodeStartOptions error = %q, want unknown field message", err)
	}
}

func TestDecodeStartOptionsDefaultsTransactionMixForTransactionProfile(t *testing.T) {
	options, err := benchmark.DecodeStartOptions(strings.NewReader(`{"profile":"transaction"}`))
	if err != nil {
		t.Fatalf("DecodeStartOptions returned error: %v", err)
	}
	if options.TransactionMix != benchmark.TransactionMixBalanced {
		t.Fatalf("TransactionMix = %q, want %q", options.TransactionMix, benchmark.TransactionMixBalanced)
	}
	if options.Reset {
		t.Fatal("Reset = true, want false")
	}
	if options.ReadPercent != nil {
		t.Fatalf("ReadPercent = %v, want nil", *options.ReadPercent)
	}
}

func TestDecodeStartOptionsAcceptsExplicitResetFlag(t *testing.T) {
	options, err := benchmark.DecodeStartOptions(strings.NewReader(`{"scale":12,"reset":true}`))
	if err != nil {
		t.Fatalf("DecodeStartOptions returned error: %v", err)
	}

	if !options.Reset {
		t.Fatal("Reset = false, want true")
	}
}

func TestDecodeAlterOptionsAcceptsClientsAndTargetTPS(t *testing.T) {
	options, err := benchmark.DecodeAlterOptions(strings.NewReader(`{"clients":4,"target_tps":200}`))
	if err != nil {
		t.Fatalf("DecodeAlterOptions returned error: %v", err)
	}
	if options.Clients == nil {
		t.Fatal("Clients = nil, want value")
	}
	if *options.Clients != 4 {
		t.Fatalf("Clients = %d, want %d", *options.Clients, 4)
	}
	if options.TargetTPS == nil {
		t.Fatal("TargetTPS = nil, want value")
	}
	if *options.TargetTPS != 200 {
		t.Fatalf("TargetTPS = %d, want %d", *options.TargetTPS, 200)
	}
}

func TestDecodeStartOptionsValidatesScaleClientsAndProfileSpecificRules(t *testing.T) {
	testCases := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "scale must be positive",
			payload: `{"scale":0}`,
			want:    "scale must be at least 1",
		},
		{
			name:    "clients must be positive",
			payload: `{"clients":0}`,
			want:    "clients must be at least 1",
		},
		{
			name:    "warmup must fit within duration",
			payload: `{"duration_seconds":10,"warmup_seconds":10}`,
			want:    "warmup_seconds must be less than duration_seconds",
		},
		{
			name:    "mixed read percent stays bounded",
			payload: `{"profile":"mixed","read_percent":101}`,
			want:    "read_percent must be between 0 and 100",
		},
		{
			name:    "read percent only applies to mixed",
			payload: `{"profile":"read","read_percent":50}`,
			want:    "read_percent is only supported",
		},
		{
			name:    "transaction mix only applies to transaction",
			payload: `{"profile":"write","transaction_mix":"balanced"}`,
			want:    "transaction_mix is only supported",
		},
		{
			name:    "lock profile requires contention",
			payload: `{"profile":"lock","clients":1}`,
			want:    "clients must be at least 2",
		},
		{
			name:    "target tps must be positive",
			payload: `{"target_tps":0}`,
			want:    "target_tps must be at least 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := benchmark.DecodeStartOptions(strings.NewReader(tc.payload))
			if err == nil {
				t.Fatal("DecodeStartOptions returned nil error for invalid payload")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("DecodeStartOptions error = %q, want message containing %q", err, tc.want)
			}
		})
	}
}

func TestDecodeAlterOptionsRejectsEmptyAndUnsupportedFields(t *testing.T) {
	testCases := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "empty payload is rejected",
			payload: `{}`,
			want:    "at least one field",
		},
		{
			name:    "scale changes are rejected",
			payload: `{"scale":20}`,
			want:    "unknown field",
		},
		{
			name:    "profile changes are rejected",
			payload: `{"profile":"read"}`,
			want:    "unknown field",
		},
		{
			name:    "duration changes are rejected",
			payload: `{"duration_seconds":30}`,
			want:    "unknown field",
		},
		{
			name:    "invalid clients are rejected",
			payload: `{"clients":0}`,
			want:    "clients must be at least 1",
		},
		{
			name:    "invalid target tps is rejected",
			payload: `{"target_tps":0}`,
			want:    "target_tps must be at least 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := benchmark.DecodeAlterOptions(strings.NewReader(tc.payload))
			if err == nil {
				t.Fatal("DecodeAlterOptions returned nil error for invalid payload")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("DecodeAlterOptions error = %q, want message containing %q", err, tc.want)
			}
		})
	}
}

func TestStartOptionsApplyAlterUpdatesOnlyPermittedRuntimeFields(t *testing.T) {
	original := benchmark.StartOptions{
		Scale:           10,
		Clients:         4,
		DurationSeconds: 90,
		WarmupSeconds:   15,
		Profile:         benchmark.ProfileMixed,
		ReadPercent:     intPtr(75),
		TargetTPS:       intPtr(200),
	}
	updated, err := original.ApplyAlter(benchmark.AlterOptions{
		Clients:   intPtr(8),
		TargetTPS: intPtr(350),
	})
	if err != nil {
		t.Fatalf("ApplyAlter returned error: %v", err)
	}

	if updated.Scale != original.Scale {
		t.Fatalf("Scale = %d, want %d", updated.Scale, original.Scale)
	}
	if updated.Clients != 8 {
		t.Fatalf("Clients = %d, want %d", updated.Clients, 8)
	}
	if updated.DurationSeconds != original.DurationSeconds {
		t.Fatalf("DurationSeconds = %d, want %d", updated.DurationSeconds, original.DurationSeconds)
	}
	if updated.Profile != original.Profile {
		t.Fatalf("Profile = %q, want %q", updated.Profile, original.Profile)
	}
	if updated.TargetTPS == nil {
		t.Fatal("TargetTPS = nil, want value")
	}
	if *updated.TargetTPS != 350 {
		t.Fatalf("TargetTPS = %d, want %d", *updated.TargetTPS, 350)
	}
	if original.Clients != 4 {
		t.Fatalf("original Clients = %d, want %d", original.Clients, 4)
	}
	if original.TargetTPS == nil {
		t.Fatal("original TargetTPS = nil, want value")
	}
	if *original.TargetTPS != 200 {
		t.Fatalf("original TargetTPS = %d, want %d", *original.TargetTPS, 200)
	}
}

func intPtr(value int) *int {
	return &value
}
