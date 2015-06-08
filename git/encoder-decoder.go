package git

type Encoder interface {
	Encode([]byte) error
}

type Decoder interface {
	Decode(*[]byte) error
}
