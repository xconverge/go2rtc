package reolink

import (
	"github.com/AlexxIT/go2rtc/internal/streams"
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/reolink"
)

func Init() {
	streams.HandleFunc("reolink", func(source string) (core.Producer, error) {
		return reolink.Dial(source)
	})

}
