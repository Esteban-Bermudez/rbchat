package network

import "encoding/hex"

var xorKey = []byte{130, 251, 47, 182, 93, 216, 38, 171, 14, 201, 65, 156, 77, 144, 8, 189}

func xorBytes(data []byte) []byte {
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = b ^ xorKey[i%len(xorKey)]
	}
	return result
}

func DecodeObfuscatedHex(hexStr string) (string, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}
	return string(xorBytes(data)), nil
}
