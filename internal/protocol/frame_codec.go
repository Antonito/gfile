package protocol

// Codec identifies how DATA payloads are encoded on the wire.
type Codec uint8

const (
	// CodecNone means the DATA payload is raw file bytes (no compression).
	CodecNone Codec = 0x00

	// CodecZstd means each DATA payload is an independent zstd frame.
	//
	// The DATA offset is the byte position in the decompressed stream.
	CodecZstd Codec = 0x01
)
