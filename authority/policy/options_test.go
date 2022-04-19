package policy

import (
	"testing"
)

func TestX509PolicyOptions_IsWildcardLiteralAllowed(t *testing.T) {
	trueValue := true
	falseValue := false
	tests := []struct {
		name    string
		options *X509PolicyOptions
		want    bool
	}{
		{
			name:    "nil-options",
			options: nil,
			want:    true,
		},
		{
			name: "nil",
			options: &X509PolicyOptions{
				AllowWildcardLiteral: nil,
			},
			want: false,
		},
		{
			name: "set-true",
			options: &X509PolicyOptions{
				AllowWildcardLiteral: &trueValue,
			},
			want: true,
		},
		{
			name: "set-false",
			options: &X509PolicyOptions{
				AllowWildcardLiteral: &falseValue,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.options.IsWildcardLiteralAllowed(); got != tt.want {
				t.Errorf("X509PolicyOptions.IsWildcardLiteralAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestX509PolicyOptions_ShouldVerifySubjectCommonName(t *testing.T) {
	trueValue := true
	falseValue := false
	tests := []struct {
		name    string
		options *X509PolicyOptions
		want    bool
	}{
		{
			name:    "nil-options",
			options: nil,
			want:    false,
		},
		{
			name: "nil",
			options: &X509PolicyOptions{
				VerifySubjectCommonName: nil,
			},
			want: true,
		},
		{
			name: "set-true",
			options: &X509PolicyOptions{
				VerifySubjectCommonName: &trueValue,
			},
			want: true,
		},
		{
			name: "set-false",
			options: &X509PolicyOptions{
				VerifySubjectCommonName: &falseValue,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.options.ShouldVerifySubjectCommonName(); got != tt.want {
				t.Errorf("X509PolicyOptions.ShouldVerifySubjectCommonName() = %v, want %v", got, tt.want)
			}
		})
	}
}
