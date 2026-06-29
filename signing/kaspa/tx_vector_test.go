package kaspa

import (
	"encoding/hex"
	"testing"

	"github.com/KernelFlowLabs/wallet-sdk/signing"
)

const (
	vecMnemonic       = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	vecPath           = "m/44'/111111'/0'/0/0"
	vecAddr           = "kaspa:qqd6e65yefepe9wk0m9vuxdufxd80sphy67gwwd0vdaumzdt4tc9s3qt0lqeh"
	vecPub            = "1bacea84ca721c95d67ecace19bc499a77c03726bc8739af637bcd89abaaf058"
	vecNativeSigHash  = "63573b5855f6b356343e038f0ce0834711db1be1e115d87a713ccf1a080d0231"
	vecNativeUnsigned = "7b2276657273696f6e223a302c22696e70757473223a5b7b2270726576696f75734f7574706f696e74223a7b227472616e73616374696f6e4944223a2231323334353637383930313233343536373839303132333435363738393031323334353637383930313233343536373839303132333435363738393031323334222c22696e646578223a307d2c227369676e6174757265536372697074223a22222c2273657175656e6365223a302c227369674f70436f756e74223a317d5d2c226f757470757473223a5b7b2276616c7565223a35303030303030302c227363726970745075626c69634b6579223a7b22736372697074223a223230316261636561383463613732316339356436376563616365313962633439396137376330333732366263383733396166363337626364383961626161663035386163222c2276657273696f6e223a307d7d2c7b2276616c7565223a34393939393030302c227363726970745075626c69634b6579223a7b22736372697074223a223230316261636561383463613732316339356436376563616365313962633439396137376330333732366263383733396166363337626364383961626161663035386163222c2276657273696f6e223a307d7d5d2c226c6f636b54696d65223a302c227375626e6574776f726b4944223a2230303030303030303030303030303030303030303030303030303030303030303030303030303030222c22676173223a302c227061796c6f6164223a22227d"
	vecKrc20Params    = "201bacea84ca721c95d67ecace19bc499a77c03726bc8739af637bcd89abaaf058ac0063076b6173706c6578004c8a7b2270223a226b72632d3230222c226f70223a227472616e73666572222c227469636b223a224b415350222c22616d74223a2231303030303030303030222c22746f223a226b617370613a7171643665363579656665706539776b306d39767578647566786438307370687936376777776430766461756d7a64743474633973337174306c716568227d68_aa207d93a303c210461e25fffa7cacce71dbc20b4b7178803d43e2928902e999a48187_kaspa:pp7e8gcrcggyv839lla8etxww8duyz6tw9ugq02ru2fgjqhfnxjgz7nsc3hmx"
)

func TestKaspaVector(t *testing.T) {
	acc, err := NewAccountFromMnemonic(vecMnemonic, vecPath)
	if err != nil {
		t.Fatal(err)
	}
	if acc.Address() != vecAddr {
		t.Fatalf("address mismatch\n got=%s\nwant=%s", acc.Address(), vecAddr)
	}
	if acc.PublicKeyHex() != vecPub {
		t.Fatalf("pubkey mismatch\n got=%s\nwant=%s", acc.PublicKeyHex(), vecPub)
	}

	pub := acc.PublicKey()
	sender := acc.Address()

	utxos := signing.NewUtxoList()
	utxos.List = append(utxos.List, &signing.UtxoInfo{
		Hash:          "1234567890123456789012345678901234567890123456789012345678901234",
		Index:         "0",
		Value:         "100000000",
		Script:        "20" + hex.EncodeToString(pub) + "ac",
		Version:       "0",
		IsCoinbase:    "false",
		BlockDAAScore: "0",
	})

	tb := NewTxBuilder(&Ingredient{
		TxType:    signing.TxTypeTransfer,
		Sender:    sender,
		Recipient: sender,
		Amount:    "50000000",
		Fee:       "1000",
		Utxos:     utxos,
	})
	if err := tb.Build(); err != nil {
		t.Fatal(err)
	}
	if got := tb.GetSigHash(); len(got) != 1 || got[0] != vecNativeSigHash {
		t.Fatalf("native sigHash mismatch\n got=%v\nwant=%s", got, vecNativeSigHash)
	}
	if got := tb.GetUnsignedHex(); got != vecNativeUnsigned {
		t.Fatalf("native unsigned mismatch\n got=%s\nwant=%s", got, vecNativeUnsigned)
	}

	krc := &Krc20Params{P: "krc-20", Op: "transfer", Tick: "KASP", Amt: "1000000000", To: sender}
	out, err := GetKrc20Params(krc, pub)
	if err != nil {
		t.Fatal(err)
	}
	if out != vecKrc20Params {
		t.Fatalf("krc20 params mismatch\n got=%s\nwant=%s", out, vecKrc20Params)
	}
}
