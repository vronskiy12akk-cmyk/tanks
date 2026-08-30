// Tanks.java
import java.io.*;
import java.nio.file.*;
import java.util.*;
import java.util.concurrent.*;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonObject;

public class Tanks {
    private static final int WIDTH = 40;
    private static final int HEIGHT = 20;
    private static final int TANK_SIZE = 3;
    private static final String BULLET_CHAR = "•";
    private static final String WALL_CHAR = "█";
    private static final String[] EXPLOSION_CHARS = {"💥", "🔥", "💫"};
    private static final String[] BONUS_TYPES = {"❤️", "⚡", "🛡️"};

    static class Player {
        int x, y, hp, score, cooldown, shieldTimer;
        String dir;
        double speed;
        boolean shield, alive;

        Player() {
            x = WIDTH / 2;
            y = HEIGHT - 3;
            dir = "up";
            hp = 3;
            score = 0;
            speed = 1.0;
            shield = false;
            shieldTimer = 0;
            cooldown = 0;
            alive = true;
        }
    }

    static class Enemy {
        int x, y, hp, cooldown;
        String dir;
        double speed;
        boolean alive;

        Enemy(int x, int y) {
            this.x = x;
            this.y = y;
            this.dir = "up";
            this.hp = 1;
            this.speed = 0.5;
            this.alive = true;
            this.cooldown = 0;
        }
    }

    static class Bullet {
        double x, y, speed;
        String dir;
        boolean isPlayer;

        Bullet(double x, double y, String dir, boolean isPlayer, double speed) {
            this.x = x;
            this.y = y;
            this.dir = dir;
            this.isPlayer = isPlayer;
            this.speed = speed;
        }
    }

    static class Bonus {
        int x, y, life;
        String type;
        boolean active;

        Bonus(int x, int y, String type) {
            this.x = x;
            this.y = y;
            this.type = type;
            this.active = true;
            this.life = 100;
        }
    }

    static class Explosion {
        double x, y;
        int life;

        Explosion(double x, double y, int life) {
            this.x = x;
            this.y = y;
            this.life = life;
        }
    }

    private Player player;
    private List<Bullet> bullets;
    private List<Enemy> enemies;
    private List<Bonus> bonuses;
    private List<Explosion> explosions;
    private List<int[]> walls;
    private int frame, level, highScore;
    private boolean gameOver, win, running;
    private Map<String, Boolean> keys;
    private Scanner scanner;

    public Tanks() {
        scanner = new Scanner(System.in);
        resetGame();
        highScore = loadHighScore();
        running = true;
        keys = new ConcurrentHashMap<>();
        startInputThread();
    }

    private void resetGame() {
        player = new Player();
        bullets = new CopyOnWriteArrayList<>();
        enemies = new CopyOnWriteArrayList<>();
        bonuses = new CopyOnWriteArrayList<>();
        explosions = new CopyOnWriteArrayList<>();
        walls = generateWalls();
        frame = 0;
        level = 1;
        gameOver = false;
        win = false;
    }

    private List<int[]> generateWalls() {
        List<int[]> walls = new ArrayList<>();
        for (int x = 0; x < WIDTH; x++) {
            walls.add(new int[]{x, 0});
            walls.add(new int[]{x, HEIGHT - 1});
        }
        for (int y = 0; y < HEIGHT; y++) {
            walls.add(new int[]{0, y});
            walls.add(new int[]{WIDTH - 1, y});
        }
        Random rand = new Random();
        for (int i = 0; i < 8; i++) {
            int x = rand.nextInt(WIDTH - 4) + 2;
            int y = rand.nextInt(HEIGHT - 4) + 2;
            for (int dx = 0; dx < 2; dx++) {
                for (int dy = 0; dy < 2; dy++) {
                    int wx = x + dx, wy = y + dy;
                    boolean found = false;
                    for (int[] w : walls) {
                        if (w[0] == wx && w[1] == wy) { found = true; break; }
                    }
                    if (!found) walls.add(new int[]{wx, wy});
                }
            }
        }
        return walls;
    }

    private int loadHighScore() {
        try {
            String json = new String(Files.readAllBytes(Paths.get("tanks_score.json")));
            JsonObject obj = new Gson().fromJson(json, JsonObject.class);
            return obj.get("high_score").getAsInt();
        } catch (Exception e) {
            return 0;
        }
    }

    private void saveHighScore() {
        try {
            JsonObject obj = new JsonObject();
            obj.addProperty("high_score", highScore);
            Files.write(Paths.get("tanks_score.json"), new GsonBuilder().setPrettyPrinting().create().toJson(obj).getBytes());
        } catch (Exception e) {}
    }

    private boolean isCollision(int x, int y) {
        for (int dx = 0; dx < TANK_SIZE; dx++) {
            for (int dy = 0; dy < TANK_SIZE; dy++) {
                int px = x + dx, py = y + dy;
                if (px < 0 || px >= WIDTH || py < 0 || py >= HEIGHT) return true;
                for (int[] w : walls) {
                    if (w[0] == px && w[1] == py) return true;
                }
            }
        }
        for (Enemy e : enemies) {
            if (e.alive) {
                for (int dx = 0; dx < TANK_SIZE; dx++) {
                    for (int dy = 0; dy < TANK_SIZE; dy++) {
                        int ex = e.x + dx, ey = e.y + dy;
                        for (int dx2 = 0; dx2 < TANK_SIZE; dx2++) {
                            for (int dy2 = 0; dy2 < TANK_SIZE; dy2++) {
                                if (x + dx2 == ex && y + dy2 == ey) return true;
                            }
                        }
                    }
                }
            }
        }
        return false;
    }

    private void spawnEnemy() {
        Random rand = new Random();
        int side = rand.nextInt(4);
        int x, y;
        switch (side) {
            case 0: x = rand.nextInt(WIDTH - TANK_SIZE - 2) + 1; y = 1; break;
            case 1: x = rand.nextInt(WIDTH - TANK_SIZE - 2) + 1; y = HEIGHT - TANK_SIZE - 1; break;
            case 2: x = 1; y = rand.nextInt(HEIGHT - TANK_SIZE - 2) + 1; break;
            default: x = WIDTH - TANK_SIZE - 1; y = rand.nextInt(HEIGHT - TANK_SIZE - 2) + 1; break;
        }
        if (!isCollision(x, y)) {
            enemies.add(new Enemy(x, y));
        }
    }

    private void spawnBonus(int x, int y) {
        Random rand = new Random();
        if (rand.nextDouble() < 0.15) {
            bonuses.add(new Bonus(x, y, BONUS_TYPES[rand.nextInt(BONUS_TYPES.length)]));
        }
    }

    private void shoot(int x, int y, String dir, boolean isPlayer) {
        double speed = isPlayer ? 2.0 : 1.5;
        bullets.add(new Bullet(x + TANK_SIZE / 2, y + TANK_SIZE / 2, dir, isPlayer, speed));
    }

    public void update() {
        if (gameOver) return;

        frame++;

        // Player movement
        if (player.alive) {
            int dx = 0, dy = 0;
            if (keys.getOrDefault("w", false) || keys.getOrDefault("arrowup", false)) { dy = -1; player.dir = "up"; }
            else if (keys.getOrDefault("s", false) || keys.getOrDefault("arrowdown", false)) { dy = 1; player.dir = "down"; }
            else if (keys.getOrDefault("a", false) || keys.getOrDefault("arrowleft", false)) { dx = -1; player.dir = "left"; }
            else if (keys.getOrDefault("d", false) || keys.getOrDefault("arrowright", false)) { dx = 1; player.dir = "right"; }
            if (dx != 0 || dy != 0) {
                int nx = player.x + dx;
                int ny = player.y + dy;
                if (!isCollision(nx, ny)) {
                    player.x = nx;
                    player.y = ny;
                }
            }
            if (keys.getOrDefault(" ", false) || keys.getOrDefault("enter", false)) {
                if (player.cooldown <= 0) {
                    shoot(player.x, player.y, player.dir, true);
                    player.cooldown = 5;
                }
            }
        }
        if (player.cooldown > 0) player.cooldown--;

        // Enemy spawn
        int spawnRate = Math.max(20, 60 - level * 4);
        int alive = 0;
        for (Enemy e : enemies) if (e.alive) alive++;
        if (frame % spawnRate == 0 && alive < 3 + level) {
            spawnEnemy();
        }

        // Enemy movement and shooting
        Random rand = new Random();
        for (Enemy e : enemies) {
            if (e.alive) {
                int dx = Integer.compare(player.x, e.x);
                int dy = Integer.compare(player.y, e.y);
                if (Math.abs(dx) > Math.abs(dy)) {
                    if (dx > 0 && !isCollision(e.x + 1, e.y)) { e.x++; e.dir = "right"; }
                    else if (dx < 0 && !isCollision(e.x - 1, e.y)) { e.x--; e.dir = "left"; }
                } else {
                    if (dy > 0 && !isCollision(e.x, e.y + 1)) { e.y++; e.dir = "down"; }
                    else if (dy < 0 && !isCollision(e.x, e.y - 1)) { e.y--; e.dir = "up"; }
                }
                e.cooldown--;
                if (e.cooldown <= 0 && rand.nextDouble() < 0.02 + level * 0.005) {
                    shoot(e.x, e.y, e.dir, false);
                    e.cooldown = 20 + rand.nextInt(20);
                }
            }
        }

        // Bullets
        for (Iterator<Bullet> it = bullets.iterator(); it.hasNext(); ) {
            Bullet b = it.next();
            switch (b.dir) {
                case "up": b.y -= b.speed; break;
                case "down": b.y += b.speed; break;
                case "left": b.x -= b.speed; break;
                case "right": b.x += b.speed; break;
            }
            int bx = (int)Math.round(b.x);
            int by = (int)Math.round(b.y);
            boolean hit = false;

            if (bx < 0 || bx >= WIDTH || by < 0 || by >= HEIGHT) { it.remove(); continue; }

            boolean wallHit = false;
            for (int[] w : walls) {
                if (w[0] == bx && w[1] == by) { wallHit = true; break; }
            }
            if (wallHit) {
                it.remove();
                explosions.add(new Explosion(bx, by, 10));
                continue;
            }

            if (b.isPlayer) {
                for (Enemy e : enemies) {
                    if (e.alive && e.x <= bx && bx < e.x + TANK_SIZE && e.y <= by && by < e.y + TANK_SIZE) {
                        e.hp--;
                        if (e.hp <= 0) {
                            e.alive = false;
                            player.score += 10;
                            explosions.add(new Explosion(e.x + 1, e.y + 1, 15));
                            spawnBonus(e.x, e.y);
                        }
                        hit = true;
                        break;
                    }
                }
            } else {
                if (player.alive && player.x <= bx && bx < player.x + TANK_SIZE &&
                    player.y <= by && by < player.y + TANK_SIZE) {
                    if (!player.shield) {
                        player.hp--;
                        explosions.add(new Explosion(player.x + 1, player.y + 1, 20));
                        if (player.hp <= 0) {
                            player.alive = false;
                            gameOver = true;
                            if (player.score > highScore) {
                                highScore = player.score;
                                saveHighScore();
                            }
                        }
                    }
                    hit = true;
                }
            }
            if (hit) it.remove();
        }

        // Win condition
        if (player.score >= 100 + level * 20) {
            win = true;
            gameOver = true;
        }

        // Bonuses
        for (Iterator<Bonus> it = bonuses.iterator(); it.hasNext(); ) {
            Bonus b = it.next();
            if (b.active) {
                if (player.x <= b.x && b.x < player.x + TANK_SIZE &&
                    player.y <= b.y && b.y < player.y + TANK_SIZE) {
                    switch (b.type) {
                        case "❤️": player.hp = Math.min(5, player.hp + 1); break;
                        case "⚡": player.speed = Math.min(3.0, player.speed + 0.5); break;
                        case "🛡️": player.shield = true; player.shieldTimer = 120; break;
                    }
                    it.remove();
                    continue;
                }
                b.life--;
                if (b.life <= 0) it.remove();
            }
        }

        // Shield timer
        if (player.shield) {
            player.shieldTimer--;
            if (player.shieldTimer <= 0) player.shield = false;
        }

        // Explosions
        for (Iterator<Explosion> it = explosions.iterator(); it.hasNext(); ) {
            Explosion e = it.next();
            e.life--;
            if (e.life <= 0) it.remove();
        }
    }

    public void draw() {
        System.out.print("\033[H\033[2J");
        System.out.flush();

        String[][] grid = new String[HEIGHT][WIDTH];
        for (int i = 0; i < HEIGHT; i++) {
            Arrays.fill(grid[i], " ");
        }

        for (int[] w : walls) {
            int x = w[0], y = w[1];
            if (y >= 0 && y < HEIGHT && x >= 0 && x < WIDTH) {
                grid[y][x] = "\033[37m" + WALL_CHAR + "\033[0m";
            }
        }

        for (Enemy e : enemies) {
            if (e.alive) {
                for (int dy = 0; dy < TANK_SIZE; dy++) {
                    for (int dx = 0; dx < TANK_SIZE; dx++) {
                        int ex = e.x + dx, ey = e.y + dy;
                        if (ey >= 0 && ey < HEIGHT && ex >= 0 && ex < WIDTH) {
                            grid[ey][ex] = (dx == 1 && dy == 1) ? "\033[31m▼\033[0m" : "\033[31m█\033[0m";
                        }
                    }
                }
            }
        }

        if (player.alive) {
            String ch = switch (player.dir) {
                case "up" -> "▲";
                case "down" -> "▼";
                case "left" -> "◄";
                default -> "►";
            };
            for (int dy = 0; dy < TANK_SIZE; dy++) {
                for (int dx = 0; dx < TANK_SIZE; dx++) {
                    int px = player.x + dx, py = player.y + dy;
                    if (py >= 0 && py < HEIGHT && px >= 0 && px < WIDTH) {
                        grid[py][px] = (dx == 1 && dy == 1) ? "\033[32m" + ch + "\033[0m" : "\033[32m█\033[0m";
                    }
                }
            }
        }

        for (Bullet b : bullets) {
            int bx = (int)Math.round(b.x);
            int by = (int)Math.round(b.y);
            if (by >= 0 && by < HEIGHT && bx >= 0 && bx < WIDTH) {
                String color = b.isPlayer ? "\033[33m" : "\033[31m";
                grid[by][bx] = color + BULLET_CHAR + "\033[0m";
            }
        }

        for (Bonus b : bonuses) {
            if (b.active && b.y >= 0 && b.y < HEIGHT && b.x >= 0 && b.x < WIDTH) {
                grid[b.y][b.x] = "\033[33m" + b.type + "\033[0m";
            }
        }

        for (Explosion e : explosions) {
            int ex = (int)Math.round(e.x);
            int ey = (int)Math.round(e.y);
            if (ey >= 0 && ey < HEIGHT && ex >= 0 && ex < WIDTH) {
                String ch = EXPLOSION_CHARS[new Random().nextInt(EXPLOSION_CHARS.length)];
                grid[ey][ex] = "\033[31m" + ch + "\033[0m";
            }
        }

        System.out.println("┌" + "─".repeat(WIDTH) + "┐");
        for (String[] row : grid) {
            System.out.println("│" + String.join("", row) + "│");
        }
        System.out.println("└" + "─".repeat(WIDTH) + "┘");

        String hpBar = "█".repeat(player.hp) + "░".repeat(5 - player.hp);
        String shieldStr = player.shield ? "🛡️ " : "";
        System.out.printf("HP: \033[32m%s\033[0m  Score: %d  Level: %d  %s Best: %d%n",
            hpBar, player.score, level, shieldStr, highScore);

        if (gameOver) {
            System.out.println(win ? "\033[36m🎉 ПОБЕДА! Нажмите R для рестарта\033[0m" :
                                    "\033[31m💀 ИГРА ОКОНЧЕНА! Нажмите R для рестарта\033[0m");
        }
    }

    private void startInputThread() {
        new Thread(() -> {
            while (running) {
                try {
                    if (System.in.available() > 0) {
                        char ch = (char) System.in.read();
                        if (ch == 'q' || ch == 'Q') { running = false; System.exit(0); }
                        if ((ch == 'r' || ch == 'R') && gameOver) { resetGame(); }
                        String key = switch (ch) {
                            case 'w', 'W' -> "w";
                            case 's', 'S' -> "s";
                            case 'a', 'A' -> "a";
                            case 'd', 'D' -> "d";
                            case ' ' -> " ";
                            case '\n' -> "enter";
                            default -> "";
                        };
                        if (!key.isEmpty()) {
                            keys.put(key, true);
                            Thread.sleep(100);
                            keys.put(key, false);
                        }
                    }
                    Thread.sleep(10);
                } catch (Exception e) {}
            }
        }).start();
    }

    public void run() {
        System.out.println("\033[36mТанки 8-бит - Управление: WASD/Стрелки, Space - стрельба, Q - выход\033[0m");
        System.out.println("Нажмите Enter для начала...");
        scanner.nextLine();

        while (running) {
            update();
            draw();
            try { Thread.sleep(50); } catch (InterruptedException e) {}
        }
    }

    public static void main(String[] args) {
        Tanks game = new Tanks();
        game.run();
    }
}
