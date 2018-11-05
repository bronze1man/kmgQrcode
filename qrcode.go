// go-qrcode
// Copyright 2014 Tom Harwood

package kmgQrcode

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"log"

	"github.com/bronze1man/kmgQrcode/kmgQrcodeBitset"
	"github.com/bronze1man/kmgQrcode/kmgQrcodeReedsolomon"
)

// Encode a QR Code and return a raw PNG image.
//
// size is both the image width and height in pixels. If size is too small then
// a larger image is silently returned. Negative values for size cause a
// variable sized image to be returned: See the documentation for Image().
//
// To serve over HTTP, remember to send a Content-Type: image/png header.
func encode(content string, level RecoveryLevel, size int) ([]byte, error) {
	var q *qRCode

	q, err := newV1(content, level)

	if err != nil {
		return nil, err
	}

	return q.pNG(size)
}

// A QRCode represents a valid encoded QRCode.
type qRCode struct {
	req EncodeReq
	// Original content encoded.
	Content string

	// QR Code type.
	Level         RecoveryLevel
	VersionNumber int

	// User settable drawing options.
	ForegroundColor color.Color
	BackgroundColor color.Color

	encoder *dataEncoder
	version qrCodeVersion

	data   *kmgQrcodeBitset.Bitset
	symbol *symbol
	mask   int
}

// New constructs a QRCode.
//
//	var q *qrcode.QRCode
//	q, err := qrcode.New("my content", qrcode.Medium)
//
// An error occurs if the content is too long.
func newV1(content string, level RecoveryLevel) (*qRCode, error) {
	encoders := []dataEncoderType{dataEncoderType1To9, dataEncoderType10To26,
		dataEncoderType27To40}

	var encoder *dataEncoder
	var encoded *kmgQrcodeBitset.Bitset
	var chosenVersion *qrCodeVersion
	var err error

	for _, t := range encoders {
		encoder = newDataEncoder(t)
		encoded, err = encoder.encode([]byte(content))

		if err != nil {
			continue
		}

		chosenVersion = chooseQRCodeVersion(level, encoder, encoded.Len())

		if chosenVersion != nil {
			break
		}
	}

	if err != nil {
		return nil, err
	} else if chosenVersion == nil {
		return nil, errors.New("content too long to encode")
	}

	q := &qRCode{
		Content: content,

		Level:         level,
		VersionNumber: chosenVersion.version,

		ForegroundColor: color.Black,
		BackgroundColor: color.White,

		encoder: encoder,
		data:    encoded,
		version: *chosenVersion,
	}

	q.encode(chosenVersion.numTerminatorBitsRequired(encoded.Len()))

	return q, nil
}

func newWithForcedVersion(content string, version int, level RecoveryLevel) (*qRCode, error) {
	var encoder *dataEncoder

	switch {
	case version >= 1 && version <= 9:
		encoder = newDataEncoder(dataEncoderType1To9)
	case version >= 10 && version <= 26:
		encoder = newDataEncoder(dataEncoderType10To26)
	case version >= 27 && version <= 40:
		encoder = newDataEncoder(dataEncoderType27To40)
	default:
		log.Fatalf("Invalid version %d (expected 1-40 inclusive)", version)
	}

	var encoded *kmgQrcodeBitset.Bitset
	encoded, err := encoder.encode([]byte(content))

	if err != nil {
		return nil, err
	}

	chosenVersion := getQRCodeVersion(level, version)

	if chosenVersion == nil {
		return nil, errors.New("cannot find QR Code version")
	}

	q := &qRCode{
		Content: content,

		Level:         level,
		VersionNumber: chosenVersion.version,

		ForegroundColor: color.Black,
		BackgroundColor: color.White,

		encoder: encoder,
		data:    encoded,
		version: *chosenVersion,
	}

	q.encode(chosenVersion.numTerminatorBitsRequired(encoded.Len()))

	return q, nil
}

// Bitmap returns the QR Code as a 2D array of 1-bit pixels.
//
// bitmap[y][x] is true if the pixel at (x, y) is set.
//
// The bitmap includes the required "quiet zone" around the QR Code to aid
// decoding.
func (q *qRCode) bitmap() [][]bool {
	return q.symbol.bitmap()
}

// Image returns the QR Code as an image.Image.
//
// A positive size sets a fixed image width and height (e.g. 256 yields an
// 256x256px image).
//
// Depending on the amount of data encoded, fixed size images can have different
// amounts of padding (white space around the QR Code). As an alternative, a
// variable sized image can be generated instead:
//
// A negative size causes a variable sized image to be returned. The image
// returned is the minimum size required for the QR Code. Choose a larger
// negative number to increase the scale of the image. e.g. a size of -5 causes
// each module (QR Code "pixel") to be 5px in size.
//func (q *QRCode) Image(size int) image.Image {
//	// Minimum pixels (both width and height) required.
//	realSize := q.symbol.size
//
//	// Variable size support.
//	if size < 0 {
//		size = size * -1 * realSize
//	}
//
//	// Actual pixels available to draw the symbol. Automatically increase the
//	// image size if it's not large enough.
//	if size < realSize {
//		size = realSize
//	}
//
//	// Size of each module drawn.
//	pixelsPerModule := size / realSize
//
//	// Center the symbol within the image.
//	offset := (size - realSize*pixelsPerModule) / 2
//
//	rect := image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{size, size}}
//
//	// Saves a few bytes to have them in this order
//	p := color.Palette([]color.Color{q.BackgroundColor, q.ForegroundColor})
//	img := image.NewPaletted(rect, p)
//
//	for i := 0; i < size; i++ {
//		for j := 0; j < size; j++ {
//			img.Set(i, j, q.BackgroundColor)
//		}
//	}
//
//	bitmap := q.symbol.bitmap()
//	for y, row := range bitmap {
//		for x, v := range row {
//			if v {
//				startX := x*pixelsPerModule + offset
//				startY := y*pixelsPerModule + offset
//				for i := startX; i < startX+pixelsPerModule; i++ {
//					for j := startY; j < startY+pixelsPerModule; j++ {
//						img.Set(i, j, q.ForegroundColor)
//					}
//				}
//			}
//		}
//	}
//
//	return img
//}

func (q *qRCode) genImage(size int) image.Image {
	// Minimum pixels (both width and height) required.
	realSize := q.symbol.size

	// Variable size support.
	if size < 0 {
		size = size * -1 * realSize
	}

	// Actual pixels available to draw the symbol. Automatically increase the
	// image size if it's not large enough.
	if size < realSize {
		size = realSize
	}

	// Size of each module drawn.
	pixelsPerModule := float64(size) / float64(realSize)

	// Center the symbol within the image.
	//offset := (size - realSize*pixelsPerModule) / 2

	rect := image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{size, size}}

	// Saves a few bytes to have them in this order
	// NewPaletted 是生成png体积最小的方案。
	p := color.Palette([]color.Color{q.BackgroundColor, q.ForegroundColor})
	img := image.NewPaletted(rect, p)

	//for i := 0; i < size; i++ {
	//	for j := 0; j < size; j++ {
	//		img.SetColorIndex(i, j, 0)
	//	}
	//}

	bitmap := q.symbol.bitmap()
	for y, row := range bitmap {
		for x, v := range row {
			if v {
				startX := int(float64(x) * float64(pixelsPerModule))
				startY := int(float64(y) * float64(pixelsPerModule))
				endX := int(float64(x+1) * float64(pixelsPerModule))
				endY := int(float64(y+1) * float64(pixelsPerModule))
				for i := startX; i < endX; i++ {
					for j := startY; j < endY; j++ {
						img.SetColorIndex(i, j, 1)
					}
				}
			}
		}
	}

	return img
}

var gPngBufferPool pngEncoderPool

// PNG returns the QR Code as a PNG image.
//
// size is both the image width and height in pixels. If size is too small then
// a larger image is silently returned. Negative values for size cause a
// variable sized image to be returned: See the documentation for Image().
func (q *qRCode) pNG(size int) ([]byte, error) {
	img := q.genImage(size)

	var b bytes.Buffer
	//err := png.Encode(&b,img)
	encoder := png.Encoder{
		CompressionLevel: q.req.PngCompressionLevel,
		//BufferPool: &gPngBufferPool,
	}
	if q.req.PngUseBufferPool {
		encoder.BufferPool = &gPngBufferPool
	}

	err := encoder.Encode(&b, img)

	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Write writes the QR Code as a PNG image to io.Writer.
//
// size is both the image width and height in pixels. If size is too small then
// a larger image is silently written. Negative values for size cause a
// variable sized image to be written: See the documentation for Image().
//func (q *qRCode) write(size int, out io.Writer) error {
//	var png []byte
//
//	png, err := q.pNG(size)
//
//	if err != nil {
//		return err
//	}
//	_, err = out.Write(png)
//	return err
//}

// encode completes the steps required to encode the QR Code. These include
// adding the terminator bits and padding, splitting the data into blocks and
// applying the error correction, and selecting the best data mask.
func (q *qRCode) encode(numTerminatorBits int) {
	q.addTerminatorBits(numTerminatorBits)
	q.addPadding()

	encoded := q.encodeBlocks()

	const numMasks int = 8
	penalty := 0

	for mask := 0; mask < numMasks; mask++ {
		var s *symbol
		var err error

		s, err = buildRegularSymbol(q.version, mask, encoded)

		if err != nil {
			log.Panic(err.Error())
		}

		numEmptyModules := s.numEmptyModules()
		if numEmptyModules != 0 {
			log.Panicf("bug: numEmptyModules is %d (expected 0) (version=%d)",
				numEmptyModules, q.VersionNumber)
		}

		p := s.penaltyScore()

		//log.Printf("mask=%d p=%3d p1=%3d p2=%3d p3=%3d p4=%d\n", mask, p, s.penalty1(), s.penalty2(), s.penalty3(), s.penalty4())

		if q.symbol == nil || p < penalty {
			q.symbol = s
			q.mask = mask
			penalty = p
		}
	}
}

// addTerminatorBits adds final terminator bits to the encoded data.
//
// The number of terminator bits required is determined when the QR Code version
// is chosen (which itself depends on the length of the data encoded). The
// terminator bits are thus added after the QR Code version
// is chosen, rather than at the data encoding stage.
func (q *qRCode) addTerminatorBits(numTerminatorBits int) {
	q.data.AppendNumBools(numTerminatorBits, false)
}

// encodeBlocks takes the completed (terminated & padded) encoded data, splits
// the data into blocks (as specified by the QR Code version), applies error
// correction to each block, then interleaves the blocks together.
//
// The QR Code's final data sequence is returned.
func (q *qRCode) encodeBlocks() *kmgQrcodeBitset.Bitset {
	// Split into blocks.
	type dataBlock struct {
		data          *kmgQrcodeBitset.Bitset
		ecStartOffset int
	}

	block := make([]dataBlock, q.version.numBlocks())

	start := 0
	end := 0
	blockID := 0

	for _, b := range q.version.block {
		for j := 0; j < b.numBlocks; j++ {
			start = end
			end = start + b.numDataCodewords*8

			// Apply error correction to each block.
			numErrorCodewords := b.numCodewords - b.numDataCodewords
			block[blockID].data = kmgQrcodeReedsolomon.Encode(q.data.Substr(start, end), numErrorCodewords)
			block[blockID].ecStartOffset = end - start

			blockID++
		}
	}

	// Interleave the blocks.

	result := kmgQrcodeBitset.New()

	// Combine data blocks.
	working := true
	for i := 0; working; i += 8 {
		working = false

		for j, b := range block {
			if i >= block[j].ecStartOffset {
				continue
			}

			result.Append(b.data.Substr(i, i+8))

			working = true
		}
	}

	// Combine error correction blocks.
	working = true
	for i := 0; working; i += 8 {
		working = false

		for j, b := range block {
			offset := i + block[j].ecStartOffset
			if offset >= block[j].data.Len() {
				continue
			}

			result.Append(b.data.Substr(offset, offset+8))

			working = true
		}
	}

	// Append remainder bits.
	result.AppendNumBools(q.version.numRemainderBits, false)

	return result
}

// max returns the maximum of a and b.
//func max(a int, b int) int {
//	if a > b {
//		return a
//	}
//
//	return b
//}

// addPadding pads the encoded data upto the full length required.
func (q *qRCode) addPadding() {
	numDataBits := q.version.numDataBits()

	if q.data.Len() == numDataBits {
		return
	}

	// Pad to the nearest codeword boundary.
	q.data.AppendNumBools(q.version.numBitsToPadToCodeword(q.data.Len()), false)

	// Pad codewords 0b11101100 and 0b00010001.
	padding := [2]*kmgQrcodeBitset.Bitset{
		kmgQrcodeBitset.New(true, true, true, false, true, true, false, false),
		kmgQrcodeBitset.New(false, false, false, true, false, false, false, true),
	}

	// Insert pad codewords alternately.
	i := 0
	for numDataBits-q.data.Len() >= 8 {
		q.data.Append(padding[i])

		i = 1 - i // Alternate between 0 and 1.
	}

	if q.data.Len() != numDataBits {
		log.Panicf("BUG: got len %d, expected %d", q.data.Len(), numDataBits)
	}
}

// ToString produces a multi-line string that forms a QR-code image.
//func (q *qRCode) toString(inverseColor bool) string {
//	bits := q.bitmap()
//	var buf bytes.Buffer
//	for y := range bits {
//		for x := range bits[y] {
//			if bits[y][x] != inverseColor {
//				buf.WriteString("  ")
//			} else {
//				buf.WriteString("██")
//			}
//		}
//		buf.WriteString("\n")
//	}
//	return buf.String()
//}
