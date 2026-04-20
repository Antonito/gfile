package transfer

import "testing"

func TestRoleString(t *testing.T) {
	cases := []struct {
		role Role
		want string
	}{
		{RoleSender, "sender"},
		{RoleReceiver, "receiver"},
		{Role(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.role.String(); got != tc.want {
			t.Errorf("Role(%d).String() = %q, want %q", tc.role, got, tc.want)
		}
	}
}
