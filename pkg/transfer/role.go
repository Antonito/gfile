package transfer

// Role identifies the sending or receiving end of a transfer.
type Role uint8

const (
	// RoleSender originates the transfer and emits the SDP offer.
	RoleSender Role = iota
	// RoleReceiver accepts the offer and writes into the target file.
	RoleReceiver
)

// String returns the lowercase identifier for r, or "unknown".
func (r Role) String() string {
	switch r {
	case RoleSender:
		return "sender"
	case RoleReceiver:
		return "receiver"
	default:
		return "unknown"
	}
}
