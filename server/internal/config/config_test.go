package config

import "testing"

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "local environment allows wildcard cors",
			cfg: Config{
				Env:             "local",
				CacheMode:       "local",
				CORSAllowOrigin: "*",
			},
			wantErr: false,
		},
		{
			name: "non local requires pg dsn",
			cfg: Config{
				Env:             "production",
				CacheMode:       "local",
				CORSAllowOrigin: "https://admin.example.com",
			},
			wantErr: true,
		},
		{
			name: "non local requires cors allow origin",
			cfg: Config{
				Env:             "production",
				PGDSN:           "postgres://user:pass@localhost:5432/app?sslmode=disable",
				CacheMode:       "local",
				CORSAllowOrigin: "",
			},
			wantErr: true,
		},
		{
			name: "non local rejects wildcard cors",
			cfg: Config{
				Env:             "production",
				PGDSN:           "postgres://user:pass@localhost:5432/app?sslmode=disable",
				CacheMode:       "local",
				CORSAllowOrigin: "*",
			},
			wantErr: true,
		},
		{
			name: "non local accepts explicit cors and redis",
			cfg: Config{
				Env:             "production",
				PGDSN:           "postgres://user:pass@localhost:5432/app?sslmode=disable",
				CacheMode:       "redis",
				RedisURL:        "redis://127.0.0.1:6379/0",
				CORSAllowOrigin: "https://admin.example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
