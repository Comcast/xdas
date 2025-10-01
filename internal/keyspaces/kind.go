/*
 * Copyright 2025 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
