package helper

import (
    "bytes"
    "compress/gzip"
    "encoding/base64"
    "io"
)

// compresses the given data
func GzipCompress(data []byte, level int) ([]byte, error) {
    var buf bytes.Buffer
    zw, err := gzip.NewWriterLevel(&buf, level)
    if err != nil {
        return nil, err
    }
    if _, err := zw.Write(data); err != nil {
        _ = zw.Close()
        return nil, err
    }
    if err := zw.Close(); err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}

// decompresses gzip-compressed bytes
func GzipDecompress(data []byte) ([]byte, error) {
    zr, err := gzip.NewReader(bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    defer func() { _ = zr.Close() }()
    return io.ReadAll(zr)
}

// returns base64-encoded string of data
func Base64Encode(data []byte) string {
    return base64.StdEncoding.EncodeToString(data)
}

// decodes base64 string to bytes
func Base64Decode(s string) ([]byte, error) {
    return base64.StdEncoding.DecodeString(s)
}

// gzip + base64. returns b64 string
func GzipAndBase64(data []byte, level int) (string, error) {
    compressed, err := GzipCompress(data, level)
    if err != nil {
        return "", err
    }
    return Base64Encode(compressed), nil
}

// gzip bestcompression + base64. returns b64 string
func GzipAndBase64Best(data []byte) (string, error) {
    return GzipAndBase64(data, gzip.BestCompression)
}

// base64 decode + gunzip if compressed
func Base64DecodeAndGunzipIf(b64 string, compressed bool) ([]byte, error) {
    decoded, err := Base64Decode(b64)
    if err != nil {
        return nil, err
    }
    if compressed {
        out, err := GzipDecompress(decoded)
        if err != nil {
            return nil, err
        }
        return out, nil
    }
    return decoded, nil
}

