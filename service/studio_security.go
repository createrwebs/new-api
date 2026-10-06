package service

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	ErrInvalidFileType = errors.New("unsupported or malicious media file type")
	ErrFileSizeExceeded = errors.New("file size exceeds maximum allowed upload limit")
	ErrSSRFForbidden    = errors.New("forbidden target address: SSRF protection blocked request to private or loopback IP")
)

// ValidateMagicBytes verifies actual file header against expected image/video formats (Section 30).
func ValidateMagicBytes(header []byte) (string, error) {
	if len(header) < 12 {
		return "", ErrInvalidFileType
	}

	// JPEG: FF D8 FF
	if header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return "image/jpeg", nil
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if bytes.HasPrefix(header, pngMagic) {
		return "image/png", nil
	}

	// WebP: RIFF .... WEBP
	if string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP" {
		return "image/webp", nil
	}

	// MP4: .... ftyp
	if len(header) >= 8 && string(header[4:8]) == "ftyp" {
		return "video/mp4", nil
	}

	return "", ErrInvalidFileType
}

// ValidateExternalURL inspects target URLs to prevent SSRF against internal resources (Section 30).
func ValidateExternalURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.New("only http and https protocols are allowed")
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return errors.New("missing hostname in target URL")
	}

	// Block standard loopback and metadata names
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" || lowerHost == "metadata.google.internal" {
		return ErrSSRFForbidden
	}

	// Resolve hostname to IP addresses
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed resolving target IP: %w", err)
	}

	for _, ip := range ips {
		if isPrivateOrLoopbackIP(ip) {
			return ErrSSRFForbidden
		}
	}

	return nil
}

func isPrivateOrLoopbackIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// AWS / GCP Metadata IP (169.254.169.254)
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}

	// IPv4 Private Blocks (RFC 1918)
	if ipv4 := ip.To4(); ipv4 != nil {
		// 10.0.0.0/8
		if ipv4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ipv4[0] == 192 && ipv4[1] == 168 {
			return true
		}
	}

	return false
}
