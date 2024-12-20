package aead

import (
	"crypto/hmac"
	"crypto/sha256"
	"hash"
)

func KDF(key []byte, path ...string) []byte {
	hmacCreator := &hMacCreator{value: []byte(KDFSaltConstVMessAEADKDF)}
	for _, v := range path {
		hmacCreator = &hMacCreator{value: []byte(v), parent: hmacCreator}
	}
	hmacf := hmacCreator.Create()
	hmacf.Write(key)
	return hmacf.Sum(nil)
}

type hMacCreator struct {
	parent *hMacCreator
	value  []byte
}
// 最终返回一个新的 HMAC 对象，该对象使用指定的底层哈希算法和密钥进行初始化。
func (h *hMacCreator) Create() hash.Hash {
	if h.parent == nil {
		return hmac.New(sha256.New, h.value)
	}
	return hmac.New(h.parent.Create, h.value)
}

func KDF16(key []byte, path ...string) []byte {
	r := KDF(key, path...)
	return r[:16]
}
