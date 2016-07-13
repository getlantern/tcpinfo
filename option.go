// Copyright 2016 Mikio Hara. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tcpinfo

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/mikioh/tcpopt"
)

var (
	errOpNoSupport    = errors.New("operation not supported")
	errBufferTooShort = errors.New("buffer too short")

	_ json.Marshaler = &Info{}
	_ tcpopt.Option  = &Info{}
)

// A State represents a state of connection.
type State int

const (
	Unknown State = iota
	Closed
	Listen
	SynSent
	SynReceived
	Established
	FinWait1
	FinWait2
	CloseWait
	LastAck
	Closing
	TimeWait
)

var states = map[State]string{
	Unknown:     "unknown",
	Closed:      "closed",
	Listen:      "listen",
	SynSent:     "syn-sent",
	SynReceived: "syn-received",
	Established: "established",
	FinWait1:    "fin-wait-1",
	FinWait2:    "fin-wait-2",
	CloseWait:   "close-wait",
	LastAck:     "last-ack",
	Closing:     "closing",
	TimeWait:    "time-wait",
}

func (st State) String() string {
	s, ok := states[st]
	if !ok {
		return "<nil>"
	}
	return s
}

// An Info represents connection information.
//
// Only supported on Darwin, FreeBSD, Linux and NetBSD.
type Info struct {
	State             State              `json:"state"`               // connection state
	Options           []Option           `json:"opts,omitempty"`      // requesting options
	PeerOptions       []Option           `json:"peer_opts,omitempty"` // options requested from peer
	SenderMSS         MaxSegSize         `json:"snd_mss"`             // maximum segment size for sender in bytes
	ReceiverMSS       MaxSegSize         `json:"rcv_mss"`             // maximum sengment size for receiver in bytes
	RTT               time.Duration      `json:"rtt"`                 // round-trip time
	RTTVar            time.Duration      `json:"rttvar"`              // round-trip time variation
	RTO               time.Duration      `json:"rto"`                 // retransmission timeout
	ATO               time.Duration      `json:"ato"`                 // delayed acknowledgement timeout [Linux only]
	LastDataSent      time.Duration      `json:"last_data_sent"`      // since last data sent [Linux only]
	LastDataReceived  time.Duration      `json:"last_data_rcvd"`      // since last data received [FreeBSD and Linux only]
	LastAckReceived   time.Duration      `json:"last_ack_rcvd"`       // since last ack received [Linux only]
	FlowControl       *FlowControl       `json:"flow_ctl,omitempty"`  // flow control information
	CongestionControl *CongestionControl `json:"cong_ctl,omitempty"`  // congestion control information
	Sys               *SysInfo           `json:"sys,omitempty"`       // platform-specific information
}

// A FlowControl represents flow control information.
type FlowControl struct {
	ReceiverWindow uint `json:"rcv_wnd"` // advertised receiver window in bytes
}

// A CongestionControl represents congestion control information.
type CongestionControl struct {
	SenderSSThreshold   uint `json:"snd_ssthresh"` // slow start threshold for sender in bytes or # of segments
	ReceiverSSThreshold uint `json:"rcv_ssthresh"` // slow start threshold for receiver in bytes [Linux only]
	SenderWindow        uint `json:"snd_cwnd"`     // congestion window for sender in bytes or # of segments
}

// Level implements the Level method of tcpopt.Option interface.
func (i *Info) Level() int { return options[soInfo].level }

// Name implements the Name method of tcpopt.Option interface.
func (i *Info) Name() int { return options[soInfo].name }

// MarshalJSON implements the MarshalJSON method of json.Marshaler
// interface.
func (i *Info) MarshalJSON() ([]byte, error) {
	raw := make(map[string]interface{})
	raw["state"] = i.State.String()
	if len(i.Options) > 0 {
		opts := make(map[string]interface{})
		for _, opt := range i.Options {
			opts[opt.Kind().String()] = opt
		}
		raw["opts"] = opts
	}
	if len(i.PeerOptions) > 0 {
		opts := make(map[string]interface{})
		for _, opt := range i.PeerOptions {
			opts[opt.Kind().String()] = opt
		}
		raw["peer_opts"] = opts
	}
	raw["snd_mss"] = i.SenderMSS
	raw["rcv_mss"] = i.ReceiverMSS
	raw["rtt"] = i.RTT
	raw["rttvar"] = i.RTTVar
	raw["rto"] = i.RTO
	raw["ato"] = i.ATO
	raw["last_data_sent"] = i.LastDataSent
	raw["last_data_rcvd"] = i.LastDataReceived
	raw["last_ack_rcvd"] = i.LastAckReceived
	if i.FlowControl != nil {
		raw["flow_ctl"] = i.FlowControl
	}
	if i.CongestionControl != nil {
		raw["cong_ctl"] = i.CongestionControl
	}
	if i.Sys != nil {
		raw["sys"] = i.Sys
	}
	return json.Marshal(&raw)
}

// A CCInfo represents raw information of congestion control
// algorithm.
//
// Only supported on Linux.
type CCInfo struct {
	Raw []byte `json:"raw,omitempty"`
}

// Level implements the Level method of tcpopt.Option interface.
func (cci *CCInfo) Level() int { return options[soCCInfo].level }

// Name implements the Name method of tcpopt.Option interface.
func (cci *CCInfo) Name() int { return options[soCCInfo].name }

// Marshal implements the Marshal method of tcpopt.Option interface.
func (cci *CCInfo) Marshal() ([]byte, error) { return cci.Raw, nil }

func parseCCInfo(b []byte) (tcpopt.Option, error) { return &CCInfo{Raw: b}, nil }

// A CCAlgorithm represents a name of congestion control algorithm.
//
// Only supported on Linux.
type CCAlgorithm string

// Level implements the Level method of tcpopt.Option interface.
func (cca CCAlgorithm) Level() int { return options[soCCAlgo].level }

// Name implements the Name method of tcpopt.Option interface.
func (cca CCAlgorithm) Name() int { return options[soCCAlgo].name }

// Marshal implements the Marshal method of tcpopt.Option interface.
func (cca CCAlgorithm) Marshal() ([]byte, error) {
	if cca == "" {
		return nil, nil
	}
	return []byte(cca), nil
}

func parseCCAlgorithm(b []byte) (tcpopt.Option, error) { return CCAlgorithm(b), nil }

// A CCAlgorithmInfo represents congestion control algorithm
// information.
//
// Only supported on Linux.
type CCAlgorithmInfo interface {
	Algorithm() string
}

// ParseCCAlgorithm parses congestion control algorithm information.
//
// Only supported on Linux.
func ParseCCAlgorithmInfo(name string, b []byte) (CCAlgorithmInfo, error) {
	ccai, err := parseCCAlgorithmInfo(name, b)
	if err != nil {
		return nil, err
	}
	return ccai, nil
}