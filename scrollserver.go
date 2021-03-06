package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/fogleman/gg"
	"github.com/kellydunn/go-opc"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/nfnt/resize"
	"github.com/whahoo/xmasLights/particles"
)

type Scroller struct {
	text string
}

type Vertex struct {
	X int
	Y int
	C colorful.Color
}

var home_c chan Scroller
var dc gg.Context

func ledStrip(ledArray []Vertex, index, count int, x, y, spacing, angle float64, reversed bool) {
	s := math.Sin(angle)
	c := math.Cos(angle)
	for i := 0; i < count; i++ {
		stripIndex := index + i
		if reversed {
			stripIndex = index + count - 1 - i
		}
		ledArray[stripIndex] = Vertex{int(x + float64(i-(count-1.0)/2.0)*spacing*c + 0.5),
			int(y + float64(i-(count-1.0)/2.0)*spacing*s + 0.5), colorful.Color{0, 0, 0}}

	}
}

func ledGrid(ledArray []Vertex, index, stripLength, numStrips int, x, y, ledSpacing, stripSpacing, angle float64, zigzag bool) {

	s := math.Sin(angle + math.Pi/2)
	c := math.Cos(angle + math.Pi/2)
	for i := 0; i < numStrips; i++ {
		ledStrip(ledArray, index+stripLength*i, stripLength,
			x+float64(i-(numStrips-1)/2.0)*stripSpacing*c,
			y+float64(i-(numStrips-1)/2.0)*stripSpacing*s,
			ledSpacing, angle, zigzag && (i%2) == 1)
	}
}

func main() {
	rand.Seed(time.Now().Unix())

	serverPtr := flag.String("fcserver", "localhost:7890", "Fadecandy server and port to connect to")
	listenPortPtr := flag.Int("port", 8080, "Port to serve UI from")
	leds_len := flag.Int("leds", 52*15, "Number of LEDs in the string")

	flag.Parse()

	home_c = make(chan Scroller, 1)

	leds := make([]Vertex, *leds_len)
	ledGrid(leds, 0, 15, 52, 400/2, 120/2, 120/15, 400/50, 1.5708, true)

	ticker := time.NewTicker(time.Millisecond * 20)
	go func() { LEDSender(home_c, *serverPtr, *leds_len, leds, *ticker) }()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/", http.StripPrefix("/", fs))
	http.HandleFunc("/update", UpdateHandler)

	log.Println("Listening on", fmt.Sprintf("http://0.0.0.0:%d", *listenPortPtr), "...")
	http.ListenAndServe(fmt.Sprintf(":%d", *listenPortPtr), nil)
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {

	// do these stupid hacks for parsing JSON.
	// go is pretty bad at this
	body, _ := ioutil.ReadAll(r.Body)
	var f interface{}
	var inscroll Scroller
	json.Unmarshal(body, &f)

	m := f.(map[string]interface{})

	inscroll.text = string(m["text"].(string))
	ss := inscroll

	//send on the home channel, nonblocking
	select {
	case home_c <- ss:
	default:
		log.Println("msg NOT sent")
	}

	fmt.Fprintf(w, "HomeHandler", "100")
}

func LEDSender(c chan Scroller, server string, leds_len int, ledArray []Vertex, ticker time.Ticker) {

	//
	props := Scroller{text: "Ok"}
	text := "Ho, Ho, Ho - Merry Christmas!!"
	timer1 := time.NewTimer(1)
	// Create a client
	oc := opc.NewClient()
	err := oc.Connect("tcp", server)
	if err != nil {
		log.Fatal("Could not connect to Fadecandy server", err)
	}
	dc := gg.NewContext(int(Width), int(Height))
	if err := dc.LoadFontFace("Arial.ttf", 96); err != nil {
		panic(err)
	}
	change := time.NewTicker(time.Second * 20)
	effect := 0
	loadImages()

	for t := range ticker.C {

		im := nextFrame(*dc, effect, ledArray, text)
		m := opc.NewMessage(0)
		m.SetLength(uint16(leds_len * 3))
		if im != nil {
			for i := 0; i < leds_len; i++ {
				pixelRed, pixelGreen, pixelBlue, _ := im.At(ledArray[i].X, ledArray[i].Y).RGBA()
				m.SetPixelColor(i, uint8(pixelRed), uint8(pixelGreen), uint8(pixelBlue))
			}
		} else {
			for i := 0; i < leds_len; i++ {
				m.SetPixelColor(i, uint8(ledArray[i].C.R*255), uint8(ledArray[i].C.G*255), uint8(ledArray[i].C.B*255))
			}
		}

		err := oc.Send(m)
		if err != nil {
			log.Println("couldn't send color", t, err)
		}

		// receive from channel
		select {
		case props = <-c:
			change.Stop()
			text = props.text
			timer1 = time.NewTimer(time.Second * 20)
			effect = 1
		case <-change.C:
			effect++
			if effect > 17 {
				effect = 0
			}
		case <-timer1.C:
			change = time.NewTicker(time.Second * 20)
			text = "Ho, Ho, Ho - Merry Christmas!!"
		default:
		}
	}
}

func loadImages() {
	for _, im := range imageList {

		if pic, err := gg.LoadImage(im); err == nil {
			images = append(images, resize.Resize(uint(Width), 0, pic, resize.NearestNeighbor))
		}
	}
}

type Vector struct {
	X float64
	Y float64
}

var st time.Time = time.Now()
var Width, Height float64 = 400, 120

var imageList []string = []string{
	"glitter.png",
	"bubbles.png",
	"color2.png",
	"redChristmasTree.png",
	"greenSnowFlakes.png",
}
var images []image.Image

func nextFrame(dc gg.Context, effect int, leds []Vertex, text string) image.Image {
	switch effect {
	case 0:
		whiteSparkles(leds)
		return nil

	case 1:
		return scrollText(dc, text) //"Ho, Ho, Ho - Merry Christmas!")
	case 2:
		return scrollImage(dc, images[0])
	case 3:
		rainbowFade(leds)
		return nil
	case 4:
		return particles.FallingBalls(dc)
	case 5:
		rainbowFire(leds)
		return nil
	case 6:
		redSparkles(leds)
		return nil
	case 7:
		return particles.Snow(dc)
	case 8:
		goldSparkles(leds)
		return nil
	case 9:
		return particles.ExpandingBalls(dc)
	case 10:
		return Triangles(dc)
	case 11:
		blues(leds)
		return nil
	case 12:
		xmasLights(leds)
		return nil
	case 13:
		waveLights(leds)
		return nil
	case 14:
		paigeLights(leds)
		return nil
	case 15:
		return scrollImage(dc, images[1])
	case 16:
		return scrollImage(dc, images[2])
	case 17:
		return scrollImage(dc, images[3])
	default:
		dc.Clear()
		return dc.Image()
	}
}

var lastFrameTime int64 = time.Now().UnixNano()
var initialHue byte = 0
var row, lastrow int = 0, 0
var rowcolor colorful.Color = colorful.Hcl(rand.Float64()*360.0, rand.Float64(), 0.6+rand.Float64()*0.4)

func whiteSparkles(leds []Vertex) {
	sparkles(leds, colorful.Color{1, 1, 1}, colorful.Color{0, 0, 0})
}
func goldSparkles(leds []Vertex) {
	sparkles(leds, colorful.Color{238.0 / 255.0, 169.0 / 255.0, 0}, colorful.Color{0, 0, 0})
}
func redSparkles(leds []Vertex) {
	sparkles(leds, colorful.Color{1, 0, 0}, colorful.Color{32.0 / 255.0, 165.0 / 255.0, 22.0 / 255.0})
}

func sparkles(leds []Vertex, sparkle, background colorful.Color) {
	for i := 0; i < len(leds); i++ {
		if rand.Intn(12) == 2 {
			leds[i].C = sparkle
		} else {
			leds[i].C = leds[i].C.BlendHcl(background, 0.5).Clamped()
		}
	}
}

var pixelOffset int = 0
var fade float64 = 255

func blues(leds []Vertex) {
	colours := []colorful.Color{colorful.Hsv(250, 0.91, 0.99), colorful.Hsv(210.0, 0.91, 0.99), colorful.Hsv(210.0, 1, 1)}
	runPixels(pixelOffset, 1, colours, leds)
	if fade == 0 {
		pixelOffset++
		fade = 255
	} else {
		fade -= 18
	}
	if fade < 20 {
		fade = 0
	}
	if pixelOffset == 2 {
		pixelOffset = 0
	}
	leds[rand.Intn(len(leds))].C = colorful.Color{1, 1, 1}
}
func xmasLights(leds []Vertex) {
	colours := []colorful.Color{colorful.Hsv(0, 1, fade/255.0), colorful.Hsv(120.0, 1, fade/255.0), colorful.Hsv(240.0, 1, fade/255.0)}
	runPixels(pixelOffset, 4, colours, leds)

	if fade == 0 {
		pixelOffset++
		fade = 255
	} else {
		fade -= 8
	}
	if fade < 20 {
		fade = 0
	}
	if pixelOffset == 4 {
		pixelOffset = 0
	}
}

//color(111,85, fade), color(141,81,fade), color(291,28,fade), color(321,39,fade)
func paigeLights(leds []Vertex) {
	colours := []colorful.Color{colorful.Hsv(111, 0.85, fade/255.0), colorful.Hsv(141.0, 0.81, fade/255.0), colorful.Hsv(291.0, .28, fade/255.0), colorful.Hsv(321.0, 0.39, fade/255)}
	runPixels(pixelOffset, 15, colours, leds)

	if fade == 0 {
		pixelOffset++
		fade = 255
	} else {
		fade -= 22
	}
	if fade < 20 {
		fade = 0
	}
	if pixelOffset == 15 {
		pixelOffset = 0
	}
}
func waveLights(leds []Vertex) {
	colours := []colorful.Color{colorful.Hsv(90, 0.85, fade/255.0), colorful.Hsv(100.0, 0.81, fade/255.0), colorful.Hsv(110.0, .28, fade/255.0), colorful.Hsv(120.0, 0.39, fade/255)}
	runPixels(pixelOffset, 7, colours, leds)

	if fade == 0 {
		pixelOffset++
		fade = 255
	} else {
		fade -= 22
	}
	if fade < 20 {
		fade = 0
	}
	if pixelOffset == 15 {
		pixelOffset = 0
	}
}
func runPixels(offset, jump int, colours []colorful.Color, leds []Vertex) {
	l := 0
	c := 0
	for l = offset; l < len(leds); l = l + jump {
		leds[l].C = colours[c]
		c = c + 1
		if c >= len(colours) {
			c = 0
		}
	}
}

func rainbowFire(leds []Vertex) {
	lastrow = row
	row = int(float64(time.Since(st).Nanoseconds()/1000000)*0.015) % 15
	if row == 0 && lastrow != 0 {
		rowcolor = colorful.Hcl(rand.Float64()*360.0, rand.Float64(), 0.6+rand.Float64()*0.4)
		//fmt.Println(rowcolor)
	}
	for i := 0; i < len(leds); i++ {
		leds[i].C = leds[i].C.BlendHcl(colorful.Hcl(0, 0, 0), 0.09).Clamped()

	}
	for i := 0; i < len(leds)/60; i++ {
		leds[(i*60)+row].C = rowcolor
		leds[(i*60)+29-row].C = rowcolor
		leds[(i*60)+30+row].C = rowcolor
		leds[(i*60)+59-row].C = rowcolor
	}
}

func rainbowFade(leds []Vertex) {
	if time.Now().UnixNano()-lastFrameTime >= int64(20*time.Millisecond) {
		initialHue++
		lastFrameTime = time.Now().UnixNano()
	}
	// RAINBOW FADE!!!!!
	hue := initialHue
	val := 204.0 / 255.0
	sat := 151.0 / 255.0
	for i := 0; i < len(leds); i++ {
		leds[i].C = colorful.Hsv(float64(hue&0xFF), sat, val)
		hue += 1
	}
	if rand.Intn(255) < 80 {
		leds[rand.Intn(len(leds))].C = colorful.Hsv(0, 0, 1)
	}
}

func scrollImage(dc gg.Context, image image.Image) image.Image {

	y := int(float64(time.Since(st).Nanoseconds()/1000000)*-0.04) % int(image.Bounds().Size().Y)

	dc.DrawImage(image, 0, y)
	dc.DrawImage(image, 0, y+image.Bounds().Size().Y)
	return dc.Image()
}

func scrollText(dc gg.Context, message string) image.Image {
	lastrow = row
	row = int(float64(time.Since(st).Nanoseconds()/1000000)*0.015) % 15
	if row == 0 && lastrow != 0 {
		rowcolor = colorful.FastHappyColor()
	}

	dc.Clear()
	dc.SetColor(rowcolor)
	textWidth, _ := dc.MeasureString(message)
	x := int(Width) + int(float64(time.Since(st).Nanoseconds()/1000000)*-0.08)%int(textWidth+Width)

	dc.DrawStringAnchored(message, float64(x), float64(Height/2), 0.5, 0.5)
	dc.DrawStringAnchored(message, textWidth+Width+float64(x), float64(Height/2), 0.5, 0.5)

	return dc.Image()
}

func Triangles(dc gg.Context) image.Image {
	dc.Clear()
	dc.SetHexColor("000")
	dc.Clear()
	n := 5
	points := Polygon(n)
	const S = 80
	for x := S / 2; x < dc.Width(); x += S {
		dc.Push()
		s := (float64(int(float64(time.Since(st).Nanoseconds()/1000000)*-0.5)%75)/75+1)*S/4 + S/4
		dc.Translate(float64(x), float64(dc.Height()/2))
		dc.Rotate(float64(int(float64(time.Since(st).Nanoseconds()/1000000)*-0.1)%360) / 360 * 2 * math.Pi)
		dc.Scale(s, s)
		for i := 0; i < n+1; i++ {
			index := (i * 2) % n
			p := points[index]
			dc.LineTo(p.X, p.Y)
		}
		dc.SetLineWidth(10)
		dc.SetHexColor("#FFCC00")
		dc.StrokePreserve()
		dc.SetHexColor("#FFE43A")
		dc.Fill()
		dc.Pop()
	}

	return dc.Image()

}
func Polygon(n int) []Vector {
	result := make([]Vector, n)
	for i := 0; i < n; i++ {
		a := float64(i)*2*math.Pi/float64(n) - math.Pi/2
		result[i] = Vector{math.Cos(a), math.Sin(a)}
	}
	return result
}
