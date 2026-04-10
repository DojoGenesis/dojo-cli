package art

import (
	"math/rand"
	"time"
)

// Petal is a falling cherry blossom particle.
type Petal struct {
	X, Y    float64 // position (fractional for smooth movement)
	VX, VY  float64 // velocity
	Char    rune    // display character: '·', '*', '°', '\'', ','
	ColorID int     // 0=gold, 1=amber, 2=terra (maps to sunset palette)
	Life    int     // remaining frames
	MaxLife int     // initial life (for fade calculation)
}

// Firefly is a rising spirit energy particle.
type Firefly struct {
	X, Y  float64
	VY    float64 // negative = rising
	Phase int     // blink phase counter
	Life  int
}

// GroundPetal is a fallen petal resting on the ground.
type GroundPetal struct {
	X       int
	Char    rune
	ColorID int
	Life    int // frames until fade
}

// ParticleEngine manages all particles in the bloom animation.
type ParticleEngine struct {
	Petals       []Petal
	Fireflies    []Firefly
	Ground       []GroundPetal
	rng          *rand.Rand
	maxPetals    int
	maxFireflies int
	maxGround    int
}

// NewParticleEngine creates an engine with default capacity limits.
func NewParticleEngine() *ParticleEngine {
	return &ParticleEngine{
		Petals:       make([]Petal, 0, 40),
		Fireflies:    make([]Firefly, 0, 8),
		Ground:       make([]GroundPetal, 0, 20),
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
		maxPetals:    40,
		maxFireflies: 8,
		maxGround:    20,
	}
}

// PetalChars returns the possible petal display characters.
func PetalChars() []rune {
	return []rune{'·', '*', '°', '\'', ','}
}

// PetalColorHex returns the hex color for a petal ColorID.
func PetalColorHex(colorID int) string {
	switch colorID {
	case 0:
		return "#ffd166" // gold
	case 1:
		return "#f4a261" // amber
	case 2:
		return "#e76f51" // terra
	default:
		return "#ffd166"
	}
}

// spawnPetalChar picks a weighted random petal character.
// Weights: · 40%, * 25%, ° 15%, ' 10%, , 10%
func (e *ParticleEngine) spawnPetalChar() rune {
	n := e.rng.Intn(100)
	switch {
	case n < 40:
		return '·'
	case n < 65:
		return '*'
	case n < 80:
		return '°'
	case n < 90:
		return '\''
	default:
		return ','
	}
}

// spawnPetalColor picks a weighted random color ID.
// Weights: gold 40%, amber 35%, terra 25%
func (e *ParticleEngine) spawnPetalColor() int {
	n := e.rng.Intn(100)
	switch {
	case n < 40:
		return 0 // gold
	case n < 75:
		return 1 // amber
	default:
		return 2 // terra
	}
}

// randFloat64Range returns a random float64 in [lo, hi).
func (e *ParticleEngine) randFloat64Range(lo, hi float64) float64 {
	return lo + e.rng.Float64()*(hi-lo)
}

// SpawnPetal creates a new petal at one of the given spawn points.
// spawnPoints are (col, row) pairs from the canopy edge.
func (e *ParticleEngine) SpawnPetal(spawnPoints [][2]int) {
	if len(e.Petals) >= e.maxPetals || len(spawnPoints) == 0 {
		return
	}

	pt := spawnPoints[e.rng.Intn(len(spawnPoints))]
	life := 50 + e.rng.Intn(31) // 50–80 frames

	p := Petal{
		X:       float64(pt[0]),
		Y:       float64(pt[1]),
		VX:      e.randFloat64Range(-0.3, 0.3),
		VY:      e.randFloat64Range(0.05, 0.2),
		Char:    e.spawnPetalChar(),
		ColorID: e.spawnPetalColor(),
		Life:    life,
		MaxLife: life,
	}
	e.Petals = append(e.Petals, p)
}

// SpawnFirefly creates a rising firefly near the base of the tree.
// baseX, baseY are the coordinates of the tree base/pot.
func (e *ParticleEngine) SpawnFirefly(baseX, baseY int) {
	if len(e.Fireflies) >= e.maxFireflies {
		return
	}

	life := 60 + e.rng.Intn(41) // 60–100 frames

	f := Firefly{
		X:     float64(baseX) + e.randFloat64Range(-2, 2),
		Y:     float64(baseY),
		VY:    e.randFloat64Range(-0.25, -0.15), // upward
		Phase: e.rng.Intn(16),                   // stagger blink phase
		Life:  life,
	}
	e.Fireflies = append(e.Fireflies, f)
}

// Update advances all particles by one frame.
// groundY is the y-coordinate of the ground line.
// windStrength is an additive vx force (0 = calm, positive = rightward).
func (e *ParticleEngine) Update(groundY int, windStrength float64) {
	// --- Update petals ---
	alive := e.Petals[:0]
	for i := range e.Petals {
		p := &e.Petals[i]

		// Physics
		p.VY += 0.12                                          // gravity
		p.VX += e.randFloat64Range(-0.08, 0.08) + windStrength // brownian + wind
		p.X += p.VX
		p.Y += p.VY
		p.Life--

		// Ground collision or life expired
		if p.Life <= 0 || p.Y >= float64(groundY) {
			// Convert to GroundPetal if there is room
			if len(e.Ground) < e.maxGround {
				gx := int(p.X)
				e.Ground = append(e.Ground, GroundPetal{
					X:       gx,
					Char:    p.Char,
					ColorID: p.ColorID,
					Life:    60,
				})
			}
			continue // do not keep in alive slice
		}

		alive = append(alive, *p)
	}
	e.Petals = alive

	// --- Update fireflies ---
	aliveFF := e.Fireflies[:0]
	for i := range e.Fireflies {
		f := &e.Fireflies[i]

		f.X += e.randFloat64Range(-0.1, 0.1) // horizontal wander
		f.Y += f.VY
		f.Phase++
		f.Life--

		if f.Life <= 0 {
			continue
		}
		aliveFF = append(aliveFF, *f)
	}
	e.Fireflies = aliveFF

	// --- Update ground petals ---
	aliveGP := e.Ground[:0]
	for i := range e.Ground {
		gp := &e.Ground[i]
		gp.Life--
		if gp.Life <= 0 {
			continue
		}
		aliveGP = append(aliveGP, *gp)
	}
	e.Ground = aliveGP
}

// ApplyWindGust adds a burst of wind to all active petals.
func (e *ParticleEngine) ApplyWindGust(strength float64) {
	for i := range e.Petals {
		e.Petals[i].VX += strength
	}
}

// ShakeBurst spawns a burst of petals from the canopy.
// count is how many to spawn; spawnPoints are canopy edge coords.
func (e *ParticleEngine) ShakeBurst(count int, spawnPoints [][2]int) {
	for i := 0; i < count; i++ {
		e.SpawnPetal(spawnPoints)
	}
}

// SpawnFireflyBurst creates multiple fireflies at once.
func (e *ParticleEngine) SpawnFireflyBurst(count, baseX, baseY int) {
	for i := 0; i < count; i++ {
		e.SpawnFirefly(baseX, baseY)
	}
}

// Reset clears all particles.
func (e *ParticleEngine) Reset() {
	e.Petals = e.Petals[:0]
	e.Fireflies = e.Fireflies[:0]
	e.Ground = e.Ground[:0]
}

// FireflyVisible reports whether the firefly should be rendered this frame.
// Visible when Phase % 16 < 10 (on 62.5% of the time).
func (f *Firefly) FireflyVisible() bool {
	return f.Phase%16 < 10
}

// FireflyChar returns the display character for a firefly.
func FireflyChar() rune {
	return '✦'
}

// FireflyColorHex returns the color for all fireflies.
func FireflyColorHex() string {
	return "#ffd166"
}
