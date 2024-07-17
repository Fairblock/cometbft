package keytypes

import (
	"fmt"
	"strings"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/bls12381"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/crypto/sr25519"
)

var keyTypes map[string]func() (crypto.PrivKey, error)

func init() {
	keyTypes = map[string]func() (crypto.PrivKey, error){
		ed25519.KeyType: func() (crypto.PrivKey, error) { //nolint: unparam
			return ed25519.GenPrivKey(), nil
		},
		secp256k1.KeyType: func() (crypto.PrivKey, error) { //nolint: unparam
			return secp256k1.GenPrivKey(), nil
		},
		sr25519.KeyType: func() (crypto.PrivKey, error) { //nolint: unparam
			return sr25519.GenPrivKey(), nil
		},
		bls12381.KeyType: func() (crypto.PrivKey, error) {
			pk, err := bls12381.GenPrivKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate BLS key: %w", err)
			}
			return pk, nil
		},
	}
}

func GenPrivKey(keyType string) (crypto.PrivKey, error) {
	genF, ok := keyTypes[keyType]
	if !ok {
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
	return genF()
}

func ListSupportedKeyTypes() string {
	keyTypesS := make([]string, 0, len(keyTypes))
	for k := range keyTypes {
		keyTypesS = append(keyTypesS, fmt.Sprintf("%q", k))
	}
	keyTypes := strings.Join(keyTypesS, ", ")
	return keyTypes
}
