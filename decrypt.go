package hlsdl

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/grafov/m3u8"
)

// map URLs to keys (byte arrays)
var cachedKeys map[string][]byte = make(map[string][]byte)
var cachedKeysMut sync.Mutex

func getCachedKey(url string) (key []byte, ok bool) {
	cachedKeysMut.Lock()
	key, ok = cachedKeys[url]
	cachedKeysMut.Unlock()
	return
}

func putCachedKey(url string, key []byte) {
	cachedKeysMut.Lock()
	cachedKeys[url] = key
	cachedKeysMut.Unlock()
}

func ClearCachedKeys() {
	cachedKeysMut.Lock()
	cachedKeys = make(map[string][]byte)
	cachedKeysMut.Unlock()
}

func (client *Client) getKey(m3u8url, keyURL string) (key []byte, err error) {
	url, err := absURL(m3u8url, keyURL)
	if err != nil {
		return
	}
	key, ok := getCachedKey(url)
	if ok {
		return
	}
	res, err := client.doRequest(url)
	if err != nil {
		return
	}
	key, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	putCachedKey(url, key)
	return
}

func (client *Client) decryptSegment(
	m3u8url string,
	seg *m3u8.MediaSegment,
	input []byte,
) (output []byte, err error) {
	if seg.Key == nil {
		output = input
		return
	}
	key, err := client.getKey(m3u8url, seg.Key.URI)
	if err != nil {
		return
	}
	var iv []byte
	if len(seg.Key.IV) == 0 {
		iv = ivFromSeqNo(seg.SeqId)
	} else {
		iv, err = ivFromHexString(seg.Key.IV)
		if err != nil {
			return
		}
	}
	output, err = aes128Decrypt(input, key, iv)
	if err != nil {
		return
	}
	// discard all data before the sync byte
	syncByte := byte(0x47)
	for i := 0; i < len(output); i++ {
		if output[i] == syncByte {
			output = output[i:]
			break
		}
	}
	return
}

func ivFromHexString(s string) (iv []byte, err error) {
	s = strings.ToLower(s)
	if s[:2] == "0x" {
		s = s[2:]
	}
	iv, err = hex.DecodeString(s)
	return
}

func ivFromSeqNo(seqNo uint64) []byte {
	// put big-endian binary representation into a 16-byte buffer,
	// padding on the left with zeros
	b := make([]byte, 16)
	for i := 0; i < 8; i++ {
		b[16-i-1] = byte(seqNo >> (i * 8))
	}
	return b
}

func aes128Decrypt(encrypted, key, iv []byte) ([]byte, error) {
	if len(key) != 16 || len(iv) != 16 {
		return nil, errors.New("key and IV must both be 128 bits")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	blockMode.CryptBlocks(decrypted, encrypted)
	decrypted = pkcs7unpad(decrypted)
	return decrypted, nil
}

func pkcs7unpad(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}

func adjustKeys(mediaPlaylist *m3u8.MediaPlaylist) (err error) {
	key := mediaPlaylist.Key
	if err = checkEncryptionMethod(key); err != nil {
		return err
	}
	for _, seg := range mediaPlaylist.Segments {
		// RFC 8216, Section 4.3.2.4:
		// The EXT-X-KEY tag "applies to very Media Segment...that appears
		// between it and the next EXT-X-KEY tag"
		if seg.Key == nil {
			seg.Key = key
		} else {
			key = seg.Key
			if err = checkEncryptionMethod(key); err != nil {
				return err
			}
		}
	}
	return
}

func checkEncryptionMethod(key *m3u8.Key) error {
	if key != nil && key.Method == "SAMPLE-AES" {
		return errors.New("SAMPLE-AES is not currently supported")
	}
	return nil
}
