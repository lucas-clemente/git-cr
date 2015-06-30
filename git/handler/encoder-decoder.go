package handler

// Encoder is used for sending data via pkt-line
// Sending a `nil` does an ACK.
type Encoder interface {
	Encode([]byte) error
}

// Decoder is used to decode data using pkt-line
// ACKs are decoded as `nil`.
type Decoder interface {
	Decode(*[]byte) error

	// Read bypasses the pkt-line decoding, used when receiving packfiles
	Read(p []byte) (n int, err error)
}
