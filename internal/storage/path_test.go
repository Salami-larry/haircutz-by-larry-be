package storage

import (
	"testing"
)

func TestObjectPathFromPublicURL(t *testing.T) {
	bucket := "haircutz"
	base := "https://abc.supabase.co/storage/v1/object/public/" + bucket + "/"

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{
			name: "valid",
			raw:  base + "hairstyles/abc-123.jpg",
			want: "hairstyles/abc-123.jpg",
		},
		{
			name: "with query",
			raw:  base + "hairstyles/abc-123.jpg?token=x",
			want: "hairstyles/abc-123.jpg",
		},
		{
			name:    "wrong bucket",
			raw:     "https://abc.supabase.co/storage/v1/object/public/other/hairstyles/x.jpg",
			wantErr: true,
		},
		{
			name:    "outside hairstyles prefix",
			raw:     base + "avatars/x.jpg",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := objectPathFromPublicURL(tc.raw, bucket)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
