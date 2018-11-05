package kmgQrcode

import (
	"encoding/base64"
	"image/png"
	"sync"
)

type EncodeReq struct {
	Content string        `json:",omitempty"`
	Level   RecoveryLevel `json:",omitempty"` // 冗余比例，默认 Highest
	PngSize int           `json:",omitempty"` // 输出的png的长宽，会输出一个正方形的图形，所以长宽都一样。 默认 256

	IsOutputDataUrl bool `json:",omitempty"` // 输出Data url,(就不输出 png了）

	MemoryCacher *EncodeV2Cacher `json:"-"` // 在进程内存中缓存生成qrcode的请求和结果。以便下次可以快速使用。
	CacheKey     string          `json:",omitempty"`

	PngUseBufferPool    bool                 `json:",omitempty"` // 是否使用 UsePngBufferPool,这个东西可以减少alloc，但是会产生一个一直不释放的内存占用
	PngCompressionLevel png.CompressionLevel `json:",omitempty"`
}
type EncodeResp struct {
	PngContent []byte
	DataUrl    string
}

func MustEncode(req EncodeReq) (resp EncodeResp) {
	if req.Level == 0 {
		req.Level = Highest
	}
	if req.PngSize == 0 {
		req.PngSize = 256
	}
	if req.PngCompressionLevel == 0 {
		req.PngCompressionLevel = png.DefaultCompression
	}
	if req.MemoryCacher != nil {
		resp, ok := req.MemoryCacher.GetByKey(req.CacheKey)
		if ok == false {
			resp = mustEncodeL1(req)
			req.MemoryCacher.SetByKey(req.CacheKey, resp)
		}
		return resp
	}
	return mustEncodeL1(req)
}

func mustEncodeL1(req EncodeReq) (resp EncodeResp) {
	var q *qRCode

	q, err := newV1(req.Content, req.Level)
	if err != nil {
		panic(err)
	}
	q.req = req
	png, err := q.pNG(req.PngSize)
	//png,err:= encode(req.Content,req.Level,req.PngSize)
	if err != nil {
		panic(err)
	}
	if req.IsOutputDataUrl {
		resp.DataUrl = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	} else {
		resp.PngContent = png
	}
	return resp
}

type EncodeV2Cacher struct {
	m            map[string]EncodeResp
	locker       sync.RWMutex
	MaxCacheSize int // 默认 1000,创建之后不要修改。
}

func (c *EncodeV2Cacher) GetByKey(key string) (resp EncodeResp, ok bool) {
	c.locker.RLock()
	if c.m == nil {
		c.locker.RUnlock()
		return resp, false
	}
	resp, ok = c.m[key]
	c.locker.RUnlock()
	return resp, ok
}

func (c *EncodeV2Cacher) SetByKey(key string, resp EncodeResp) {
	c.locker.Lock()
	if c.MaxCacheSize == 0 {
		c.MaxCacheSize = defaultMaxCacheSize
	}
	if c.m == nil {
		c.m = map[string]EncodeResp{}
	} else if len(c.m) >= c.MaxCacheSize {
		for k := range c.m {
			delete(c.m, k)
		}
	}
	c.m[key] = resp
	c.locker.Unlock()
}

const defaultMaxCacheSize = 1000
