// tanks.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	WIDTH      = 40
	HEIGHT     = 20
	TANK_SIZE  = 3
	BULLET_CHAR = "•"
	WALL_CHAR  = "█"
)

var EXPLOSION_CHARS = []string{"💥", "🔥", "💫"}
var BONUS_TYPES = []string{"❤️", "⚡", "🛡️"}

type Player struct {
	X, Y      int
	Dir       string
	HP        int
	Score     int
	Speed     float64
	Shield    bool
	ShieldTimer int
	Cooldown  int
	Alive     bool
}

type Enemy struct {
	X, Y      int
	Dir       string
	HP        int
	Speed     float64
	Alive     bool
	Cooldown  int
}

type Bullet struct {
	X, Y      float64
	Dir       string
	IsPlayer  bool
	Speed     float64
}

type Bonus struct {
	X, Y      int
	Type      string
	Active    bool
	Life      int
}

type Explosion struct {
	X, Y      float64
	Life      int
}

type Game struct {
	Width, Height int
	Player        Player
	Bullets       []Bullet
	Enemies       []Enemy
	Bonuses       []Bonus
	Explosions    []Explosion
	Walls         [][2]int
	Frame         int
	Level         int
	GameOver      bool
	Win           bool
	HighScore     int
	Keys          map[string]bool
	Running       bool
}

func NewGame() *Game {
	g := &Game{
		Width:  WIDTH,
		Height: HEIGHT,
		Keys:   make(map[string]bool),
		Running: true,
	}
	g.resetGame()
	g.HighScore = g.loadHighScore()
	return g
}

func (g *Game) resetGame() {
	g.Player = Player{
		X:      WIDTH / 2,
		Y:      HEIGHT - 3,
		Dir:    "up",
		HP:     3,
		Score:  0,
		Speed:  1.0,
		Shield: false,
		Alive:  true,
	}
	g.Bullets = []Bullet{}
	g.Enemies = []Enemy{}
	g.Bonuses = []Bonus{}
	g.Explosions = []Explosion{}
	g.Walls = g.generateWalls()
	g.Frame = 0
	g.Level = 1
	g.GameOver = false
	g.Win = false
}

func (g *Game) generateWalls() [][2]int {
	walls := [][2]int{}
	for x := 0; x < WIDTH; x++ {
		walls = append(walls, [2]int{x, 0}, [2]int{x, HEIGHT - 1})
	}
	for y := 0; y < HEIGHT; y++ {
		walls = append(walls, [2]int{0, y}, [2]int{WIDTH - 1, y})
	}
	for i := 0; i < 8; i++ {
		x := rand.Intn(WIDTH-4) + 2
		y := rand.Intn(HEIGHT-4) + 2
		for dx := 0; dx < 2; dx++ {
			for dy := 0; dy < 2; dy++ {
				wx, wy := x+dx, y+dy
				found := false
				for _, w := range walls {
					if w[0] == wx && w[1] == wy {
						found = true
						break
					}
				}
				if !found {
					walls = append(walls, [2]int{wx, wy})
				}
			}
		}
	}
	return walls
}

func (g *Game) loadHighScore() int {
	data, err := os.ReadFile("tanks_score.json")
	if err != nil {
		return 0
	}
	var score map[string]int
	if err := json.Unmarshal(data, &score); err != nil {
		return 0
	}
	return score["high_score"]
}

func (g *Game) saveHighScore() {
	data, _ := json.Marshal(map[string]int{"high_score": g.HighScore})
	os.WriteFile("tanks_score.json", data, 0644)
}

func (g *Game) isCollision(x, y int) bool {
	for dx := 0; dx < TANK_SIZE; dx++ {
		for dy := 0; dy < TANK_SIZE; dy++ {
			px, py := x+dx, y+dy
			if px < 0 || px >= WIDTH || py < 0 || py >= HEIGHT {
				return true
			}
			for _, w := range g.Walls {
				if w[0] == px && w[1] == py {
					return true
				}
			}
		}
	}
	for _, e := range g.Enemies {
		if e.Alive {
			for dx := 0; dx < TANK_SIZE; dx++ {
				for dy := 0; dy < TANK_SIZE; dy++ {
					ex, ey := e.X+dx, e.Y+dy
					for dx2 := 0; dx2 < TANK_SIZE; dx2++ {
						for dy2 := 0; dy2 < TANK_SIZE; dy2++ {
							if x+dx2 == ex && y+dy2 == ey {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

func (g *Game) spawnEnemy() {
	side := rand.Intn(4)
	var x, y int
	switch side {
	case 0:
		x = rand.Intn(WIDTH-TANK_SIZE-2) + 1
		y = 1
	case 1:
		x = rand.Intn(WIDTH-TANK_SIZE-2) + 1
		y = HEIGHT - TANK_SIZE - 1
	case 2:
		x = 1
		y = rand.Intn(HEIGHT-TANK_SIZE-2) + 1
	default:
		x = WIDTH - TANK_SIZE - 1
		y = rand.Intn(HEIGHT-TANK_SIZE-2) + 1
	}
	if !g.isCollision(x, y) {
		dirs := []string{"up", "down", "left", "right"}
		g.Enemies = append(g.Enemies, Enemy{
			X:        x,
			Y:        y,
			Dir:      dirs[rand.Intn(4)],
			HP:       1,
			Speed:    0.5 + float64(g.Level)*0.1,
			Alive:    true,
			Cooldown: rand.Intn(30),
		})
	}
}

func (g *Game) spawnBonus(x, y int) {
	if rand.Float64() < 0.15 {
		g.Bonuses = append(g.Bonuses, Bonus{
			X:      x,
			Y:      y,
			Type:   BONUS_TYPES[rand.Intn(len(BONUS_TYPES))],
			Active: true,
			Life:   100,
		})
	}
}

func (g *Game) shoot(x, y int, dir string, isPlayer bool) {
	g.Bullets = append(g.Bullets, Bullet{
		X:        float64(x + TANK_SIZE/2),
		Y:        float64(y + TANK_SIZE/2),
		Dir:      dir,
		IsPlayer: isPlayer,
		Speed:    2.0,
	})
	if !isPlayer {
		g.Bullets[len(g.Bullets)-1].Speed = 1.5
	}
}

func (g *Game) moveEnemy(e *Enemy) {
	if !e.Alive {
		return
	}
	dx := g.Player.X - e.X
	dy := g.Player.Y - e.Y
	if abs(dx) > abs(dy) {
		if dx > 0 && !g.isCollision(e.X+1, e.Y) {
			e.X++
			e.Dir = "right"
		} else if dx < 0 && !g.isCollision(e.X-1, e.Y) {
			e.X--
			e.Dir = "left"
		}
	} else {
		if dy > 0 && !g.isCollision(e.X, e.Y+1) {
			e.Y++
			e.Dir = "down"
		} else if dy < 0 && !g.isCollision(e.X, e.Y-1) {
			e.Y--
			e.Dir = "up"
		}
	}
}

func (g *Game) update() {
	if g.GameOver {
		return
	}

	g.Frame++

	// Player movement
	if g.Player.Alive {
		dx, dy := 0, 0
		if g.Keys["w"] || g.Keys["arrowup"] {
			dy = -1
			g.Player.Dir = "up"
		} else if g.Keys["s"] || g.Keys["arrowdown"] {
			dy = 1
			g.Player.Dir = "down"
		} else if g.Keys["a"] || g.Keys["arrowleft"] {
			dx = -1
			g.Player.Dir = "left"
		} else if g.Keys["d"] || g.Keys["arrowright"] {
			dx = 1
			g.Player.Dir = "right"
		}
		if dx != 0 || dy != 0 {
			nx, ny := g.Player.X+dx, g.Player.Y+dy
			if !g.isCollision(nx, ny) {
				g.Player.X = nx
				g.Player.Y = ny
			}
		}
		if g.Keys[" "] || g.Keys["enter"] {
			if g.Player.Cooldown <= 0 {
				g.shoot(g.Player.X, g.Player.Y, g.Player.Dir, true)
				g.Player.Cooldown = 5
			}
		}
	}
	if g.Player.Cooldown > 0 {
		g.Player.Cooldown--
	}

	// Enemy spawn
	spawnRate := 60 - g.Level*4
	if spawnRate < 20 {
		spawnRate = 20
	}
	alive := 0
	for _, e := range g.Enemies {
		if e.Alive {
			alive++
		}
	}
	if g.Frame%spawnRate == 0 && alive < 3+g.Level {
		g.spawnEnemy()
	}

	// Enemy movement and shooting
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if e.Alive {
			g.moveEnemy(e)
			e.Cooldown--
			if e.Cooldown <= 0 && rand.Float64() < 0.02+float64(g.Level)*0.005 {
				g.shoot(e.X, e.Y, e.Dir, false)
				e.Cooldown = 20 + rand.Intn(20)
			}
		}
	}

	// Bullets
	for i := len(g.Bullets) - 1; i >= 0; i-- {
		b := &g.Bullets[i]
		switch b.Dir {
		case "up":
			b.Y -= b.Speed
		case "down":
			b.Y += b.Speed
		case "left":
			b.X -= b.Speed
		case "right":
			b.X += b.Speed
		}
		bx, by := int(b.X), int(b.Y)
		hit := false

		if bx < 0 || bx >= WIDTH || by < 0 || by >= HEIGHT {
			g.Bullets = append(g.Bullets[:i], g.Bullets[i+1:]...)
			continue
		}

		wallHit := false
		for _, w := range g.Walls {
			if w[0] == bx && w[1] == by {
				wallHit = true
				break
			}
		}
		if wallHit {
			g.Bullets = append(g.Bullets[:i], g.Bullets[i+1:]...)
			g.Explosions = append(g.Explosions, Explosion{X: float64(bx), Y: float64(by), Life: 10})
			continue
		}

		if b.IsPlayer {
			for j := range g.Enemies {
				e := &g.Enemies[j]
				if e.Alive && e.X <= bx && bx < e.X+TANK_SIZE && e.Y <= by && by < e.Y+TANK_SIZE {
					e.HP--
					if e.HP <= 0 {
						e.Alive = false
						g.Player.Score += 10
						g.Explosions = append(g.Explosions, Explosion{X: float64(e.X + 1), Y: float64(e.Y + 1), Life: 15})
						g.spawnBonus(e.X, e.Y)
					}
					hit = true
					break
				}
			}
		} else {
			if g.Player.Alive && g.Player.X <= bx && bx < g.Player.X+TANK_SIZE &&
				g.Player.Y <= by && by < g.Player.Y+TANK_SIZE {
				if !g.Player.Shield {
					g.Player.HP--
					g.Explosions = append(g.Explosions, Explosion{X: float64(g.Player.X + 1), Y: float64(g.Player.Y + 1), Life: 20})
					if g.Player.HP <= 0 {
						g.Player.Alive = false
						g.GameOver = true
						if g.Player.Score > g.HighScore {
							g.HighScore = g.Player.Score
							g.saveHighScore()
						}
					}
				}
				hit = true
			}
		}
		if hit {
			g.Bullets = append(g.Bullets[:i], g.Bullets[i+1:]...)
		}
	}

	// Win condition
	if g.Player.Score >= 100+g.Level*20 {
		g.Win = true
		g.GameOver = true
	}

	// Bonuses
	for i := len(g.Bonuses) - 1; i >= 0; i-- {
		b := &g.Bonuses[i]
		if b.Active {
			if g.Player.X <= b.X && b.X < g.Player.X+TANK_SIZE &&
				g.Player.Y <= b.Y && b.Y < g.Player.Y+TANK_SIZE {
				switch b.Type {
				case "❤️":
					if g.Player.HP < 5 {
						g.Player.HP++
					}
				case "⚡":
					if g.Player.Speed < 3.0 {
						g.Player.Speed += 0.5
					}
				case "🛡️":
					g.Player.Shield = true
					g.Player.ShieldTimer = 120
				}
				g.Bonuses = append(g.Bonuses[:i], g.Bonuses[i+1:]...)
				continue
			}
			b.Life--
			if b.Life <= 0 {
				g.Bonuses = append(g.Bonuses[:i], g.Bonuses[i+1:]...)
			}
		}
	}

	// Shield timer
	if g.Player.Shield {
		g.Player.ShieldTimer--
		if g.Player.ShieldTimer <= 0 {
			g.Player.Shield = false
		}
	}

	// Explosions
	for i := len(g.Explosions) - 1; i >= 0; i-- {
		g.Explosions[i].Life--
		if g.Explosions[i].Life <= 0 {
			g.Explosions = append(g.Explosions[:i], g.Explosions[i+1:]...)
		}
	}
}

func (g *Game) draw() {
	clearScreen()
	grid := make([][]string, HEIGHT)
	for i := range grid {
		grid[i] = make([]string, WIDTH)
		for j := range grid[i] {
			grid[i][j] = " "
		}
	}

	for _, w := range g.Walls {
		x, y := w[0], w[1]
		if y >= 0 && y < HEIGHT && x >= 0 && x < WIDTH {
			grid[y][x] = "\033[37m" + WALL_CHAR + "\033[0m"
		}
	}

	for _, e := range g.Enemies {
		if e.Alive {
			for dy := 0; dy < TANK_SIZE; dy++ {
				for dx := 0; dx < TANK_SIZE; dx++ {
					ex, ey := e.X+dx, e.Y+dy
					if ey >= 0 && ey < HEIGHT && ex >= 0 && ex < WIDTH {
						if dx == 1 && dy == 1 {
							grid[ey][ex] = "\033[31m▼\033[0m"
						} else {
							grid[ey][ex] = "\033[31m█\033[0m"
						}
					}
				}
			}
		}
	}

	if g.Player.Alive {
		dirChars := map[string]string{"up": "▲", "down": "▼", "left": "◄", "right": "►"}
		ch := dirChars[g.Player.Dir]
		if ch == "" {
			ch = "▲"
		}
		for dy := 0; dy < TANK_SIZE; dy++ {
			for dx := 0; dx < TANK_SIZE; dx++ {
				px, py := g.Player.X+dx, g.Player.Y+dy
				if py >= 0 && py < HEIGHT && px >= 0 && px < WIDTH {
					if dx == 1 && dy == 1 {
						grid[py][px] = "\033[32m" + ch + "\033[0m"
					} else {
						grid[py][px] = "\033[32m█\033[0m"
					}
				}
			}
		}
	}

	for _, b := range g.Bullets {
		bx, by := int(b.X), int(b.Y)
		if by >= 0 && by < HEIGHT && bx >= 0 && bx < WIDTH {
			color := "\033[33m"
			if !b.IsPlayer {
				color = "\033[31m"
			}
			grid[by][bx] = color + BULLET_CHAR + "\033[0m"
		}
	}

	for _, b := range g.Bonuses {
		if b.Active && b.Y >= 0 && b.Y < HEIGHT && b.X >= 0 && b.X < WIDTH {
			grid[b.Y][b.X] = "\033[33m" + b.Type + "\033[0m"
		}
	}

	for _, e := range g.Explosions {
		ex, ey := int(e.X), int(e.Y)
		if ey >= 0 && ey < HEIGHT && ex >= 0 && ex < WIDTH {
			ch := EXPLOSION_CHARS[rand.Intn(len(EXPLOSION_CHARS))]
			grid[ey][ex] = "\033[31m" + ch + "\033[0m"
		}
	}

	fmt.Println("┌" + strings.Repeat("─", WIDTH) + "┐")
	for _, row := range grid {
		fmt.Println("│" + strings.Join(row, "") + "│")
	}
	fmt.Println("└" + strings.Repeat("─", WIDTH) + "┘")

	hpBar := strings.Repeat("█", g.Player.HP) + strings.Repeat("░", 5-g.Player.HP)
	shieldStr := ""
	if g.Player.Shield {
		shieldStr = "🛡️ "
	}
	fmt.Printf("HP: \033[32m%s\033[0m  Score: %d  Level: %d  %s Best: %d\n", hpBar, g.Player.Score, g.Level, shieldStr, g.HighScore)

	if g.GameOver {
		if g.Win {
			fmt.Println("\033[36m🎉 ПОБЕДА! Нажмите R для рестарта\033[0m")
		} else {
			fmt.Println("\033[31m💀 ИГРА ОКОНЧЕНА! Нажмите R для рестарта\033[0m")
		}
	}
}

func clearScreen() {
	cmd := exec.Command("clear")
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	rand.Seed(time.Now().UnixNano())
	game := NewGame()

	fmt.Println("\033[36mТанки 8-бит - Управление: WASD/Стрелки, Space - стрельба, Q - выход\033[0m")
	fmt.Println("Нажмите Enter для начала...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	go func() {
		for game.Running {
			var input string
			fmt.Scanln(&input)
			if input == "q" || input == "Q" {
				game.Running = false
				return
			}
			if (input == "r" || input == "R") && game.GameOver {
				game.resetGame()
			}
			if input == "w" || input == "W" {
				game.Keys["w"] = true
				go func() { time.Sleep(100 * time.Millisecond); game.Keys["w"] = false }()
			}
			if input == "s" || input == "S" {
				game.Keys["s"] = true
				go func() { time.Sleep(100 * time.Millisecond); game.Keys["s"] = false }()
			}
			if input == "a" || input == "A" {
				game.Keys["a"] = true
				go func() { time.Sleep(100 * time.Millisecond); game.Keys["a"] = false }()
			}
			if input == "d" || input == "D" {
				game.Keys["d"] = true
				go func() { time.Sleep(100 * time.Millisecond); game.Keys["d"] = false }()
			}
			if input == " " {
				game.Keys[" "] = true
				go func() { time.Sleep(100 * time.Millisecond); game.Keys[" "] = false }()
			}
		}
	}()

	ticker := time.NewTicker(50 * time.Millisecond)
	for game.Running {
		game.update()
		game.draw()
		<-ticker.C
	}
}
