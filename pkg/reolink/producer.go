package reolink

import (
	"fmt"
	"net/url"

	"github.com/AlexxIT/go2rtc/pkg/aac"
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/h264"
	"github.com/AlexxIT/go2rtc/pkg/h264/annexb"
	"github.com/pion/rtp"
)

type Producer struct {
	core.Connection

	bcConn       *BCConn
	streamReader *BCStreamReader
}

func Dial(source string) (core.Producer, error) {
	// fmt.Println(source)
	sourceUrl, err := url.Parse(source)
	if err != nil {
		return nil, err
	}

	ip := sourceUrl.Hostname()
	port := sourceUrl.Port()
	username := sourceUrl.User.Username()
	password, ok := sourceUrl.User.Password()
	if !ok {
		return nil, fmt.Errorf("password is required")
	}

	bcConn := NewBCConn(ip, port, username, password)

	prod := &Producer{
		Connection: core.Connection{
			ID:         core.NewID(),
			FormatName: "reolink",
			Protocol:   "tcp", // wss
			RemoteAddr: ip,
			Source:     source,
			URL:        ip,
			Transport:  bcConn,
		},
		bcConn: bcConn,
	}
	if err = prod.probe(); err != nil {
		return nil, err
	}

	return prod, nil
}

func (p *Producer) Start() error {
	var video *core.Receiver
	for _, r := range p.Receivers {
		if r.Codec.Name == core.CodecH264 {
			video = r
		}
	}

	for {
		packet := p.streamReader.Next()
		if packet == nil || packet.Codec == "EOF" {
			return nil
		}
		switch packet.Codec {
		case "H264":
			if video == nil {
				continue
			}
			pkt := &rtp.Packet{
				Header: rtp.Header{Timestamp: core.Now90000()},
				Payload: annexb.EncodeToAVCC(packet.Data),
			}
			video.Input(pkt)
			// case "AAC": handle when audio receiver exists
		}
	}
}

func (p *Producer) Stop() error {
	if p.bcConn != nil {
		_ = p.bcConn.Close()
	}
	p.streamReader = nil
	return nil
}

func (p *Producer) probe() error {
	var packets int

	p.streamReader = p.bcConn.startStream()

	for packets != 10 {
		packet := p.streamReader.Next()
		switch packet.Codec {
		case "H264":
			p.Medias = append(p.Medias, &core.Media{
				Kind:      core.KindVideo,
				Direction: core.DirectionRecvonly,
				Codecs: []*core.Codec{
					{
						Name:        core.CodecH264,
						ClockRate:   90000,
						PayloadType: core.PayloadTypeRAW,
						FmtpLine:    h264.GetFmtpLine(annexb.EncodeToAVCC(packet.Data)),
					},
				},
			})
		case "AAC":
			codec := aac.ConfigToCodec(packet.Data)
			p.Medias = append(p.Medias, &core.Media{
				Kind:      core.KindAudio,
				Direction: core.DirectionRecvonly,
				Codecs:    []*core.Codec{codec},
			})
		}
		packets++
	}

	return nil
}
