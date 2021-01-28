package iota

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const (
	// The minimum index of a referenced UTXO.
	RefUTXOIndexMin = 0
	// The maximum index of a referenced UTXO.
	RefUTXOIndexMax = 126

	// The size of a UTXO input: input type + tx id + index
	UTXOInputSize = SmallTypeDenotationByteSize + TransactionIDLength + UInt16ByteSize
)

// UTXOInputID defines the identifier for an UTXO input which consists
// out of the referenced transaction ID and the given output index.
type UTXOInputID [TransactionIDLength + UInt16ByteSize]byte

// ToHex converts the UTXOInputID to its hex representation.
func (utxoInputID UTXOInputID) ToHex() string {
	return fmt.Sprintf("%x", utxoInputID)
}

// UTXOInputIDs is a slice of UTXOInputID.
type UTXOInputIDs []UTXOInputID

// ToHex converts all UTXOInput to their hex string representation.
func (utxoInputIDs UTXOInputIDs) ToHex() []string {
	ids := make([]string, len(utxoInputIDs))
	for i := range utxoInputIDs {
		ids[i] = fmt.Sprintf("%x", utxoInputIDs[i])
	}
	return ids
}

// UTXOInput references an unspent transaction output by the Transaction's ID and the corresponding index of the output.
type UTXOInput struct {
	// The transaction ID of the referenced transaction.
	TransactionID [TransactionIDLength]byte
	// The output index of the output on the referenced transaction.
	TransactionOutputIndex uint16
}

// ID returns the UTXOInputID.
func (u *UTXOInput) ID() UTXOInputID {
	var id UTXOInputID
	copy(id[:TransactionIDLength], u.TransactionID[:])
	binary.LittleEndian.PutUint16(id[TransactionIDLength:], u.TransactionOutputIndex)
	return id
}

func (u *UTXOInput) Deserialize(data []byte, deSeriMode DeSerializationMode) (int, error) {
	return NewDeserializer(data).
		AbortIf(func(err error) error {
			if deSeriMode.HasMode(DeSeriModePerformValidation) {
				if err := checkMinByteLength(UTXOInputSize, len(data)); err != nil {
					return fmt.Errorf("invalid UTXO input bytes: %w", err)
				}
				if err := checkTypeByte(data, InputUTXO); err != nil {
					return fmt.Errorf("unable to deserialize UTXO input: %w", err)
				}
			}
			return nil
		}).
		Skip(SmallTypeDenotationByteSize, func(err error) error {
			return fmt.Errorf("unable to skip UTXO input type during deserialization: %w", err)
		}).
		ReadArrayOf32Bytes(&u.TransactionID, func(err error) error {
			return fmt.Errorf("unable to deserialize transaction ID in UTXO input: %w", err)
		}).
		ReadNum(&u.TransactionOutputIndex, func(err error) error {
			return fmt.Errorf("unable to deserialize transaction output index in UTXO input: %w", err)
		}).
		AbortIf(func(err error) error {
			if deSeriMode.HasMode(DeSeriModePerformValidation) {
				if err := utxoInputRefBoundsValidator(-1, u); err != nil {
					return fmt.Errorf("%w: unable to deserialize UTXO input", err)
				}
			}
			return nil
		}).
		Done()
}

func (u *UTXOInput) Serialize(deSeriMode DeSerializationMode) (data []byte, err error) {
	return NewSerializer().
		AbortIf(func(err error) error {
			if deSeriMode.HasMode(DeSeriModePerformValidation) {
				if err := utxoInputRefBoundsValidator(-1, u); err != nil {
					return fmt.Errorf("%w: unable to serialize UTXO input", err)
				}
			}
			return nil
		}).
		WriteNum(InputUTXO, func(err error) error {
			return fmt.Errorf("unable to serialize UTXO input type ID: %w", err)
		}).
		WriteBytes(u.TransactionID[:], func(err error) error {
			return fmt.Errorf("unable to serialize UTXO input transaction ID: %w", err)
		}).
		WriteNum(u.TransactionOutputIndex, func(err error) error {
			return fmt.Errorf("unable to serialize UTXO input transaction output index: %w", err)
		}).Serialize()
}

func (u *UTXOInput) MarshalJSON() ([]byte, error) {
	jsonUTXO := &jsonutxoinput{}
	jsonUTXO.TransactionID = hex.EncodeToString(u.TransactionID[:])
	jsonUTXO.TransactionOutputIndex = int(u.TransactionOutputIndex)
	jsonUTXO.Type = int(InputUTXO)
	return json.Marshal(jsonUTXO)
}

func (u *UTXOInput) UnmarshalJSON(bytes []byte) error {
	jsonUTXO := &jsonutxoinput{}
	if err := json.Unmarshal(bytes, jsonUTXO); err != nil {
		return err
	}
	seri, err := jsonUTXO.ToSerializable()
	if err != nil {
		return err
	}
	*u = *seri.(*UTXOInput)
	return nil
}