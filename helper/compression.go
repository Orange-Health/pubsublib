package helper

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
)

/*
	Expects a string and returns a base64 encoded string of the compressed string.
	Approximately reduces the size of the string by atleast 90%
*/
func CompressString(stringToCompress string) (string, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(stringToCompress)); err != nil {
		return "", err
	}
	if err := gz.Flush(); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

/*
	Expects a compressed string and returns the decompressed string.
*/
func DecompressString(stringToDecompress string) (string, error) {
	decodedString, err := base64.StdEncoding.DecodeString(stringToDecompress)
	if err != nil {
		return "", err
	}
	rdata := bytes.NewReader(decodedString)
	r, _ := gzip.NewReader(rdata)
	s, _ := ioutil.ReadAll(r)

	return string(s), nil
}
