// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package keyspaces

import "fmt"

const (
	KSString Kind = iota // string based keyspace
	KSDM                 //
	KSAtomic             // atomic keyspace
	KSHashes             // hashes keyspace
)

type Kind int

func (k Kind) String() string {
	return []string{
		"string",
		"dm",
		"atomic",
		"hashes",
	}[int(k)]
}

func (k *Kind) UnmarshalText(text []byte) error {
	kinds := map[string]Kind{
		"string": KSString,
		"dm":     KSDM,
		"atomic": KSAtomic,
		"hashes": KSHashes,
	}
	kind, ok := kinds[string(text)]
	if !ok {
		return fmt.Errorf("cannot parse %q as a keyspace type", string(text))
	}
	*k = kind
	return nil

}

func (k Kind) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}
