package kmgQrcode

import (
	"image/png"
	"sync"
)

type pngEncoderPool struct {
	pool sync.Pool
	//b *png.EncoderBuffer
}

func (p *pngEncoderPool) Get() *png.EncoderBuffer {
	obj := p.pool.Get()
	if obj == nil {
		return nil
	}
	return obj.(*png.EncoderBuffer)
	//return p.b
}

func (p *pngEncoderPool) Put(b *png.EncoderBuffer) {
	p.pool.Put(b)
}
