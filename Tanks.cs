// Tanks.cs
using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;

class Tanks
{
    const int WIDTH = 40;
    const int HEIGHT = 20;
    const int TANK_SIZE = 3;
    const string BULLET_CHAR = "•";
    const string WALL_CHAR = "█";
    static readonly string[] EXPLOSION_CHARS = { "💥", "🔥", "💫" };
    static readonly string[] BONUS_TYPES = { "❤️", "⚡", "🛡️" };

    class Player
    {
        public int X, Y, HP, Score, Cooldown, ShieldTimer;
        public string Dir = "up";
        public double Speed = 1.0;
        public bool Shield = false, Alive = true;
    }

    class Enemy
    {
        public int X, Y, HP, Cooldown;
        public string Dir = "up";
        public double Speed = 0.5;
        public bool Alive = true;
    }

    class Bullet
    {
        public double X, Y, Speed;
        public string Dir;
        public bool IsPlayer;
    }

    class Bonus
    {
        public int X, Y, Life = 100;
        public string Type;
        public bool Active = true;
    }

    class Explosion
    {
        public double X, Y;
        public int Life;
    }

    Player player;
    List<Bullet> bullets = new List<Bullet>();
    List<Enemy> enemies = new List<Enemy>();
    List<Bonus> bonuses = new List<Bonus>();
    List<Explosion> explosions = new List<Explosion>();
    List<(int, int)> walls;
    int frame, level, highScore;
    bool gameOver, win, running = true;
    ConcurrentDictionary<string, bool> keys = new ConcurrentDictionary<string, bool>();
    Random rand = new Random();

    public Tanks()
    {
        resetGame();
        highScore = loadHighScore();
        Task.Run(inputLoop);
    }

    void resetGame()
    {
        player = new Player { X = WIDTH / 2, Y = HEIGHT - 3 };
        bullets.Clear();
        enemies.Clear();
        bonuses.Clear();
        explosions.Clear();
        walls = generateWalls();
        frame = 0;
        level = 1;
        gameOver = false;
        win = false;
    }

    List<(int, int)> generateWalls()
    {
        var walls = new List<(int, int)>();
        for (int x = 0; x < WIDTH; x++)
        {
            walls.Add((x, 0));
            walls.Add((x, HEIGHT - 1));
        }
        for (int y = 0; y < HEIGHT; y++)
        {
            walls.Add((0, y));
            walls.Add((WIDTH - 1, y));
        }
        for (int i = 0; i < 8; i++)
        {
            int x = rand.Next(2, WIDTH - 2);
            int y = rand.Next(2, HEIGHT - 2);
            for (int dx = 0; dx < 2; dx++)
                for (int dy = 0; dy < 2; dy++)
                {
                    int wx = x + dx, wy = y + dy;
                    if (!walls.Contains((wx, wy))) walls.Add((wx, wy));
                }
        }
        return walls;
    }

    int loadHighScore()
    {
        try
        {
            string json = File.ReadAllText("tanks_score.json");
            var doc = JsonDocument.Parse(json);
            return doc.RootElement.GetProperty("high_score").GetInt32();
        }
        catch { return 0; }
    }

    void saveHighScore()
    {
        try
        {
            var json = JsonSerializer.Serialize(new { high_score = highScore });
            File.WriteAllText("tanks_score.json", json);
        }
        catch { }
    }

    bool isCollision(int x, int y)
    {
        for (int dx = 0; dx < TANK_SIZE; dx++)
            for (int dy = 0; dy < TANK_SIZE; dy++)
            {
                int px = x + dx, py = y + dy;
                if (px < 0 || px >= WIDTH || py < 0 || py >= HEIGHT) return true;
                if (walls.Contains((px, py))) return true;
            }
        foreach (var e in enemies)
            if (e.Alive)
                for (int dx = 0; dx < TANK_SIZE; dx++)
                    for (int dy = 0; dy < TANK_SIZE; dy++)
                    {
                        int ex = e.X + dx, ey = e.Y + dy;
                        for (int dx2 = 0; dx2 < TANK_SIZE; dx2++)
                            for (int dy2 = 0; dy2 < TANK_SIZE; dy2++)
                                if (x + dx2 == ex && y + dy2 == ey) return true;
                    }
        return false;
    }

    void spawnEnemy()
    {
        int side = rand.Next(4);
        int x, y;
        switch (side)
        {
            case 0: x = rand.Next(1, WIDTH - TANK_SIZE - 1); y = 1; break;
            case 1: x = rand.Next(1, WIDTH - TANK_SIZE - 1); y = HEIGHT - TANK_SIZE - 1; break;
            case 2: x = 1; y = rand.Next(1, HEIGHT - TANK_SIZE - 1); break;
            default: x = WIDTH - TANK_SIZE - 1; y = rand.Next(1, HEIGHT - TANK_SIZE - 1); break;
        }
        if (!isCollision(x, y))
            enemies.Add(new Enemy { X = x, Y = y });
    }

    void spawnBonus(int x, int y)
    {
        if (rand.NextDouble() < 0.15)
            bonuses.Add(new Bonus { X = x, Y = y, Type = BONUS_TYPES[rand.Next(BONUS_TYPES.Length)] });
    }

    void shoot(int x, int y, string dir, bool isPlayer)
    {
        bullets.Add(new Bullet
        {
            X = x + TANK_SIZE / 2,
            Y = y + TANK_SIZE / 2,
            Dir = dir,
            IsPlayer = isPlayer,
            Speed = isPlayer ? 2.0 : 1.5
        });
    }

    public void update()
    {
        if (gameOver) return;

        frame++;

        // Player
        if (player.Alive)
        {
            int dx = 0, dy = 0;
            if (keys.ContainsKey("w") && keys["w"] || keys.ContainsKey("arrowup") && keys["arrowup"]) { dy = -1; player.Dir = "up"; }
            else if (keys.ContainsKey("s") && keys["s"] || keys.ContainsKey("arrowdown") && keys["arrowdown"]) { dy = 1; player.Dir = "down"; }
            else if (keys.ContainsKey("a") && keys["a"] || keys.ContainsKey("arrowleft") && keys["arrowleft"]) { dx = -1; player.Dir = "left"; }
            else if (keys.ContainsKey("d") && keys["d"] || keys.ContainsKey("arrowright") && keys["arrowright"]) { dx = 1; player.Dir = "right"; }
            if (dx != 0 || dy != 0)
            {
                int nx = player.X + dx;
                int ny = player.Y + dy;
                if (!isCollision(nx, ny)) { player.X = nx; player.Y = ny; }
            }
            if (keys.ContainsKey(" ") && keys[" "] || keys.ContainsKey("enter") && keys["enter"])
            {
                if (player.Cooldown <= 0)
                {
                    shoot(player.X, player.Y, player.Dir, true);
                    player.Cooldown = 5;
                }
            }
        }
        if (player.Cooldown > 0) player.Cooldown--;

        // Enemy spawn
        int spawnRate = Math.Max(20, 60 - level * 4);
        int alive = enemies.Count(e => e.Alive);
        if (frame % spawnRate == 0 && alive < 3 + level) spawnEnemy();

        // Enemies
        foreach (var e in enemies)
        {
            if (e.Alive)
            {
                int dx = Math.Sign(player.X - e.X);
                int dy = Math.Sign(player.Y - e.Y);
                if (Math.Abs(dx) > Math.Abs(dy))
                {
                    if (dx > 0 && !isCollision(e.X + 1, e.Y)) { e.X++; e.Dir = "right"; }
                    else if (dx < 0 && !isCollision(e.X - 1, e.Y)) { e.X--; e.Dir = "left"; }
                }
                else
                {
                    if (dy > 0 && !isCollision(e.X, e.Y + 1)) { e.Y++; e.Dir = "down"; }
                    else if (dy < 0 && !isCollision(e.X, e.Y - 1)) { e.Y--; e.Dir = "up"; }
                }
                e.Cooldown--;
                if (e.Cooldown <= 0 && rand.NextDouble() < 0.02 + level * 0.005)
                {
                    shoot(e.X, e.Y, e.Dir, false);
                    e.Cooldown = 20 + rand.Next(20);
                }
            }
        }

        // Bullets
        for (int i = bullets.Count - 1; i >= 0; i--)
        {
            var b = bullets[i];
            switch (b.Dir)
            {
                case "up": b.Y -= b.Speed; break;
                case "down": b.Y += b.Speed; break;
                case "left": b.X -= b.Speed; break;
                case "right": b.X += b.Speed; break;
            }
            int bx = (int)Math.Round(b.X);
            int by = (int)Math.Round(b.Y);
            bool hit = false;

            if (bx < 0 || bx >= WIDTH || by < 0 || by >= HEIGHT) { bullets.RemoveAt(i); continue; }

            if (walls.Contains((bx, by)))
            {
                bullets.RemoveAt(i);
                explosions.Add(new Explosion { X = bx, Y = by, Life = 10 });
                continue;
            }

            if (b.IsPlayer)
            {
                foreach (var e in enemies)
                {
                    if (e.Alive && e.X <= bx && bx < e.X + TANK_SIZE && e.Y <= by && by < e.Y + TANK_SIZE)
                    {
                        e.HP--;
                        if (e.HP <= 0)
                        {
                            e.Alive = false;
                            player.Score += 10;
                            explosions.Add(new Explosion { X = e.X + 1, Y = e.Y + 1, Life = 15 });
                            spawnBonus(e.X, e.Y);
                        }
                        hit = true;
                        break;
                    }
                }
            }
            else
            {
                if (player.Alive && player.X <= bx && bx < player.X + TANK_SIZE &&
                    player.Y <= by && by < player.Y + TANK_SIZE)
                {
                    if (!player.Shield)
                    {
                        player.HP--;
                        explosions.Add(new Explosion { X = player.X + 1, Y = player.Y + 1, Life = 20 });
                        if (player.HP <= 0)
                        {
                            player.Alive = false;
                            gameOver = true;
                            if (player.Score > highScore) { highScore = player.Score; saveHighScore(); }
                        }
                    }
                    hit = true;
                }
            }
            if (hit) bullets.RemoveAt(i);
        }

        // Win
        if (player.Score >= 100 + level * 20) { win = true; gameOver = true; }

        // Bonuses
        for (int i = bonuses.Count - 1; i >= 0; i--)
        {
            var b = bonuses[i];
            if (b.Active)
            {
                if (player.X <= b.X && b.X < player.X + TANK_SIZE &&
                    player.Y <= b.Y && b.Y < player.Y + TANK_SIZE)
                {
                    switch (b.Type)
                    {
                        case "❤️": player.HP = Math.Min(5, player.HP + 1); break;
                        case "⚡": player.Speed = Math.Min(3.0, player.Speed + 0.5); break;
                        case "🛡️": player.Shield = true; player.ShieldTimer = 120; break;
                    }
                    bonuses.RemoveAt(i);
                    continue;
                }
                b.Life--;
                if (b.Life <= 0) bonuses.RemoveAt(i);
            }
        }

        // Shield
        if (player.Shield)
        {
            player.ShieldTimer--;
            if (player.ShieldTimer <= 0) player.Shield = false;
        }

        // Explosions
        for (int i = explosions.Count - 1; i >= 0; i--)
        {
            explosions[i].Life--;
            if (explosions[i].Life <= 0) explosions.RemoveAt(i);
        }
    }

    public void draw()
    {
        Console.Clear();
        string[][] grid = new string[HEIGHT][];
        for (int i = 0; i < HEIGHT; i++) grid[i] = new string[WIDTH];

        for (int i = 0; i < HEIGHT; i++)
            for (int j = 0; j < WIDTH; j++)
                grid[i][j] = " ";

        foreach (var w in walls)
        {
            int x = w.Item1, y = w.Item2;
            if (y >= 0 && y < HEIGHT && x >= 0 && x < WIDTH)
                grid[y][x] = "\x1b[37m" + WALL_CHAR + "\x1b[0m";
        }

        foreach (var e in enemies)
            if (e.Alive)
                for (int dy = 0; dy < TANK_SIZE; dy++)
                    for (int dx = 0; dx < TANK_SIZE; dx++)
                    {
                        int ex = e.X + dx, ey = e.Y + dy;
                        if (ey >= 0 && ey < HEIGHT && ex >= 0 && ex < WIDTH)
                            grid[ey][ex] = (dx == 1 && dy == 1) ? "\x1b[31m▼\x1b[0m" : "\x1b[31m█\x1b[0m";
                    }

        if (player.Alive)
        {
            string ch = player.Dir switch { "up" => "▲", "down" => "▼", "left" => "◄", _ => "►" };
            for (int dy = 0; dy < TANK_SIZE; dy++)
                for (int dx = 0; dx < TANK_SIZE; dx++)
                {
                    int px = player.X + dx, py = player.Y + dy;
                    if (py >= 0 && py < HEIGHT && px >= 0 && px < WIDTH)
                        grid[py][px] = (dx == 1 && dy == 1) ? "\x1b[32m" + ch + "\x1b[0m" : "\x1b[32m█\x1b[0m";
                }
        }

        foreach (var b in bullets)
        {
            int bx = (int)Math.Round(b.X), by = (int)Math.Round(b.Y);
            if (by >= 0 && by < HEIGHT && bx >= 0 && bx < WIDTH)
                grid[by][bx] = (b.IsPlayer ? "\x1b[33m" : "\x1b[31m") + BULLET_CHAR + "\x1b[0m";
        }

        foreach (var b in bonuses)
            if (b.Active && b.Y >= 0 && b.Y < HEIGHT && b.X >= 0 && b.X < WIDTH)
                grid[b.Y][b.X] = "\x1b[33m" + b.Type + "\x1b[0m";

        foreach (var e in explosions)
        {
            int ex = (int)Math.Round(e.X), ey = (int)Math.Round(e.Y);
            if (ey >= 0 && ey < HEIGHT && ex >= 0 && ex < WIDTH)
                grid[ey][ex] = "\x1b[31m" + EXPLOSION_CHARS[rand.Next(EXPLOSION_CHARS.Length)] + "\x1b[0m";
        }

        Console.WriteLine("┌" + new string('─', WIDTH) + "┐");
        foreach (var row in grid) Console.WriteLine("│" + string.Join("", row) + "│");
        Console.WriteLine("└" + new string('─', WIDTH) + "┘");

        string hpBar = new string('█', player.HP) + new string('░', 5 - player.HP);
        string shieldStr = player.Shield ? "🛡️ " : "";
        Console.WriteLine($"HP: \x1b[32m{hpBar}\x1b[0m  Score: {player.Score}  Level: {level}  {shieldStr} Best: {highScore}");

        if (gameOver)
            Console.WriteLine(win ? "\x1b[36m🎉 ПОБЕДА! Нажмите R для рестарта\x1b[0m" : "\x1b[31m💀 ИГРА ОКОНЧЕНА! Нажмите R для рестарта\x1b[0m");
    }

    void inputLoop()
    {
        while (running)
        {
            if (Console.KeyAvailable)
            {
                var key = Console.ReadKey(true).Key;
                if (key == ConsoleKey.Q) { running = false; Environment.Exit(0); }
                if ((key == ConsoleKey.R) && gameOver) resetGame();
                string k = key switch
                {
                    ConsoleKey.W => "w",
                    ConsoleKey.S => "s",
                    ConsoleKey.A => "a",
                    ConsoleKey.D => "d",
                    ConsoleKey.UpArrow => "arrowup",
                    ConsoleKey.DownArrow => "arrowdown",
                    ConsoleKey.LeftArrow => "arrowleft",
                    ConsoleKey.RightArrow => "arrowright",
                    ConsoleKey.Spacebar => " ",
                    ConsoleKey.Enter => "enter",
                    _ => ""
                };
                if (!string.IsNullOrEmpty(k))
                {
                    keys[k] = true;
                    Task.Delay(100).ContinueWith(_ => { keys.TryRemove(k, out _); });
                }
            }
            Thread.Sleep(10);
        }
    }

    public void run()
    {
        Console.WriteLine("\x1b[36mТанки 8-бит - Управление: WASD/Стрелки, Space - стрельба, Q - выход\x1b[0m");
        Console.WriteLine("Нажмите Enter для начала...");
        Console.ReadLine();

        while (running)
        {
            update();
            draw();
            Thread.Sleep(50);
        }
    }

    public static void Main()
    {
        var game = new Tanks();
        game.run();
    }
}
