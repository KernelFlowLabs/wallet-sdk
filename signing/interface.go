package signing

type AccountHandler interface {
	PrivateKey() []byte
	PublicKey() []byte
	PrivateKeyHex() string
	PublicKeyHex() string
	Address() string
	SignData(data []byte) ([]byte, error)
	VerifySignData(data, sig []byte) bool
	Wipe()
}

type TxBuilderHandler interface {
	Build() error
	Sign(privateKey []byte) (string, error)
	ConcatSignature(signature string, isDerFormat bool) (string, error)
	GetTxHash() string
	GetSigHash() []string
	GetUnsignedHex() string
	SetSigHash(sigHash []string)
	SetUnsignedHex(unsignedHex string)
}
