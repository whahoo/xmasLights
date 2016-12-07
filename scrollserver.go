package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fogleman/gg"
	"github.com/kellydunn/go-opc"
	"github.com/lucasb-eyer/go-colorful"
	"image"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"
)

type Color struct {
	R, G, B uint8
}

type Scroller struct {
	delay, train_len int
	random           bool
	color            Color
}

type Vertex struct {
	X int
	Y int
}

var color = Color{255, 0, 0}
var home_c chan Scroller

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
			int(y + float64(i-(count-1.0)/2.0)*spacing*s + 0.5)}
		//		fmt.Println(stripIndex, ledArray[index])

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
	leds_len := flag.Int("leds", 750, "Number of LEDs in the string")
	flag.Parse()

	home_c = make(chan Scroller, 1)

	leds := make([]Vertex, *leds_len)
	ledGrid(leds, 0, 15, 50, 400/2, 120/2, 120/15, 400/50, 1.5708, true)

	go func() { LEDSender(home_c, *serverPtr, *leds_len, leds) }()

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

	inscroll.delay = int(m["delay"].(float64))
	inscroll.train_len = int(m["train_len"].(float64))
	inscroll.random = bool(m["random"].(bool))
	colormap := m["color"].(map[string]interface{})
	inscroll.color.R = uint8(colormap["r"].(float64))
	inscroll.color.G = uint8(colormap["g"].(float64))
	inscroll.color.B = uint8(colormap["b"].(float64))

	ss := inscroll

	//send on the home channel, nonblocking
	select {
	case home_c <- ss:
	default:
		log.Println("msg NOT sent")
	}

	fmt.Fprintf(w, "HomeHandler", ss.delay)
}

func LEDSender(c chan Scroller, server string, leds_len int, ledArray []Vertex) {

	props := Scroller{40, 7, false, Color{255, 0, 0}}
	props.delay = 10

	// Create a client
	oc := opc.NewClient()
	err := oc.Connect("tcp", server)
	if err != nil {
		log.Fatal("Could not connect to Fadecandy server", err)
	}
	for {

		im := nextFrame()
		m := opc.NewMessage(0)
		m.SetLength(uint16(leds_len * 3))
		for i := 0; i < leds_len; i++ {
			pixelRed, pixelGreen, pixelBlue, _ := im.At(ledArray[i].X, ledArray[i].Y).RGBA()
			m.SetPixelColor(i, uint8(pixelRed), uint8(pixelGreen), uint8(pixelBlue))
		}
		err := oc.Send(m)
		if err != nil {
			log.Println("couldn't send color", err)
		}
		time.Sleep(time.Duration(props.delay) * time.Millisecond)

		// receive from channel
		select {
		case props = <-c:
		default:
			//	}
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

var Width, Height int = 400, 120
var dotCenter Vector = Vector{float64(Width / 2.0), float64(Height / 2.0)}
var ps1 ParticleSystem = ParticleSystem{maxParticles: 50, Origin: Vector{float64(Width / 2), float64(10)}}

func nextFrame() image.Image {
	//	move := Vector{randomFloat(-4, 4), randomFloat(-4, 5)}
	//	dotCenter.Add(move)
	dc := gg.NewContext(Width, Height)
	//	dc.DrawCircle(dotCenter.X, dotCenter.Y, 40)

	//	dc.SetColor(colorful.FastHappyColor())
	//	dc.Fill()

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
		//	if int(ps.Particles[i].Location.X) > Width+10 || int(ps.Particles[i].Location.X) < -10 ||
		//		int(ps.Particles[i].Location.Y) > Height+10 || int(ps.Particles[i].Location.Y) < -10 {
		//		ps.Particles[len(ps.Particles)-1], ps.Particles[i] = ps.Particles[i], ps.Particles[len(ps.Particles)-1]
		//		ps.Particles = ps.Particles[:len(ps.Particles)-1]
		//	}
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
