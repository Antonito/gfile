package transfer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	qrterminal "github.com/mdp/qrterminal/v3"

	"github.com/antonito/gfile/internal/output"
	"github.com/antonito/gfile/internal/stream"
	"github.com/antonito/gfile/internal/utils"
)

// qrPayloadLimit caps SDP length for QR rendering. Past ~1200 chars the
// grid stays scannable but becomes too dense to read comfortably in a
// terminal window.
const qrPayloadLimit = 1200

// EmitSDP writes the encoded local SDP to out, tagged with role.
func EmitSDP(out io.Writer, role Role, encoded string) {
	output.SDP(out, role.String(), encoded)
}

// MaybeShowQR renders a QR of encoded to stderr when not disabled, text mode is active,
// and the payload fits qrPayloadLimit. Oversize payloads fall back to a short notice.
func MaybeShowQR(encoded string, disableQR bool) {
	if disableQR {
		return
	}
	if output.CurrentMode() != output.ModeText {
		return
	}
	if len(encoded) <= qrPayloadLimit {
		fmt.Fprintln(os.Stderr, "\nOr scan this QR code:")
		qrterminal.GenerateWithConfig(encoded, qrterminal.Config{
			Level:          qrterminal.L,
			Writer:         os.Stderr,
			HalfBlocks:     true,
			BlackChar:      qrterminal.BLACK_BLACK,
			WhiteBlackChar: qrterminal.WHITE_BLACK,
			WhiteChar:      qrterminal.WHITE_WHITE,
			BlackWhiteChar: qrterminal.BLACK_WHITE,
			QuietZone:      1,
		})
		return
	}
	fmt.Fprintln(os.Stderr, "\n(SDP too large for a scannable QR code -- use the text above)")
}

// ReadRemoteSDP prompts, reads from in, optionally unwraps a {"sdp":"..."} envelope,
// and returns the encoded SDP. Re-prompts on failure.
func ReadRemoteSDP(in io.Reader) (string, error) {
	output.Prompt("Please, paste the remote SDP:")

	for {
		encoded, err := stream.MustReadStream(in)
		if err != nil {
			output.Prompt("Invalid SDP, try again...")
			continue
		}

		var env struct {
			SDP string `json:"sdp"`
		}
		if jErr := json.Unmarshal([]byte(encoded), &env); jErr == nil && env.SDP != "" {
			encoded = env.SDP
		}

		if _, derr := utils.DecodeSDP(encoded); derr == nil {
			return encoded, nil
		}
	}
}
