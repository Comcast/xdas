// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package conversion

import (
	"errors"
	"fmt"
	"strconv"
	"xdas/internal/magicbyte"
	"xdas/internal/rediscrypto"

	"github.com/klauspost/compress/zstd"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	defaultMetrics metricsProvider

	zstdDec *zstd.Decoder
	zstdEnc *zstd.Encoder

	ErrUnknownKeyspace     = errors.New("unknown keyspace definition")
	ErrUnknownEncodingType = errors.New("unkown encoding type")
	ErrUnknownContentType  = errors.New("unknown content-type")
)

// pbMessage holds the appropriate data structure for the keyspace
var pbMessage = map[string]func() proto.Message{}

func init() {
	defaultMetrics = &noMetrics{}
	zstdDec, _ = zstd.NewReader(nil)
	zstdEnc, _ = zstd.NewWriter(nil, zstd.WithZeroFrames(true))
}

// Init sets the proper metricsProvider
func Init(promReg prometheus.Registerer, promNamespace string, keyspaces []string) {
	fmt.Println(keyspaces)
	if promReg != nil {
		defaultMetrics = &prometheusMetrics{
			PromReg:       promReg,
			PromNamespace: promNamespace,
			Keyspaces:     keyspaces,
		}
	}
	err := defaultMetrics.init()
	if err != nil {
		defaultMetrics = &noMetrics{}
	}
}

// Convert returns data based on outMagicByte
func Convert(keyspace string, inMagicByte, outMagicByte magicbyte.MagicByte, inData []byte) (magicbyte.MagicByte, []byte, error) {
	if outMagicByte.GetCTV() == 0 {
		outMagicByte = magicbyte.NewMagicByte(outMagicByte.GetCEV(), inMagicByte.GetCTV(),
			outMagicByte.GetEncryption())
	}

	if inMagicByte.GetCTV() != outMagicByte.GetCTV() && outMagicByte.GetCTV() != 0 {
		// decrypt -> decompress -> unmarshal -> marshal -> compress -> encrypt
		pb, err := Unpack(keyspace, inMagicByte, inData)
		if err != nil {
			defaultMetrics.incContentTypeFail(keyspace)
			return inMagicByte, inData, err
		}
		data, err := Pack(outMagicByte, pb)
		if err != nil {
			defaultMetrics.incContentTypeFail(keyspace)
			return inMagicByte, inData, err
		}
		defaultMetrics.incContentTypeSuc(keyspace)
		return outMagicByte, data, err
	}
	if inMagicByte.GetCEV() != outMagicByte.GetCEV() {
		// decrypt -> decompress -> compress -> encrypt
		data, err := Decrypt(inMagicByte.GetEncryption(), inData)
		if err != nil {
			defaultMetrics.incContentEncodingFail(keyspace)
			return inMagicByte, inData, err
		}
		data, err = Decompress(inMagicByte.GetCEV(), data)
		if err != nil {
			defaultMetrics.incContentEncodingFail(keyspace)
			return inMagicByte, inData, err
		}
		data, err = Compress(outMagicByte.GetCEV(), data)
		if err != nil {
			defaultMetrics.incContentEncodingFail(keyspace)
			return inMagicByte, inData, err
		}
		data, err = Encrypt(outMagicByte.GetEncryption(), data)
		if err != nil {
			defaultMetrics.incContentEncodingFail(keyspace)
			return inMagicByte, inData, err
		}
		defaultMetrics.incContentEncodingSuc(keyspace)
		return outMagicByte, data, err
	}
	if inMagicByte.GetEncryption() != outMagicByte.GetEncryption() {
		// decrypt -> encrypt
		data, err := Decrypt(inMagicByte.GetEncryption(), inData)
		if err != nil {
			defaultMetrics.incEncryptionFail(keyspace)
			return inMagicByte, inData, err
		}
		data, err = Encrypt(outMagicByte.GetEncryption(), data)
		if err != nil {
			defaultMetrics.incEncryptionFail(keyspace)
			return inMagicByte, inData, err
		}
		defaultMetrics.incEncryptionSuc(keyspace)
		return outMagicByte, data, err
	}
	return inMagicByte, inData, nil
}

// Unpack will Decrypt, Decompress and Unmarshal the inData based on inMagicByte and returns a Message
func Unpack(keyspace string, inMagicByte magicbyte.MagicByte, inData []byte) (proto.Message, error) {
	if newPb, ok := pbMessage[keyspace]; ok {
		return UnPackByPB(newPb(), inMagicByte, inData)
	}
	return nil, ErrUnknownKeyspace
}

// UnPackByPB will Decrypt, Decompress and Unmarshal the inData based on inMagicByte and returns a Message
func UnPackByPB(pb proto.Message, inMagicByte magicbyte.MagicByte, inData []byte) (proto.Message, error) {
	data, err := Decrypt(inMagicByte.GetEncryption(), inData)
	if err != nil {
		return nil, err
	}
	data, err = Decompress(inMagicByte.GetCEV(), data)
	if err != nil {
		return nil, err
	}
	err = Unmarshal(inMagicByte.GetCTV(), pb, data)
	return pb, err
}

// Pack will Marshal, Compress and Encrypt the Message based on outMagicByte and returns data
func Pack(outMagicByte magicbyte.MagicByte, pb proto.Message) ([]byte, error) {
	data, err := Marshal(outMagicByte.GetCTV(), pb)
	if err != nil {
		return data, err
	}
	data, err = Compress(outMagicByte.GetCEV(), data)
	if err != nil {
		return data, err
	}
	return Encrypt(outMagicByte.GetEncryption(), data)
}

// Decrypt will return the decrypted data based on the encryption format
func Decrypt(encryption int, inData []byte) ([]byte, error) {
	switch encryption {
	case 0:
		return inData, nil
	case 1:
		return rediscrypto.Decrypt(inData)
	default:
		return inData, errors.New("unknown decryption type " + strconv.Itoa(encryption))
	}
}

// Encrypt will return the encrypted data based on the encryption format
func Encrypt(encryption int, inData []byte) ([]byte, error) {
	switch encryption {
	case 0:
		return inData, nil
	case 1:
		return rediscrypto.Encrypt(inData)
	default:
		return inData, errors.New("unknown encryption type " + strconv.Itoa(encryption))
	}
}

// Decompress will return the decompressed data based on the contentEncodingValue
func Decompress(contentEncodingValue int, inData []byte) ([]byte, error) {
	switch contentEncodingValue {
	case magicbyte.ContentEncodingNone:
		return inData, nil
	case magicbyte.ContentEncodingZstd:
		return zstdDec.DecodeAll(inData, nil)
	case magicbyte.ContentEncodingZlib:
		// to be implemented in the future
		return inData, ErrUnknownEncodingType
	default:
		return inData, ErrUnknownEncodingType
	}
}

// Compress will return the compressed data based on the contentEncodingValue
func Compress(contentEncodingValue int, inData []byte) ([]byte, error) {
	switch contentEncodingValue {
	case magicbyte.ContentEncodingNone:
		return inData, nil
	case magicbyte.ContentEncodingZstd:
		return zstdEnc.EncodeAll(inData, nil), nil
	case magicbyte.ContentEncodingZlib:
		// to be implemented in the future
		return inData, ErrUnknownEncodingType
	default:
		return inData, ErrUnknownEncodingType
	}
}

// Unmarshal will unmarshal to pb based on the inType
func Unmarshal(inType int, pb proto.Message, inData []byte) error {
	switch inType {
	case magicbyte.ContentTypeProtoBuf:
		return proto.Unmarshal(inData, pb)
	case magicbyte.ContentTypeJson:
		return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(inData, pb)
		// return jsonpb.Unmarshal(bytes.NewReader(inData), pb)
	default:
		return ErrUnknownContentType
	}
}

// Marshal will marshal to JSON or Protobuf based on outType
func Marshal(outType int, pb proto.Message) ([]byte, error) {
	switch outType {
	case magicbyte.ContentTypeProtoBuf:
		return proto.Marshal(pb)
	case magicbyte.ContentTypeJson:
		pjm := protojson.MarshalOptions{UseProtoNames: false}
		result, err := pjm.Marshal(pb)
		if err != nil {
			return nil, err
		}
		return result, err
	default:
		return nil, ErrUnknownContentType
	}
}
