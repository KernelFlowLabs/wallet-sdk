package kaspa

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

type DomainTransactionWrapper struct {
	*externalapi.DomainTransaction
}

func (tw *DomainTransactionWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Version      uint16          `json:"version"`
		Inputs       []InputWrapper  `json:"inputs"`
		Outputs      []OutputWrapper `json:"outputs"`
		LockTime     uint64          `json:"lockTime"`
		SubnetworkID string          `json:"subnetworkID"`
		Gas          uint64          `json:"gas"`
		Payload      string          `json:"payload"`
	}{
		Version:      tw.Version,
		Inputs:       wrapInputs(tw.Inputs),
		Outputs:      wrapOutputs(tw.Outputs),
		LockTime:     tw.LockTime,
		SubnetworkID: hex.EncodeToString(tw.SubnetworkID[:]),
		Gas:          tw.Gas,
		Payload:      hex.EncodeToString(tw.Payload),
	})
}

func (tw *DomainTransactionWrapper) UnmarshalJSON(data []byte) error {
	var aux struct {
		Version      uint16          `json:"version"`
		Inputs       []InputWrapper  `json:"inputs"`
		Outputs      []OutputWrapper `json:"outputs"`
		LockTime     uint64          `json:"lockTime"`
		SubnetworkID string          `json:"subnetworkID"`
		Gas          uint64          `json:"gas"`
		Payload      string          `json:"payload"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	subnetworkID, err := hex.DecodeString(aux.SubnetworkID)
	if err != nil {
		return fmt.Errorf("invalid subnetworkID: %v", err)
	}

	payload, err := hex.DecodeString(aux.Payload)
	if err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	tw.DomainTransaction = &externalapi.DomainTransaction{
		Version:      aux.Version,
		Inputs:       unwrapInputs(aux.Inputs),
		Outputs:      unwrapOutputs(aux.Outputs),
		LockTime:     aux.LockTime,
		SubnetworkID: externalapi.DomainSubnetworkID(subnetworkID),
		Gas:          aux.Gas,
		Payload:      payload,
	}

	return nil
}

type InputWrapper struct {
	*externalapi.DomainTransactionInput
}

func (iw InputWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PreviousOutpoint struct {
			TransactionID string `json:"transactionID"`
			Index         uint32 `json:"index"`
		} `json:"previousOutpoint"`
		SignatureScript string `json:"signatureScript"`
		Sequence        uint64 `json:"sequence"`
		SigOpCount      byte   `json:"sigOpCount"`
	}{
		PreviousOutpoint: struct {
			TransactionID string `json:"transactionID"`
			Index         uint32 `json:"index"`
		}{
			TransactionID: hex.EncodeToString(iw.PreviousOutpoint.TransactionID.ByteSlice()),
			Index:         iw.PreviousOutpoint.Index,
		},
		SignatureScript: hex.EncodeToString(iw.SignatureScript),
		Sequence:        iw.Sequence,
		SigOpCount:      iw.SigOpCount,
	})
}

func (iw *InputWrapper) UnmarshalJSON(data []byte) error {
	var aux struct {
		PreviousOutpoint struct {
			TransactionID string `json:"transactionID"`
			Index         uint32 `json:"index"`
		} `json:"previousOutpoint"`
		SignatureScript string `json:"signatureScript"`
		Sequence        uint64 `json:"sequence"`
		SigOpCount      byte   `json:"sigOpCount"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	transactionID, err := hex.DecodeString(aux.PreviousOutpoint.TransactionID)
	if err != nil {
		return fmt.Errorf("invalid transactionID: %v", err)
	}

	signatureScript, err := hex.DecodeString(aux.SignatureScript)
	if err != nil {
		return fmt.Errorf("invalid signatureScript: %v", err)
	}

	domainTransactionID, err := externalapi.NewDomainTransactionIDFromByteSlice(transactionID)
	if err != nil {
		return fmt.Errorf("invalid transactionID: %v", err)
	}
	iw.DomainTransactionInput = &externalapi.DomainTransactionInput{
		PreviousOutpoint: externalapi.DomainOutpoint{
			TransactionID: *domainTransactionID,
			Index:         aux.PreviousOutpoint.Index,
		},
		SignatureScript: signatureScript,
		Sequence:        aux.Sequence,
		SigOpCount:      aux.SigOpCount,
	}

	return nil
}

type OutputWrapper struct {
	*externalapi.DomainTransactionOutput
}

func (ow *OutputWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Value           uint64 `json:"value"`
		ScriptPublicKey struct {
			Script  string `json:"script"`
			Version uint16 `json:"version"`
		} `json:"scriptPublicKey"`
	}{
		Value: ow.Value,
		ScriptPublicKey: struct {
			Script  string `json:"script"`
			Version uint16 `json:"version"`
		}{
			Script:  hex.EncodeToString(ow.ScriptPublicKey.Script),
			Version: ow.ScriptPublicKey.Version,
		},
	})
}

func (ow *OutputWrapper) UnmarshalJSON(data []byte) error {
	var aux struct {
		Value           uint64 `json:"value"`
		ScriptPublicKey struct {
			Script  string `json:"script"`
			Version uint16 `json:"version"`
		} `json:"scriptPublicKey"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	script, err := hex.DecodeString(aux.ScriptPublicKey.Script)
	if err != nil {
		return fmt.Errorf("invalid script: %v", err)
	}

	ow.DomainTransactionOutput = &externalapi.DomainTransactionOutput{
		Value: aux.Value,
		ScriptPublicKey: &externalapi.ScriptPublicKey{
			Script:  script,
			Version: aux.ScriptPublicKey.Version,
		},
	}

	return nil
}

func wrapInputs(inputs []*externalapi.DomainTransactionInput) []InputWrapper {
	wrapped := make([]InputWrapper, len(inputs))
	for i, input := range inputs {
		wrapped[i] = InputWrapper{input}
	}
	return wrapped
}

func wrapOutputs(outputs []*externalapi.DomainTransactionOutput) []OutputWrapper {
	wrapped := make([]OutputWrapper, len(outputs))
	for i, output := range outputs {
		wrapped[i] = OutputWrapper{output}
	}
	return wrapped
}

func unwrapInputs(wrapped []InputWrapper) []*externalapi.DomainTransactionInput {
	unwrapped := make([]*externalapi.DomainTransactionInput, len(wrapped))
	for i, w := range wrapped {
		unwrapped[i] = w.DomainTransactionInput
	}
	return unwrapped
}

func unwrapOutputs(wrapped []OutputWrapper) []*externalapi.DomainTransactionOutput {
	unwrapped := make([]*externalapi.DomainTransactionOutput, len(wrapped))
	for i, w := range wrapped {
		unwrapped[i] = w.DomainTransactionOutput
	}
	return unwrapped
}
