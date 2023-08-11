/*
Copyright 2023 Avi Zimmerman <avi.zimmerman@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package campfire

import (
	"bytes"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"text/template"

	"github.com/pion/datachannel"
	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3"

	"github.com/webmeshproj/webmesh/pkg/context"
)

var (
	//go:embed zcampfire.crt
	campfireCert []byte
	//go:embed zcampfire.key
	campfireKey []byte
)

// Wait waits for a peer to join the camp fire and dispatches a connections
// and errors to the appropriate channels.
func Wait(ctx context.Context, opts Options) (CampFire, error) {
	certs, _, err := loadCertificate()
	if err != nil {
		return nil, fmt.Errorf("load certificate: %w", err)
	}
	location, err := Find(opts.PSK, opts.TURNServers)
	if err != nil {
		return nil, fmt.Errorf("find campfire: %w", err)
	}
	s := webrtc.SettingEngine{}
	s.DetachDataChannels()
	s.DisableCertificateFingerprintVerification(true)
	s.SetICECredentials(location.RemoteUfrag(), location.RemotePwd())
	s.SetIncludeLoopbackCandidate(true)
	cf := offlineCampFire{
		api:      webrtc.NewAPI(webrtc.WithSettingEngine(s)),
		certs:    certs,
		psk:      string(opts.PSK),
		location: location,
		errc:     make(chan error, 3),
		readyc:   make(chan struct{}),
		acceptc:  make(chan datachannel.ReadWriteCloser, 1),
		closec:   make(chan struct{}),
		log:      context.LoggerFrom(ctx).With("protocol", "campfire"),
	}
	go cf.handlePeerConnections()
	return &cf, nil
}

type offlineCampFire struct {
	api      *webrtc.API
	certs    []webrtc.Certificate
	location *Location
	psk      string
	errc     chan error
	readyc   chan struct{}
	acceptc  chan datachannel.ReadWriteCloser
	closec   chan struct{}
	log      *slog.Logger
}

func (o *offlineCampFire) handlePeerConnections() {
	host, err := net.ResolveUDPAddr("udp", strings.TrimPrefix(o.location.TURNServer, "turn:"))
	if err != nil {
		o.errc <- fmt.Errorf("split host port: %w", err)
		return
	}
	var remoteDescription bytes.Buffer
	turnAddr := host.AddrPort().Addr().String()
	err = waiterRemoteTemplate.Execute(&remoteDescription, map[string]any{
		"SessionID":  o.location.SessionID(),
		"Username":   o.location.LocalUfrag(),
		"Secret":     o.location.LocalPwd(),
		"TURNServer": turnAddr,
		"TURNPort":   host.Port,
	})
	if err != nil {
		o.errc <- fmt.Errorf("execute remote template: %w", err)
		return
	}
	pc, err := o.api.NewPeerConnection(webrtc.Configuration{
		Certificates:       o.certs,
		ICETransportPolicy: webrtc.ICETransportPolicyRelay,
		ICEServers: []webrtc.ICEServer{
			{
				URLs:       []string{o.location.TURNServer},
				Username:   "-",
				Credential: "-",
			},
		},
	})
	if err != nil {
		o.errc <- fmt.Errorf("new peer connection: %w", err)
		return
	}
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		o.log.Debug("ICE candidate", "candidate", c.String())
	})
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		o.log.Debug("ICE connection state changed", "state", state.String())
		switch state {
		case webrtc.ICEConnectionStateConnected:
			close(o.readyc)
		case webrtc.ICEConnectionStateFailed:
			o.errc <- fmt.Errorf("ice connection failed")
		}
	})
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		o.log.Debug("Data channel opened", "label", dc.Label())
		if dc.Label() != o.psk {
			o.errc <- fmt.Errorf("unexpected data channel label: %q", dc.Label())
			return
		}
		rw, err := dc.Detach()
		if err != nil {
			o.errc <- fmt.Errorf("detach data channel: %w", err)
			return
		}
		o.acceptc <- rw
	})
	err = pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  remoteDescription.String(),
	})
	if err != nil {
		o.errc <- fmt.Errorf("set remote description: %w", err)
		return
	}
	turnCandidate, err := ice.NewCandidateRelay(&ice.CandidateRelayConfig{
		Network: "udp",
		Address: turnAddr,
		Port:    host.Port,
	})
	if err != nil {
		o.errc <- fmt.Errorf("new turn candidate: %w", err)
		return
	}
	err = pc.AddICECandidate(webrtc.ICECandidateInit{
		Candidate: turnCandidate.Marshal(),
	})
	if err != nil {
		o.errc <- fmt.Errorf("add turn candidate: %w", err)
		return
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		o.errc <- fmt.Errorf("create answer: %w", err)
		return
	}
	err = pc.SetLocalDescription(answer)
	if err != nil {
		o.errc <- fmt.Errorf("set local description: %w", err)
		return
	}
}

// Accept returns a connection to a peer.
func (o *offlineCampFire) Accept() (io.ReadWriteCloser, error) {
	select {
	case <-o.closec:
		return nil, ErrClosed
	case <-o.readyc:
	}
	select {
	case <-o.closec:
		return nil, ErrClosed
	case conn := <-o.acceptc:
		return conn, nil
	}
}

// Close closes the camp fire.
func (o *offlineCampFire) Close() error {
	select {
	case <-o.closec:
		return ErrClosed
	default:
		close(o.closec)
	}
	return nil
}

// Errors returns a channel of errors.
func (o *offlineCampFire) Errors() <-chan error {
	return o.errc
}

// Ready returns a channel that is closed when the camp fire is ready.
func (o *offlineCampFire) Ready() <-chan struct{} {
	return o.readyc
}

var (
	offlineX509Cert  *x509.Certificate
	offlineCerts     []webrtc.Certificate
	offlineCertsErr  error
	offlineCertsOnce sync.Once
)

func loadCertificate() ([]webrtc.Certificate, *x509.Certificate, error) {
	offlineCertsOnce.Do(func() {
		certPem, extra := pem.Decode(campfireCert)
		if len(extra) > 0 {
			offlineCertsErr = fmt.Errorf("extra data after certificate")
			return
		}
		if certPem == nil {
			offlineCertsErr = fmt.Errorf("failed to decode certificate")
			return
		}
		keyPem, extra := pem.Decode(campfireKey)
		if len(extra) > 0 {
			offlineCertsErr = fmt.Errorf("extra data after key")
			return
		}
		if keyPem == nil {
			offlineCertsErr = fmt.Errorf("failed to decode key")
			return
		}
		cert, err := x509.ParseCertificate(certPem.Bytes)
		if err != nil {
			offlineCertsErr = fmt.Errorf("parse certificate: %w", err)
			return
		}
		offlineX509Cert = cert
		key, err := x509.ParsePKCS8PrivateKey(keyPem.Bytes)
		if err != nil {
			offlineCertsErr = fmt.Errorf("parse key: %w", err)
			return
		}
		offlineCerts = []webrtc.Certificate{webrtc.CertificateFromX509(key, cert)}
	})
	return offlineCerts, offlineX509Cert, offlineCertsErr
}

var waiterRemoteTemplate = template.Must(template.New("srv-remote-desc").Parse(`v=0
o=- {{ .SessionID }} 2 IN IP4 0.0.0.0
s=-
t=0 0
a=group:BUNDLE 0
a=msid-semantic: WMS
m=application 9 UDP/DTLS/SCTP webrtc-datachannel
c=IN IP4 0.0.0.0
a=ice-ufrag:{{ .Username }}
a=ice-pwd:{{ .Secret }}
a=fingerprint:sha-256 invalidFingerprint
a=setup:actpass
a=mid:0
a=sctp-port:5000
a=candidate:1 1 UDP 99999 {{ .TURNServer }} {{ .TURNPort }} typ relay 127.0.0.1 50000
`))