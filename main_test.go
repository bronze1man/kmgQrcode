package kmgQrcode

import (
	"testing"
	"image"
	"bytes"
)

func TestQrcode(ot *testing.T){
	resp:= MustEncode(EncodeReq{
		Content: "https://www.google.com/",
	})
	if (len(resp.PngContent)>0)==false{
		panic("fail")
	}
	img,typ,err:=image.Decode(bytes.NewReader(resp.PngContent))
	if err!=nil{
		panic(err)
	}
	if img.Bounds().Dx()!=256{
		panic("fail")
	}
	if img.Bounds().Dy()!=256{
		panic("fail")
	}
	if typ!="png"{
		panic("png")
	}
}