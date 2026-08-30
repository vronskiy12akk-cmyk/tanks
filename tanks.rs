// tanks.rs
use std::collections::HashMap;
use std::fs;
use std::io::{self, Write, Read};
use std::time::{Duration, Instant};
use rand::Rng;
use colored::*;

const WIDTH: usize = 40;
const HEIGHT: usize = 20;
const TANK_SIZE: usize = 3;
const BULLET_CHAR: &str = "•";
const WALL_CHAR: &str = "█";
const EXPLOSION_CHARS: [&str; 3] = ["💥", "🔥", "💫"];
const BONUS_TYPES: [&str; 3] = ["❤️", "⚡", "🛡️"];

struct Player {
    x: usize,
    y: usize,
    dir: String,
    hp: i32,
    score: i32,
    speed: f64,
    shield: bool,
    shield_timer: i32,
    cooldown: i32,
    alive: bool,
}

struct Enemy {
    x: usize,
    y: usize,
    dir: String,
    hp: i32,
    speed: f64,
    alive: bool,
    cooldown: i32,
}

struct Bullet {
    x: f64,
    y: f64,
    dir: String,
    is_player: bool,
    speed: f64,
}

struct Bonus {
    x: usize,
    y: usize,
    typ: String,
    active: bool,
    life: i32,
}

struct Explosion {
    x: f64,
    y: f64,
    life: i32,
}

struct Game {
    width: usize,
    height: usize,
    player: Player,
    bullets: Vec<Bullet>,
    enemies: Vec<Enemy>,
    bonuses: Vec<Bonus>,
    explosions: Vec<Explosion>,
    walls: Vec<(usize, usize)>,
    frame: i32,
    level: i32,
    game_over: bool,
    win: bool,
    high_score: i32,
    keys: HashMap<String, bool>,
    running: bool,
}

impl Game {
    fn new() -> Self {
        let mut game = Game {
            width: WIDTH,
            height: HEIGHT,
            player: Player {
                x: WIDTH / 2,
                y: HEIGHT - 3,
                dir: "up".to_string(),
                hp: 3,
                score: 0,
                speed: 1.0,
                shield: false,
                shield_timer: 0,
                cooldown: 0,
                alive: true,
            },
            bullets: Vec::new(),
            enemies: Vec::new(),
            bonuses: Vec::new(),
            explosions: Vec::new(),
            walls: Vec::new(),
            frame: 0,
            level: 1,
            game_over: false,
            win: false,
            high_score: 0,
            keys: HashMap::new(),
            running: true,
        };
        game.walls = game.generate_walls();
        game.high_score = game.load_high_score();
        game
    }

    fn generate_walls(&self) -> Vec<(usize, usize)> {
        let mut walls = Vec::new();
        for x in 0..WIDTH {
            walls.push((x, 0));
            walls.push((x, HEIGHT - 1));
        }
        for y in 0..HEIGHT {
            walls.push((0, y));
            walls.push((WIDTH - 1, y));
        }
        let mut rng = rand::thread_rng();
        for _ in 0..8 {
            let x = rng.gen_range(2..WIDTH - 2);
            let y = rng.gen_range(2..HEIGHT - 2);
            for dx in 0..2 {
                for dy in 0..2 {
                    let wx = x + dx;
                    let wy = y + dy;
                    if !walls.contains(&(wx, wy)) {
                        walls.push((wx, wy));
                    }
                }
            }
        }
        walls
    }

    fn load_high_score(&self) -> i32 {
        if let Ok(data) = fs::read_to_string("tanks_score.json") {
            if let Ok(json) = serde_json::from_str::<serde_json::Value>(&data) {
                return json.get("high_score").and_then(|v| v.as_i64()).unwrap_or(0) as i32;
            }
        }
        0
    }

    fn save_high_score(&self) {
        let data = serde_json::json!({ "high_score": self.high_score });
        let _ = fs::write("tanks_score.json", serde_json::to_string_pretty(&data).unwrap());
    }

    fn is_collision(&self, x: usize, y: usize) -> bool {
        for dx in 0..TANK_SIZE {
            for dy in 0..TANK_SIZE {
                let px = x + dx;
                let py = y + dy;
                if px >= self.width || py >= self.height { return true; }
                if self.walls.contains(&(px, py)) { return true; }
            }
        }
        for e in &self.enemies {
            if e.alive {
                for dx in 0..TANK_SIZE {
                    for dy in 0..TANK_SIZE {
                        let ex = e.x + dx;
                        let ey = e.y + dy;
                        for dx2 in 0..TANK_SIZE {
                            for dy2 in 0..TANK_SIZE {
                                if x + dx2 == ex && y + dy2 == ey { return true; }
                            }
                        }
                    }
                }
            }
        }
        false
    }

    fn spawn_enemy(&mut self) {
        let mut rng = rand::thread_rng();
        let side = rng.gen_range(0..4);
        let (x, y) = match side {
            0 => (rng.gen_range(1..WIDTH - TANK_SIZE - 1), 1),
            1 => (rng.gen_range(1..WIDTH - TANK_SIZE - 1), HEIGHT - TANK_SIZE - 1),
            2 => (1, rng.gen_range(1..HEIGHT - TANK_SIZE - 1)),
            _ => (WIDTH - TANK_SIZE - 1, rng.gen_range(1..HEIGHT - TANK_SIZE - 1)),
        };
        if !self.is_collision(x, y) {
            let dirs = ["up", "down", "left", "right"];
            self.enemies.push(Enemy {
                x,
                y,
                dir: dirs[rng.gen_range(0..4)].to_string(),
                hp: 1,
                speed: 0.5 + self.level as f64 * 0.1,
                alive: true,
                cooldown: rng.gen_range(0..30),
            });
        }
    }

    fn spawn_bonus(&mut self, x: usize, y: usize) {
        let mut rng = rand::thread_rng();
        if rng.gen_bool(0.15) {
            self.bonuses.push(Bonus {
                x,
                y,
                typ: BONUS_TYPES[rng.gen_range(0..BONUS_TYPES.len())].to_string(),
                active: true,
                life: 100,
            });
        }
    }

    fn shoot(&mut self, x: usize, y: usize, dir: &str, is_player: bool) {
        let speed = if is_player { 2.0 } else { 1.5 };
        self.bullets.push(Bullet {
            x: (x + TANK_SIZE / 2) as f64,
            y: (y + TANK_SIZE / 2) as f64,
            dir: dir.to_string(),
            is_player,
            speed,
        });
    }

    fn update(&mut self) {
        if self.game_over { return; }

        self.frame += 1;

        // Player movement
        if self.player.alive {
            let mut dx = 0;
            let mut dy = 0;
            if self.keys.get("w").unwrap_or(&false) || self.keys.get("arrowup").unwrap_or(&false) {
                dy = -1;
                self.player.dir = "up".to_string();
            } else if self.keys.get("s").unwrap_or(&false) || self.keys.get("arrowdown").unwrap_or(&false) {
                dy = 1;
                self.player.dir = "down".to_string();
            } else if self.keys.get("a").unwrap_or(&false) || self.keys.get("arrowleft").unwrap_or(&false) {
                dx = -1;
                self.player.dir = "left".to_string();
            } else if self.keys.get("d").unwrap_or(&false) || self.keys.get("arrowright").unwrap_or(&false) {
                dx = 1;
                self.player.dir = "right".to_string();
            }
            if dx != 0 || dy != 0 {
                let nx = if dx < 0 && self.player.x > 0 { self.player.x - 1 } else { self.player.x + dx as usize };
                let ny = if dy < 0 && self.player.y > 0 { self.player.y - 1 } else { self.player.y + dy as usize };
                if !self.is_collision(nx, ny) {
                    self.player.x = nx;
                    self.player.y = ny;
                }
            }
            if self.keys.get(" ").unwrap_or(&false) || self.keys.get("enter").unwrap_or(&false) {
                if self.player.cooldown <= 0 {
                    self.shoot(self.player.x, self.player.y, &self.player.dir, true);
                    self.player.cooldown = 5;
                }
            }
        }
        self.player.cooldown = (self.player.cooldown - 1).max(0);

        // Enemy spawn
        let spawn_rate = (60 - self.level * 4).max(20);
        let alive_count = self.enemies.iter().filter(|e| e.alive).count();
        if self.frame % spawn_rate == 0 && alive_count < 3 + self.level as usize {
            self.spawn_enemy();
        }

        // Enemy movement and shooting
        for e in &mut self.enemies {
            if e.alive {
                let dx = if self.player.x > e.x { 1 } else if self.player.x < e.x { -1 } else { 0 };
                let dy = if self.player.y > e.y { 1 } else if self.player.y < e.y { -1 } else { 0 };
                if dx.abs() > dy.abs() {
                    if dx > 0 && !self.is_collision(e.x + 1, e.y) { e.x += 1; e.dir = "right".to_string(); }
                    else if dx < 0 && !self.is_collision(e.x - 1, e.y) { e.x -= 1; e.dir = "left".to_string(); }
                } else {
                    if dy > 0 && !self.is_collision(e.x, e.y + 1) { e.y += 1; e.dir = "down".to_string(); }
                    else if dy < 0 && !self.is_collision(e.x, e.y - 1) { e.y -= 1; e.dir = "up".to_string(); }
                }
                e.cooldown -= 1;
                if e.cooldown <= 0 && rand::thread_rng().gen_bool(0.02 + self.level as f64 * 0.005) {
                    self.shoot(e.x, e.y, &e.dir, false);
                    e.cooldown = 20 + rand::thread_rng().gen_range(0..20);
                }
            }
        }

        // Bullets
        let mut i = 0;
        while i < self.bullets.len() {
            let b = &mut self.bullets[i];
            match b.dir.as_str() {
                "up" => b.y -= b.speed,
                "down" => b.y += b.speed,
                "left" => b.x -= b.speed,
                _ => b.x += b.speed,
            }
            let bx = b.x as usize;
            let by = b.y as usize;
            let mut hit = false;

            if bx >= self.width || by >= self.height {
                self.bullets.remove(i);
                continue;
            }

            if self.walls.contains(&(bx, by)) {
                self.bullets.remove(i);
                self.explosions.push(Explosion { x: bx as f64, y: by as f64, life: 10 });
                continue;
            }

            if b.is_player {
                for e in &mut self.enemies {
                    if e.alive && e.x <= bx && bx < e.x + TANK_SIZE && e.y <= by && by < e.y + TANK_SIZE {
                        e.hp -= 1;
                        if e.hp <= 0 {
                            e.alive = false;
                            self.player.score += 10;
                            self.explosions.push(Explosion { x: (e.x + 1) as f64, y: (e.y + 1) as f64, life: 15 });
                            self.spawn_bonus(e.x, e.y);
                        }
                        hit = true;
                        break;
                    }
                }
            } else {
                if self.player.alive &&
                    self.player.x <= bx && bx < self.player.x + TANK_SIZE &&
                    self.player.y <= by && by < self.player.y + TANK_SIZE {
                    if !self.player.shield {
                        self.player.hp -= 1;
                        self.explosions.push(Explosion { x: (self.player.x + 1) as f64, y: (self.player.y + 1) as f64, life: 20 });
                        if self.player.hp <= 0 {
                            self.player.alive = false;
                            self.game_over = true;
                            if self.player.score > self.high_score {
                                self.high_score = self.player.score;
                                self.save_high_score();
                            }
                        }
                    }
                    hit = true;
                }
            }

            if hit {
                self.bullets.remove(i);
            } else {
                i += 1;
            }
        }

        // Win condition
        if self.player.score >= 100 + self.level * 20 {
            self.win = true;
            self.game_over = true;
        }

        // Bonuses
        let mut i = 0;
        while i < self.bonuses.len() {
            let b = &mut self.bonuses[i];
            if b.active {
                if self.player.x <= b.x && b.x < self.player.x + TANK_SIZE &&
                    self.player.y <= b.y && b.y < self.player.y + TANK_SIZE {
                    match b.typ.as_str() {
                        "❤️" => { self.player.hp = (self.player.hp + 1).min(5); }
                        "⚡" => { self.player.speed = (self.player.speed + 0.5).min(3.0); }
                        "🛡️" => { self.player.shield = true; self.player.shield_timer = 120; }
                        _ => {}
                    }
                    self.bonuses.remove(i);
                    continue;
                }
                b.life -= 1;
                if b.life <= 0 {
                    self.bonuses.remove(i);
                    continue;
                }
            }
            i += 1;
        }

        // Shield timer
        if self.player.shield {
            self.player.shield_timer -= 1;
            if self.player.shield_timer <= 0 {
                self.player.shield = false;
            }
        }

        // Explosions
        let mut i = 0;
        while i < self.explosions.len() {
            self.explosions[i].life -= 1;
            if self.explosions[i].life <= 0 {
                self.explosions.remove(i);
            } else {
                i += 1;
            }
        }
    }

    fn draw(&self) {
        print!("\x1B[2J\x1B[1;1H");
        let mut grid = vec![vec![" ".to_string(); self.width]; self.height];

        for (x, y) in &self.walls {
            if *y < self.height && *x < self.width {
                grid[*y][*x] = format!("{}{}{}", " ".white().on_white(), WALL_CHAR, " ".white().on_white());
            }
        }

        for e in &self.enemies {
            if e.alive {
                for dy in 0..TANK_SIZE {
                    for dx in 0..TANK_SIZE {
                        let ex = e.x + dx;
                        let ey = e.y + dy;
                        if ey < self.height && ex < self.width {
                            grid[ey][ex] = if dx == 1 && dy == 1 {
                                "▼".red().to_string()
                            } else {
                                "█".red().to_string()
                            };
                        }
                    }
                }
            }
        }

        if self.player.alive {
            let dir_chars = std::collections::HashMap::from([
                ("up", "▲"), ("down", "▼"), ("left", "◄"), ("right", "►")
            ]);
            let ch = dir_chars.get(self.player.dir.as_str()).unwrap_or(&"▲");
            for dy in 0..TANK_SIZE {
                for dx in 0..TANK_SIZE {
                    let px = self.player.x + dx;
                    let py = self.player.y + dy;
                    if py < self.height && px < self.width {
                        grid[py][px] = if dx == 1 && dy == 1 {
                            ch.green().to_string()
                        } else {
                            "█".green().to_string()
                        };
                    }
                }
            }
        }

        for b in &self.bullets {
            let bx = b.x as usize;
            let by = b.y as usize;
            if by < self.height && bx < self.width {
                let color = if b.is_player { "yellow" } else { "red" };
                grid[by][bx] = BULLET_CHAR.color(color).to_string();
            }
        }

        for b in &self.bonuses {
            if b.active && b.y < self.height && b.x < self.width {
                grid[b.y][b.x] = b.typ.yellow().to_string();
            }
        }

        for e in &self.explosions {
            let ex = e.x as usize;
            let ey = e.y as usize;
            if ey < self.height && ex < self.width {
                let ch = EXPLOSION_CHARS[rand::thread_rng().gen_range(0..EXPLOSION_CHARS.len())];
                grid[ey][ex] = ch.red().to_string();
            }
        }

        println!("┌{}┐", "─".repeat(self.width));
        for row in grid {
            println!("│{}│", row.join(""));
        }
        println!("└{}┘", "─".repeat(self.width));

        let hp_bar = "█".repeat(self.player.hp as usize) + &"░".repeat((5 - self.player.hp) as usize);
        let shield_str = if self.player.shield { "🛡️ " } else { "" };
        println!("HP: {}  Score: {}  Level: {}  {} Best: {}",
            hp_bar.green(), self.player.score, self.level, shield_str, self.high_score);

        if self.game_over {
            if self.win {
                println!("{}", "🎉 ПОБЕДА! Нажмите R для рестарта".cyan());
            } else {
                println!("{}", "💀 ИГРА ОКОНЧЕНА! Нажмите R для рестарта".red());
            }
        }
    }
}

fn main() {
    let mut game = Game::new();
    println!("{}", "Танки 8-бит - Управление: WASD/Стрелки, Space - стрельба, Q - выход".cyan());
    println!("Нажмите Enter для начала...");
    let mut input = String::new();
    io::stdin().read_line(&mut input).unwrap();

    // Set up terminal for raw input
    let _ = termion::raw::into_raw_mode(&mut io::stdout()).unwrap();
    let mut stdout = io::stdout();
    let mut stdin = io::stdin();

    // Key handling in separate thread
    let keys = game.keys.clone();
    std::thread::spawn(move || {
        for c in stdin.bytes() {
            if let Ok(b) = c {
                let ch = b as char;
                if ch == 'q' || ch == 'Q' {
                    std::process::exit(0);
                }
                if (ch == 'r' || ch == 'R') && game.game_over {
                    game.reset_game();
                }
                // Handle keys
                let key = match ch {
                    'w' | 'W' => "w",
                    's' | 'S' => "s",
                    'a' | 'A' => "a",
                    'd' | 'D' => "d",
                    ' ' => " ",
                    '\n' => "enter",
                    _ => "",
                };
                if !key.is_empty() {
                    game.keys.insert(key.to_string(), true);
                    std::thread::sleep(std::time::Duration::from_millis(100));
                    game.keys.insert(key.to_string(), false);
                }
            }
        }
    });

    let frame_duration = Duration::from_millis(50);
    while game.running {
        let start = Instant::now();
        game.update();
        game.draw();
        let elapsed = start.elapsed();
        if elapsed < frame_duration {
            std::thread::sleep(frame_duration - elapsed);
        }
    }
}
