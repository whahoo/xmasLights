package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fogleman/gg"
	"github.com/kellydunn/go-opc"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/nfnt/resize"
	"image"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"
)

type Scroller struct {
	train_len int
}

type Vertex struct {
	X int
	Y int
	C colorful.Color
}

var home_c chan Scroller
var dc gg.Context

func randomFloat(min, max float64) float64 {
	return rand.Float64()*(max-min) + min
}

func random(min, max int) uint8 {
	xr := rand.Intn(max-min) + min
	return uint8(xr)
}

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

	inscroll.train_len = int(m["train_len"].(float64))

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
	//props := Scroller{40}

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

		im := nextFrame(*dc, effect, ledArray)
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
		//case props = <-c:
		case <-change.C:
			effect++
			if effect > 2 {
				effect = 0
			}
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

func (v1 *Vector) Add(v2 Vector) {
	v1.X += v2.X
	v1.Y += v2.Y
}

var st time.Time = time.Now()
var Width, Height float64 = 400, 120
var dotCenter Vector = Vector{float64(Width / 2.0), float64(Height / 2.0)}
var ps1 ParticleSystem = ParticleSystem{maxParticles: 50, Origin: Vector{float64(Width / 2), float64(10)}}
var imageList []string = []string{
	"glitter.png",
}
var images []image.Image

func nextFrame(dc gg.Context, effect int, leds []Vertex) image.Image {
	switch effect {
	case 0:
		rainbowFire(leds)
		return nil
		//return fallingBalls(dc)
	case 1:
		return scrollText(dc, "Ho, Ho, Ho - Merry Christmas!")
	case 2:
		return scrollImage(dc, images[0])
	case 3:
		rainbowFade(leds)
		return nil
	default:
		dc.Clear()
		return dc.Image()
	}
}

var lastFrameTime int64 = time.Now().UnixNano()
var initialHue byte = 0

func rainbowFire(leds []Vertex) {
	hue := float64(initialHue) //randomFloat(0, 360)
	row := int(float64(time.Since(st).Nanoseconds()/1000000)*0.015) % 15
	if row == 0 {
		initialHue = random(0, 360)
	}
	//for row := 0; row < 15; row++ {
	for i := 0; i < len(leds); i++ {
		//oldPixel := leds[i].C
		//oldHue, oldSat, oldBright := oldPixel.Hsv()
		leds[i].C = leds[i].C.BlendHcl(colorful.Hcl(0, 0, 0), 0.1) //colorful.Hsv(oldHue, oldSat, oldBright/1.15)

	}
	for i := 0; i < len(leds)/60; i++ {
		//	fmt.Println(i, i*60+row, i*60+29-row, i*60+30+row, i*60+59-row)
		leds[(i*60)+row].C = colorful.Hcl(hue, 1, 1)
		leds[(i*60)+29-row].C = colorful.Hcl(hue, 1, 1)
		leds[(i*60)+30+row].C = colorful.Hcl(hue, 1, 1)
		leds[(i*60)+59-row].C = colorful.Hcl(hue, 1, 1)
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
	if random(0, 255) < 80 {
		leds[random(0, len(leds))].C = colorful.Hsv(0, 0, 255)
	}
}

func scrollImage(dc gg.Context, image image.Image) image.Image {

	y := int(float64(time.Since(st).Nanoseconds()/1000000)*-0.04) % int(image.Bounds().Size().Y)

	dc.DrawImage(image, 0, y)
	dc.DrawImage(image, 0, y+image.Bounds().Size().Y)
	return dc.Image()
}

func scrollText(dc gg.Context, message string) image.Image {
	dc.Clear()
	dc.SetColor(colorful.FastHappyColor())
	textWidth, _ := dc.MeasureString(message)
	x := int(Width) + int(float64(time.Since(st).Nanoseconds()/1000000)*-0.08)%int(textWidth+Width)

	dc.DrawStringAnchored(message, float64(x), float64(Height/2), 0.5, 0.5)
	dc.DrawStringAnchored(message, textWidth+Width+float64(x), float64(Height/2), 0.5, 0.5)

	return dc.Image()
}

func fallingBalls(dc gg.Context) image.Image {
	dc.Clear()
	ps1.addParticle()
	ps1.run()
	for _, p := range ps1.Particles {
		dc.DrawCircle(p.Location.X, p.Location.Y, p.size)
		dc.SetColor(p.color)
		dc.Fill()
	}
	return dc.Image()
}

type ParticleSystem struct {
	Origin       Vector
	Particles    []Particle
	maxParticles int
}

func (ps *ParticleSystem) addParticle() {
	if len(ps.Particles) > ps.maxParticles {
		ps.Particles = ps.Particles[1:]
	}
	p := Particle{
		ps.Origin,
		Vector{randomFloat(-5, 5), randomFloat(-5, 5)},
		Vector{0, randomFloat(0.01, 0.12)},
		colorful.Hcl(rand.Float64()*360.0, rand.Float64(), 0.6+rand.Float64()*0.4),
		8,
		255,
	}
	ps.Particles = append(ps.Particles, p)
}
func (ps *ParticleSystem) run() {
	for i, _ := range ps.Particles {
		ps.Particles[i].update()
	}
}

type Particle struct {
	Location, Velocity, Acceleration Vector
	color                            colorful.Color
	size, lifespan                   float64
}

func (p *Particle) update() {
	p.Velocity.Add(p.Acceleration)
	p.Location.Add(p.Velocity)
}
