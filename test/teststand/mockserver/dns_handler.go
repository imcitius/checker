package main

import (
	"log"
	"net"

	"github.com/miekg/dns"
)

func startDNS(addr string) {
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true

		for _, q := range r.Question {
			switch q.Name {
			case "pass.test.":
				if q.Qtype == dns.TypeA {
					m.Answer = append(m.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
						A:   net.ParseIP("1.2.3.4"),
					})
				}
			case "fail.test.":
				m.SetRcode(r, dns.RcodeNameError)
			default:
				m.SetRcode(r, dns.RcodeNameError)
			}
		}

		w.WriteMsg(m)
	})

	server := &dns.Server{Addr: addr, Net: "udp", Handler: handler}
	log.Printf("DNS listening on %s (UDP)", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("DNS server failed: %v", err)
	}
}
