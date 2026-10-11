package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	MaxImageUploadSize = 15 * 1024 * 1024 // 15MB (Section 21)
	MaxVideoUploadSize = 30 * 1024 * 1024 // 30MB
)

var (
	ErrInvalidFileType = errors.New("unsupported or malicious media file type")
	ErrFileSizeExceeded = errors.New("file size exceeds maximum allowed upload limit")
	ErrSSRFForbidden    = errors.New("forbidden target address: SSRF protection blocked request to private, loopback, or cloud metadata IP")
	ErrVideoNotAllowed  = errors.New("video uploads are not permitted for this tool endpoint")
	ErrBase64Video      = errors.New("video uploads via raw base64 are disabled; provide an external URL or direct file upload")
)

// ValidateMagicBytes verifies actual file header against expected image/video formats (Section 30).
func ValidateMagicBytes(header []byte) (string, error) {
	if len(header) < 4 {
		return "", ErrInvalidFileType
	}

	// JPEG: FF D8 FF
	if len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return "image/jpeg", nil
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if bytes.HasPrefix(header, pngMagic) {
		return "image/png", nil
	}

	// WebP: RIFF .... WEBP
	if len(header) >= 12 && string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP" {
		return "image/webp", nil
	}

	// GIF: GIF87a or GIF89a
	if len(header) >= 6 && (string(header[0:6]) == "GIF87a" || string(header[0:6]) == "GIF89a") {
		return "image/gif", nil
	}

	// MP4: .... ftyp
	if len(header) >= 8 && string(header[4:8]) == "ftyp" {
		return "video/mp4", nil
	}

	// WebM: 1A 45 DF A3
	webmMagic := []byte{0x1A, 0x45, 0xDF, 0xA3}
	if bytes.HasPrefix(header, webmMagic) {
		return "video/webm", nil
	}

	return "", ErrInvalidFileType
}

// ValidateMediaUpload checks file size and verifies magic bytes header.
func ValidateMediaUpload(data []byte, filename string, isVideoAllowed bool) (string, error) {
	if len(data) == 0 {
		return "", errors.New("empty file payload")
	}

	headerLen := len(data)
	if headerLen > 32 {
		headerLen = 32
	}
	mime, err := ValidateMagicBytes(data[:headerLen])
	if err != nil {
		return "", err
	}

	isVideo := strings.HasPrefix(mime, "video/")
	if isVideo {
		if !isVideoAllowed {
			return "", ErrVideoNotAllowed
		}
		if int64(len(data)) > MaxVideoUploadSize {
			return "", ErrFileSizeExceeded
		}
	} else {
		if int64(len(data)) > MaxImageUploadSize {
			return "", ErrFileSizeExceeded
		}
	}

	return mime, nil
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

	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" ||
		lowerHost == "127.0.0.1" ||
		lowerHost == "::1" ||
		lowerHost == "metadata.google.internal" ||
		lowerHost == "instance-data" {
		return ErrSSRFForbidden
	}

	// Check dangerous internal ports if port is explicitly provided
	if portStr := parsed.Port(); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			if isBlockedInternalPort(port) {
				return ErrSSRFForbidden
			}
		}
	}

	// Check if hostname is direct IP literal
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateOrLoopbackIP(ip) {
			return ErrSSRFForbidden
		}
		return nil
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

func isBlockedInternalPort(port int) bool {
	// Block standard internal daemon/database ports
	switch port {
	case 22, 23, 25, 111, 2375, 2376, 3306, 5432, 6379, 8000, 8080, 9000, 9200, 27017:
		return true
	default:
		return false
	}
}

func isPrivateOrLoopbackIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Normalize IPv4-mapped IPv6 addresses (e.g. ::ffff:127.0.0.1 -> 127.0.0.1)
	if ipv4 := ip.To4(); ipv4 != nil {
		if ipv4.IsLoopback() || ipv4.IsUnspecified() || ipv4.IsLinkLocalUnicast() || ipv4.IsLinkLocalMulticast() {
			return true
		}

		// Cloud Metadata IPs:
		// AWS / GCP / Azure: 169.254.169.254
		// AWS ECS: 169.254.170.2
		// Alibaba Cloud: 100.100.100.200
		if ipv4.Equal(net.ParseIP("169.254.169.254")) ||
			ipv4.Equal(net.ParseIP("169.254.170.2")) ||
			ipv4.Equal(net.ParseIP("100.100.100.200")) {
			return true
		}

		// RFC 1918 Private Blocks
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

		// Carrier-Grade NAT (RFC 6598): 100.64.0.0/10 (100.64.0.0 - 100.127.255.255)
		if ipv4[0] == 100 && (ipv4[1]&0xC0) == 64 {
			return true
		}

		// Loopback block (127.0.0.0/8)
		if ipv4[0] == 127 {
			return true
		}

		return false
	}

	// IPv6 Checks
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// AWS IPv6 metadata: fd00:ec2::254
	if ip.Equal(net.ParseIP("fd00:ec2::254")) {
		return true
	}

	// Unique Local Address (fc00::/7)
	if len(ip) == 16 && (ip[0]&0xFE) == 0xFC {
		return true
	}

	return false
}

// SafeHTTPClient creates an HTTP client fortified against SSRF, DNS rebinding, and redirect hijacking (Section 30).
func SafeHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// Validate IP literal directly
			if ip := net.ParseIP(host); ip != nil {
				if isPrivateOrLoopbackIP(ip) {
					return nil, ErrSSRFForbidden
				}
			} else {
				// Mitigate DNS rebinding: resolve right at dial time and verify every address
				ips, err := net.LookupIP(host)
				if err != nil {
					return nil, fmt.Errorf("failed resolving target during dial: %w", err)
				}
				for _, ip := range ips {
					if isPrivateOrLoopbackIP(ip) {
						return nil, ErrSSRFForbidden
					}
				}
			}

			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		},
		ResponseHeaderTimeout: 15 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			// Enforce SSRF validation on every redirect hop
			return ValidateExternalURL(req.URL.String())
		},
	}
}
