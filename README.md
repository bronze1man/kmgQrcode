Description
===================
[![GoDoc](https://godoc.org/github.com/bronze1man/kmgQrcode?status.svg)](https://godoc.org/github.com/bronze1man/kmgQrcode)
[![GitHub issues](https://img.shields.io/github/issues/bronze1man/kmgQrcode.svg)](https://github.com/bronze1man/kmgQrcode/issues)
[![GitHub stars](https://img.shields.io/github/stars/bronze1man/kmgQrcode.svg)](https://github.com/bronze1man/kmgQrcode/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/bronze1man/kmgQrcode.svg)](https://github.com/bronze1man/kmgQrcode/network)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://github.com/bronze1man/kmgQrcode/blob/master/LICENSE)

a golang qrcode writer

A QR Code is a matrix (two-dimensional) barcode. Arbitrary content may be
encoded.

A QR Code contains error recovery information to aid reading damaged or
obscured codes. There are four levels of error recovery: qrcode.{Low, Medium,
High, Highest}. QR Codes with a higher recovery level are more robust to damage,
at the cost of being physically larger.

The maximum capacity of a QR Code varies according to the content encoded and
the error recovery level. The maximum capacity is 2,953 bytes, 4,296
alphanumeric characters, 7,089 numeric digits, or a combination of these.

This package implements a subset of QR Code 2005, as defined in ISO/IEC
18004:2006.

Example
===================
```golang
package main

import (
	"github.com/bronze1man/kmgQrcode"
	"io/ioutil"
)

func main(){
	resp:= kmgQrcode.MustEncode(kmgQrcode.EncodeReq{
		Content: "https://www.google.com/",
	})
	err:=ioutil.WriteFile("tmp.png",resp.PngContent,0777)
	if err!=nil{
		panic(err)
	}
}
```

Notice
===================
* It can not read qrcode into text, it can only translate qrcode text into png byte slice.
* fork from https://github.com/skip2/go-qrcode , with following change:
    * Make the package name easy to remember and import.
    * Simplify the caller interface, leave only one function `kmgQrcode.MustEncode` .
    * Delete surrounding blank, make the qrcode part as large as possible.(use float64 instead of int to generate image.)