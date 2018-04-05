package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/miekg/dns"
)

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {

	defer w.Close()

	Logger, _ := initLogger()

	m := new(dns.Msg)
	fmt.Println("handleRequest:inbound message:")
	fmt.Printf("%+v", r)

	for _, q := range r.Question {
		name := q.Name

		var host inventory

		_, err := bucket.Get(name[:len(name)-1], &host)

		if err != nil {
			totalRequestDns.WithLabelValues(strconv.Itoa(dns.RcodeNameError)).Inc()
			Logger.Errorf("DNS: Failed get: %s", name[:len(name)-1])
			fmt.Fprint(w, "DNS: Failed get: %s \n", name[:len(name)-1])
			m.SetReply(r)
			fmt.Println(m.Answer)
			w.WriteMsg(m)
			return
		} else {
			totalRequestDns.WithLabelValues(strconv.Itoa(dns.RcodeSuccess)).Inc()
		}
		answer := new(dns.A)
		answer.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: Config.Server.DNS.TTL}
		answer.A = net.ParseIP(host.IP)
		m.Answer = append(m.Answer, answer)
	}
	m.SetReply(r)
	fmt.Printf("%+v\n", m)
	w.WriteMsg(m)
}
