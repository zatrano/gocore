// Package safehttp, SSRF (Server-Side Request Forgery) korumalı bir HTTP client
// sağlar. Kullanıcı kontrollü URL'lere yapılan dış çağrılarda kullanılmalıdır.
package safehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ErrBlockedAddress, özel/dahili bir adrese bağlantı engellendiğinde döner.
var ErrBlockedAddress = errors.New("safehttp: dahili/özel adrese erişim engellendi")

// NewClient, her bağlantıda hedef IP'yi doğrulayan SSRF-korumalı bir
// *http.Client döner. Loopback, private, link-local ve unique-local adreslere
// bağlantıyı reddeder. Böylece kullanıcı "http://169.254.169.254/..." gibi
// metadata/dahili servislere ulaşamaz.
func NewClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			// DNS çöz ve TÜM sonuçları doğrula (DNS rebinding'e karşı).
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			for _, ip := range ips {
				if isBlocked(ip.IP) {
					return nil, fmt.Errorf("%w: %s", ErrBlockedAddress, ip.IP)
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
		// Yönlendirmeler ayrıca CheckRedirect ile kontrol edilir.
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("safehttp: çok fazla yönlendirme")
			}
			return nil
		},
	}
}

// isBlocked, verilen IP'nin dahili/özel olup olmadığını kontrol eder.
func isBlocked(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}
