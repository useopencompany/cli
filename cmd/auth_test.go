package cmd

import "testing"

func TestPersonalDisplayNamePromptLabel(t *testing.T) {
	if got, want := personalDisplayNamePromptLabel(), "What should we call you"; got != want {
		t.Fatalf("personalDisplayNamePromptLabel() = %q, want %q", got, want)
	}
}

func TestShouldPromptForDisplayName(t *testing.T) {
	tests := []struct {
		name    string
		orgName string
		orgID   string
		want    bool
	}{
		{
			name: "missing org name",
			want: true,
		},
		{
			name:    "org name still looks like raw org id",
			orgName: "org_123",
			orgID:   "org_123",
			want:    true,
		},
		{
			name:    "legacy personal org name",
			orgName: "johann personal",
			orgID:   "org_123",
			want:    true,
		},
		{
			name:    "named organization already exists",
			orgName: "Johann's Workspace",
			orgID:   "org_123",
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldPromptForDisplayName(tc.orgName, tc.orgID); got != tc.want {
				t.Fatalf("shouldPromptForDisplayName(%q, %q) = %t, want %t", tc.orgName, tc.orgID, got, tc.want)
			}
		})
	}
}
