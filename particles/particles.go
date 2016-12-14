package particles

import (
	"image"
	"math/rand"

	"github.com/fogleman/gg"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/whahoo/xmasLights/util"
)

var ps1 ParticleSystem = ParticleSystem{maxParticles: 50, Origin: Vector{0, 0}}

var snow ParticleSystem = ParticleSystem{maxParticles: 100}

var balls ParticleSystem = ParticleSystem{maxParticles: 20}

func ExpandingBalls(dc gg.Context) image.Image {
	dc.Clear()
	balls.addParticle(Particle{
		Vector{float64(rand.Intn(dc.Width())), float64(rand.Intn(dc.Height()))},
		Vector{util.RandomFloat(-0.1, 0.2), util.RandomFloat(-0.2, 0.2)},
		Vector{util.RandomFloat(-0.1, 0.21), util.RandomFloat(-0.2, 0.2)},
		colorful.Hcl(rand.Float64()*360.0, rand.Float64(), 0.6+rand.Float64()*0.4),
		4,
		255,
	})
	for i, p := range balls.Particles {
		balls.Particles[i].size += 1
		balls.Particles[i].update()
		dc.DrawCircle(balls.Particles[i].Location.X, balls.Particles[i].Location.Y, balls.Particles[i].size)
		dc.SetColor(p.color)
		dc.Fill()
	}
	return dc.Image()

}
func FallingBalls(dc gg.Context) image.Image {
	dc.Clear()
	ps1.Origin = Vector{float64(dc.Width() / 2), float64(10)}

	ps1.addParticle(Particle{
		ps1.Origin,
		Vector{util.RandomFloat(-16, 16), util.RandomFloat(-6, 8)},
		Vector{0, util.RandomFloat(0.01, 0.12)},
		colorful.Hcl(rand.Float64()*360.0, rand.Float64(), 0.6+rand.Float64()*0.4),
		8,
		255,
	})
	for i, p := range ps1.Particles {
		ps1.Particles[i].update()
		dc.DrawCircle(ps1.Particles[i].Location.X, ps1.Particles[i].Location.Y, ps1.Particles[i].size)
		dc.SetColor(p.color)
		dc.Fill()
	}
	return dc.Image()
}

func Snow(dc gg.Context) image.Image {
	dc.Clear()
	snow.Origin = Vector{float64(2), float64(dc.Height() / 2)}
	snow.addParticle(Particle{
		snow.Origin,
		Vector{util.RandomFloat(-5, 5), util.RandomFloat(-5, 5)},
		Vector{0, util.RandomFloat(0.00, 0.08)},
		colorful.Color{1, 1, 1},
		6,
		255,
	})
	wind := Vector{util.RandomFloat(0.15, 0.04), util.RandomFloat(-1, 1)}
	for i, p := range snow.Particles {
		snow.Particles[i].Acceleration = wind
		snow.Particles[i].update()
		dc.DrawCircle(snow.Particles[i].Location.X, snow.Particles[i].Location.Y, snow.Particles[i].size)
		dc.SetColor(p.color)
		dc.Fill()
	}
	return dc.Image()
}

type Vector struct {
	X float64
	Y float64
}

func (v1 *Vector) Add(v2 Vector) {
	v1.X += v2.X
	v1.Y += v2.Y
}

type ParticleSystem struct {
	Origin       Vector
	Particles    []Particle
	maxParticles int
}

func (ps *ParticleSystem) addParticle(p Particle) {
	if len(ps.Particles) > ps.maxParticles {
		ps.Particles = ps.Particles[1:]
	}
	ps.Particles = append(ps.Particles, p)
}

func (ps *ParticleSystem) run() {
	for i, _ := range ps.Particles {
		ps.Particles[i].update()
	}
}

func (ps *ParticleSystem) blow() {
	wind := Vector{util.RandomFloat(0.15, 0.04), util.RandomFloat(-1, 1)}
	for i, _ := range ps.Particles {
		ps.Particles[i].Acceleration = wind
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
