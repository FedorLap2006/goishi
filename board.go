package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

var boardFont *truetype.Font

func init() {
	var err error
	boardFont, err = truetype.Parse(goregular.TTF)
	if err != nil {
		panic("could not parse goregular font: " + err.Error())
	}
}

type BoardCellState uint8

const (
	BoardCellEmpty BoardCellState = 0
	BoardCellBlack BoardCellState = 'b'
	BoardCellWhite BoardCellState = 'w'
)

func (c BoardCellState) Encode() byte {
	switch c {
	case BoardCellBlack:
		return byte(0x1)
	case BoardCellWhite:
		return byte(0x2)
	}
	return byte(0x0)
}

func DecodeBoardCell(b byte) BoardCellState {
	switch b {
	case byte(0x1):
		return BoardCellBlack
	case byte(0x2):
		return BoardCellWhite
	}
	return BoardCellEmpty
}

type Board struct {
	sizeX, sizeY int
	data         []BoardCellState
	lastMove     [2]int
	WhiteToMove  bool
}

var ErrBoardTooLarge = fmt.Errorf("board too large")

func NewBoard(sizeX, sizeY int) (*Board, error) {
	if sizeX > 19 || sizeY > 19 {
		return nil, ErrBoardTooLarge
	}
	return &Board{sizeX: sizeX, sizeY: sizeY, data: make([]BoardCellState, sizeX*sizeY)}, nil
}

func CoordToIndex(row byte, col int) [2]int {
	return [2]int{int(row) - int('A'), col - 1}
}

var starPoints9 = [][2]int{
	{2, 2}, {8 - 2, 2},
	{4, 4},
	{2, 8 - 2}, {8 - 2, 8 - 2},
}

var starPoints13 = [][2]int{
	{6, 6},
	{3, 3},
	{3, 12 - 3},
	{12 - 3, 3},
	{12 - 3, 12 - 3},
}

var starPoints19 = [][2]int{
	{3, 3}, {9, 3}, {18 - 3, 3},
	{3, 9}, {9, 9}, {18 - 3, 9},
	{3, 18 - 3}, {9, 18 - 3}, {18 - 3, 18 - 3},
}

func (b Board) Draw(cnv *gg.Context, padding float64, showCoords bool) {
	cnv.Clear()
	drawBoard(cnv, padding, b.sizeX, b.sizeY, showCoords)
	stepX := boardStep(float64(cnv.Width()), padding, b.sizeX)
	stepY := boardStep(float64(cnv.Height()), padding, b.sizeY)

	// TODO: star points. From the looks of it, they are diving the board into three parts by the sides.
	// Two side ones make one part if you connect them.

	// starPointsX := (cnv.sizeX-1) / 3
	// cnv.SetHexColor("#000000")
	// cnv.DrawCircle(padding + step * )

	stepMin := math.Min(stepX, stepY)

	var starPoints [][2]int

	switch {
	case b.sizeX == 9 && b.sizeY == 9:
		starPoints = starPoints9
	case b.sizeX == 13 && b.sizeY == 13:
		starPoints = starPoints13
	case b.sizeX == 19 && b.sizeY == 19:
		starPoints = starPoints19
	}

	for _, c := range starPoints {
		posX, posY := stonePosition(padding, stepX, stepY, c[0], c[1])
		cnv.DrawCircle(posX, posY, stepMin/15)
		cnv.Fill()
	}

	for y := 0; y < b.sizeY; y++ {
		for x := 0; x < b.sizeX; x++ {
			if b.data[y*b.sizeX+x] == BoardCellEmpty {
				continue
			}
			posX, posY := stonePosition(padding, stepX, stepY, x, y)
			// drawStone(cnv, padding, stepMin/2.5, posX, posY, b.data[y*b.sizeX+x] == BoardCellWhite)
			drawStone(cnv, padding, stepMin/3, posX, posY, b.data[y*b.sizeX+x] == BoardCellWhite)
		}
	}
}

func (b Board) Encode() (d []byte) {
	bitlen := (b.sizeX * b.sizeY * 2)
	bytelen := bitlen / 8
	if bitlen%8 != 0 {
		bytelen++
	}
	d = make([]byte, bytelen)
	for i, v := range b.data {
		d[i/4] |= v.Encode() << ((i % 4) * 2)
		// if v == BoardCellEmpty {
		// 	v = ' '
		// }
		// fmt.Print(string(rune(v)), "/", fmt.Sprintf("%.8b", d[i/4]), " ")
		// fmt.Println()
	}
	return
}

func DecodeBoard(sizeX, sizeY int, data []byte) (b *Board, err error) {
	b, err = NewBoard(sizeX, sizeY)
	if err != nil {
		return nil, err
	}

	for i, v := range data {
		for j := 0; j < 4; j++ {
			idx := i*4 + j
			if idx < sizeX*sizeY {
				// b.data[i*4+j] = DecompressBoardCell(v >> ((i % 4) * 2))
				b.data[i*4+j] = DecodeBoardCell((v >> (j * 2)) & 0x3)
			}
		}
	}
	return
}

func (b Board) checkCoordBounds(x, y int) (bool, bool) {
	return x < b.sizeX && x >= 0, y < b.sizeY && y >= 0
}

func (b *Board) SetCell(x, y int, state BoardCellState) {
	if rx, ry := b.checkCoordBounds(x, y); !rx || !ry {
		return
	}

	b.data[y*b.sizeX+x] = state
}

func (b Board) GetCell(x, y int) BoardCellState {
	if rx, ry := b.checkCoordBounds(x, y); !rx || !ry {
		return BoardCellEmpty
	}
	return b.data[y*b.sizeX+x]
}

func (b Board) checkLegality(x, y int, white bool) bool {
	return true
}

func (b *Board) MakeMoveWithPlayer(x, y int, white bool) bool { // TODO: pass & current move
	stone := BoardCellBlack
	if white {
		stone = BoardCellWhite
	}

	if !b.checkLegality(x, y, white) {
		return false
	}

	b.SetCell(x, y, stone)
	return true
}

func (b *Board) MakeMove(x, y int) (res bool) {
	res = b.MakeMoveWithPlayer(x, y, b.WhiteToMove)
	if res {
		b.WhiteToMove = !b.WhiteToMove
	}

	return
}

// Returns board size in (x, y) form
func (b Board) Size() (int, int) {
	return b.sizeX, b.sizeY
}

func boardStep(max float64, padding float64, size int) float64 {
	return (float64(max) - padding*2.0 - 0.5) / float64(size-1)
}

func drawBoard(cnv *gg.Context, padding float64, boardSizeX, boardSizeY int, showCoords bool) {
	if cnv.Height() != cnv.Width() {
		return
	}

	// cnv.SetHexColor("FF0AA0")
	cnv.DrawRectangle(0, 0, float64(cnv.Height()), float64(cnv.Width()))
	cnv.SetHexColor("#D19A33")
	cnv.Fill()

	stepX := boardStep(float64(cnv.Height()), padding, boardSizeX)
	stepY := boardStep(float64(cnv.Height()), padding, boardSizeY)

	cnv.SetHexColor("#0a0a0a")
	for x := padding; x <= float64(cnv.Width())-padding; x += stepX {
		// fmt.Println(x, 0, x, float64(cnv.Height())-padding)
		cnv.DrawLine(x, padding, x, float64(cnv.Height())-padding)
	}
	cnv.Stroke()
	for y := padding; y <= float64(cnv.Height())-padding; y += stepY {
		// fmt.Println(x, 0, x, float64(cnv.Height())-padding)
		cnv.DrawLine(padding, y, float64(cnv.Height())-padding, y)
	}
	cnv.Stroke()
	cnv.Fill()

	if showCoords && padding > 30.0 {
		cnv.SetHexColor("#0a0a0a")
		dpi := 96.0
		ptdpiRatio := (72 / dpi)
		fontSize := math.Min((padding*0.35)*ptdpiRatio, 24)
		// fmt.Println(fontSize)
		face := truetype.NewFace(boardFont, &truetype.Options{Size: fontSize, DPI: dpi})
		cnv.SetFontFace(face)
		for i := 0; i < boardSizeX; i++ {
			cnv.DrawStringAnchored(string('A'+rune(i)), padding+float64(i)*stepX, padding*0.5, 0.5, 0.5)
			cnv.DrawStringAnchored(string('A'+rune(i)), padding+float64(i)*stepX, float64(cnv.Height())-padding*0.5, 0.5, 0.5)
		}
		for i := 0; i < boardSizeY; i++ {
			// fmt.Printf("%v %+v %v\n", face.Metrics().Ascent.Round(), face.Metrics(), padding/2.0)
			// cnv.DrawStringAnchored(strconv.Itoa(i+1), padding*0.5, padding+float64(i)*stepY-float64(face.Metrics().Height.Round())/(ptdpiRatio*2.0), 0.5, 0)
			// cnv.DrawStringAnchored(strconv.Itoa(i+1), float64(cnv.Width())-padding*0.5, padding+float64(i)*stepY-float64(face.Metrics().Height.Round())/(ptdpiRatio*2.0), 0.5, 0)
			// fmt.Println(float64(face.Metrics().Height.Ceil()), ptdpiRatio, float64(face.Metrics().Height.Ceil())/ptdpiRatio)
			cnv.DrawStringAnchored(strconv.Itoa(i+1), padding*0.5, padding+float64(i)*stepY-float64(face.Metrics().Height.Round())/(ptdpiRatio*8), 0.5, 0.5)
			cnv.DrawStringAnchored(strconv.Itoa(i+1), float64(cnv.Width())-padding*0.5, padding+float64(i)*stepY-float64(face.Metrics().Height.Round())/(ptdpiRatio*8), 0.5, 0.5)

		}
		cnv.Fill()
	}

	// cnv.SetRGB(200, 100, 200)
}

func stonePosition(padding float64, stepX, stepY float64, x int, y int) (float64, float64) {
	return padding + stepX*float64(x), padding + stepY*float64(y)
}

func drawStone(cnv *gg.Context, padding float64, size float64, posx float64, posy float64, white bool) {
	cnv.SetHexColor("#000000")
	if white {
		cnv.SetLineWidth(2)
		cnv.DrawCircle(posx, posy, size+2)
		cnv.Fill()
		cnv.SetHexColor("#FFFFFF")
	}
	cnv.DrawCircle(posx, posy, size)
	cnv.Fill()
}
