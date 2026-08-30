// tanks.cpp
#include <iostream>
#include <vector>
#include <string>
#include <map>
#include <cstdlib>
#include <ctime>
#include <thread>
#include <chrono>
#include <fstream>
#include <json/json.h> // using jsoncpp

using namespace std;

const int WIDTH = 40;
const int HEIGHT = 20;
const int TANK_SIZE = 3;
const string BULLET_CHAR = "•";
const string WALL_CHAR = "█";
const vector<string> EXPLOSION_CHARS = {"💥", "🔥", "💫"};
const vector<string> BONUS_TYPES = {"❤️", "⚡", "🛡️"};

struct Player {
    int x, y, hp, score, cooldown, shieldTimer;
    string dir;
    double speed;
    bool shield, alive;
    Player() : x(WIDTH/2), y(HEIGHT-3), hp(3), score(0), cooldown(0), shieldTimer(0),
               dir("up"), speed(1.0), shield(false), alive(true) {}
};

struct Enemy {
    int x, y, hp, cooldown;
    string dir;
    double speed;
    bool alive;
    Enemy(int x, int y) : x(x), y(y), hp(1), cooldown(0), dir("up"), speed(0.5), alive(true) {}
};

struct Bullet {
    double x, y, speed;
    string dir;
    bool isPlayer;
    Bullet(double x, double y, string dir, bool isPlayer, double speed) : x(x), y(y), dir(dir), isPlayer(isPlayer), speed(speed) {}
};

struct Bonus {
    int x, y, life;
    string type;
    bool active;
    Bonus(int x, int y, string type) : x(x), y(y), type(type), life(100), active(true) {}
};

struct Explosion {
    double x, y;
    int life;
    Explosion(double x, double y, int life) : x(x), y(y), life(life) {}
};

class Game {
private:
    Player player;
    vector<Bullet> bullets;
    vector<Enemy> enemies;
    vector<Bonus> bonuses;
    vector<Explosion> explosions;
    vector<pair<int,int>> walls;
    int frame, level, highScore;
    bool gameOver, win, running;
    map<string, bool> keys;
    mt19937 rng;

    vector<pair<int,int>> generateWalls() {
        vector<pair<int,int>> w;
        for (int x = 0; x < WIDTH; x++) {
            w.push_back({x, 0});
            w.push_back({x, HEIGHT-1});
        }
        for (int y = 0; y < HEIGHT; y++) {
            w.push_back({0, y});
            w.push_back({WIDTH-1, y});
        }
        uniform_int_distribution<int> dist(2, WIDTH-3);
        for (int i = 0; i < 8; i++) {
            int x = dist(rng);
            int y = dist(rng);
            for (int dx = 0; dx < 2; dx++) {
                for (int dy = 0; dy < 2; dy++) {
                    int wx = x+dx, wy = y+dy;
                    bool found = false;
                    for (auto& p : w) if (p.first == wx && p.second == wy) { found = true; break; }
                    if (!found) w.push_back({wx, wy});
                }
            }
        }
        return w;
    }

    bool isCollision(int x, int y) {
        for (int dx = 0; dx < TANK_SIZE; dx++) {
            for (int dy = 0; dy < TANK_SIZE; dy++) {
                int px = x+dx, py = y+dy;
                if (px < 0 || px >= WIDTH || py < 0 || py >= HEIGHT) return true;
                for (auto& w : walls) if (w.first == px && w.second == py) return true;
            }
        }
        for (auto& e : enemies) {
            if (e.alive) {
                for (int dx = 0; dx < TANK_SIZE; dx++) {
                    for (int dy = 0; dy < TANK_SIZE; dy++) {
                        int ex = e.x+dx, ey = e.y+dy;
                        for (int dx2 = 0; dx2 < TANK_SIZE; dx2++) {
                            for (int dy2 = 0; dy2 < TANK_SIZE; dy2++) {
                                if (x+dx2 == ex && y+dy2 == ey) return true;
                            }
                        }
                    }
                }
            }
        }
        return false;
    }

    void spawnEnemy() {
        uniform_int_distribution<int> sideDist(0, 3);
        uniform_int_distribution<int> posDist(1, WIDTH-TANK_SIZE-2);
        int side = sideDist(rng);
        int x, y;
        if (side == 0) { x = posDist(rng); y = 1; }
        else if (side == 1) { x = posDist(rng); y = HEIGHT-TANK_SIZE-1; }
        else if (side == 2) { x = 1; y = posDist(rng); }
        else { x = WIDTH-TANK_SIZE-1; y = posDist(rng); }
        if (!isCollision(x, y)) {
            enemies.push_back(Enemy(x, y));
        }
    }

    void spawnBonus(int x, int y) {
        uniform_real_distribution<double> prob(0, 1);
        uniform_int_distribution<int> typeDist(0, BONUS_TYPES.size()-1);
        if (prob(rng) < 0.15) {
            bonuses.push_back(Bonus(x, y, BONUS_TYPES[typeDist(rng)]));
        }
    }

    void shoot(int x, int y, string dir, bool isPlayer) {
        double speed = isPlayer ? 2.0 : 1.5;
        bullets.push_back(Bullet(x + TANK_SIZE/2, y + TANK_SIZE/2, dir, isPlayer, speed));
    }

public:
    Game() : frame(0), level(1), highScore(0), gameOver(false), win(false), running(true) {
        rng.seed(time(nullptr));
        resetGame();
        loadHighScore();
    }

    void resetGame() {
        player = Player();
        bullets.clear();
        enemies.clear();
        bonuses.clear();
        explosions.clear();
        walls = generateWalls();
        frame = 0;
        level = 1;
        gameOver = false;
        win = false;
    }

    void loadHighScore() {
        ifstream ifs("tanks_score.json");
        if (ifs) {
            Json::Value root;
            ifs >> root;
            highScore = root.get("high_score", 0).asInt();
        }
    }

    void saveHighScore() {
        Json::Value root;
        root["high_score"] = highScore;
        ofstream ofs("tanks_score.json");
        ofs << root.toStyledString();
    }

    void update() {
        if (gameOver) return;

        frame++;

        // Player
        if (player.alive) {
            int dx = 0, dy = 0;
            if (keys["w"] || keys["arrowup"]) { dy = -1; player.dir = "up"; }
            else if (keys["s"] || keys["arrowdown"]) { dy = 1; player.dir = "down"; }
            else if (keys["a"] || keys["arrowleft"]) { dx = -1; player.dir = "left"; }
            else if (keys["d"] || keys["arrowright"]) { dx = 1; player.dir = "right"; }
            if (dx != 0 || dy != 0) {
                int nx = player.x + dx;
                int ny = player.y + dy;
                if (!isCollision(nx, ny)) { player.x = nx; player.y = ny; }
            }
            if (keys[" "] || keys["enter"]) {
                if (player.cooldown <= 0) {
                    shoot(player.x, player.y, player.dir, true);
                    player.cooldown = 5;
                }
            }
        }
        if (player.cooldown > 0) player.cooldown--;

        // Enemy spawn
        int spawnRate = max(20, 60 - level * 4);
        int alive = 0;
        for (auto& e : enemies) if (e.alive) alive++;
        if (frame % spawnRate == 0 && alive < 3 + level) spawnEnemy();

        // Enemies
        uniform_real_distribution<double> prob(0, 1);
        for (auto& e : enemies) {
            if (e.alive) {
                int dx = (player.x > e.x) ? 1 : (player.x < e.x) ? -1 : 0;
                int dy = (player.y > e.y) ? 1 : (player.y < e.y) ? -1 : 0;
                if (abs(dx) > abs(dy)) {
                    if (dx > 0 && !isCollision(e.x+1, e.y)) { e.x++; e.dir = "right"; }
                    else if (dx < 0 && !isCollision(e.x-1, e.y)) { e.x--; e.dir = "left"; }
                } else {
                    if (dy > 0 && !isCollision(e.x, e.y+1)) { e.y++; e.dir = "down"; }
                    else if (dy < 0 && !isCollision(e.x, e.y-1)) { e.y--; e.dir = "up"; }
                }
                e.cooldown--;
                if (e.cooldown <= 0 && prob(rng) < 0.02 + level * 0.005) {
                    shoot(e.x, e.y, e.dir, false);
                    e.cooldown = 20 + (int)(prob(rng) * 20);
                }
            }
        }

        // Bullets
        for (int i = bullets.size()-1; i >= 0; i--) {
            Bullet& b = bullets[i];
            if (b.dir == "up") b.y -= b.speed;
            else if (b.dir == "down") b.y += b.speed;
            else if (b.dir == "left") b.x -= b.speed;
            else if (b.dir == "right") b.x += b.speed;
            int bx = (int)round(b.x);
            int by = (int)round(b.y);
            bool hit = false;

            if (bx < 0 || bx >= WIDTH || by < 0 || by >= HEIGHT) {
                bullets.erase(bullets.begin() + i);
                continue;
            }

            bool wallHit = false;
            for (auto& w : walls) if (w.first == bx && w.second == by) { wallHit = true; break; }
            if (wallHit) {
                bullets.erase(bullets.begin() + i);
                explosions.push_back(Explosion(bx, by, 10));
                continue;
            }

            if (b.isPlayer) {
                for (auto& e : enemies) {
                    if (e.alive && e.x <= bx && bx < e.x + TANK_SIZE && e.y <= by && by < e.y + TANK_SIZE) {
                        e.hp--;
                        if (e.hp <= 0) {
                            e.alive = false;
                            player.score += 10;
                            explosions.push_back(Explosion(e.x+1, e.y+1, 15));
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
                        explosions.push_back(Explosion(player.x+1, player.y+1, 20));
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
            if (hit) bullets.erase(bullets.begin() + i);
        }

        // Win
        if (player.score >= 100 + level * 20) { win = true; gameOver = true; }

        // Bonuses
        for (int i = bonuses.size()-1; i >= 0; i--) {
            Bonus& b = bonuses[i];
            if (b.active) {
                if (player.x <= b.x && b.x < player.x + TANK_SIZE &&
                    player.y <= b.y && b.y < player.y + TANK_SIZE) {
                    if (b.type == "❤️") player.hp = min(5, player.hp + 1);
                    else if (b.type == "⚡") player.speed = min(3.0, player.speed + 0.5);
                    else if (b.type == "🛡️") { player.shield = true; player.shieldTimer = 120; }
                    bonuses.erase(bonuses.begin() + i);
                    continue;
                }
                b.life--;
                if (b.life <= 0) bonuses.erase(bonuses.begin() + i);
            }
        }

        // Shield
        if (player.shield) {
            player.shieldTimer--;
            if (player.shieldTimer <= 0) player.shield = false;
        }

        // Explosions
        for (int i = explosions.size()-1; i >= 0; i--) {
            explosions[i].life--;
            if (explosions[i].life <= 0) explosions.erase(explosions.begin() + i);
        }
    }

    void draw() {
        system("clear");
        vector<vector<string>> grid(HEIGHT, vector<string>(WIDTH, " "));

        for (auto& w : walls) {
            int x = w.first, y = w.second;
            if (y >= 0 && y < HEIGHT && x >= 0 && x < WIDTH)
                grid[y][x] = "\033[37m" + WALL_CHAR + "\033[0m";
        }

        for (auto& e : enemies) {
            if (e.alive) {
                for (int dy = 0; dy < TANK_SIZE; dy++) {
                    for (int dx = 0; dx < TANK_SIZE; dx++) {
                        int ex = e.x+dx, ey = e.y+dy;
                        if (ey >= 0 && ey < HEIGHT && ex >= 0 && ex < WIDTH)
                            grid[ey][ex] = (dx == 1 && dy == 1) ? "\033[31m▼\033[0m" : "\033[31m█\033[0m";
                    }
                }
            }
        }

        if (player.alive) {
            string ch = player.dir == "up" ? "▲" : player.dir == "down" ? "▼" : player.dir == "left" ? "◄" : "►";
            for (int dy = 0; dy < TANK_SIZE; dy++) {
                for (int dx = 0; dx < TANK_SIZE; dx++) {
                    int px = player.x+dx, py = player.y+dy;
                    if (py >= 0 && py < HEIGHT && px >= 0 && px < WIDTH)
                        grid[py][px] = (dx == 1 && dy == 1) ? "\033[32m" + ch + "\033[0m" : "\033[32m█\033[0m";
                }
            }
        }

        for (auto& b : bullets) {
            int bx = (int)round(b.x), by = (int)round(b.y);
            if (by >= 0 && by < HEIGHT && bx >= 0 && bx < WIDTH)
                grid[by][bx] = (b.isPlayer ? "\033[33m" : "\033[31m") + BULLET_CHAR + "\033[0m";
        }

        for (auto& b : bonuses) {
            if (b.active && b.y >= 0 && b.y < HEIGHT && b.x >= 0 && b.x < WIDTH)
                grid[b.y][b.x] = "\033[33m" + b.type + "\033[0m";
        }

        for (auto& e : explosions) {
            int ex = (int)round(e.x), ey = (int)round(e.y);
            if (ey >= 0 && ey < HEIGHT && ex >= 0 && ex < WIDTH)
                grid[ey][ex] = "\033[31m" + EXPLOSION_CHARS[rand() % EXPLOSION_CHARS.size()] + "\033[0m";
        }

        cout << "┌" << string(WIDTH, '─') << "┐" << endl;
        for (auto& row : grid) {
            cout << "│";
            for (auto& c : row) cout << c;
            cout << "│" << endl;
        }
        cout << "└" << string(WIDTH, '─') << "┘" << endl;

        string hpBar = string(player.hp, '█') + string(5 - player.hp, '░');
        string shieldStr = player.shield ? "🛡️ " : "";
        cout << "HP: \033[32m" << hpBar << "\033[0m  Score: " << player.score << "  Level: " << level << "  " << shieldStr << "Best: " << highScore << endl;

        if (gameOver) {
            if (win) cout << "\033[36m🎉 ПОБЕДА! Нажмите R для рестарта\033[0m" << endl;
            else cout << "\033[31m💀 ИГРА ОКОНЧЕНА! Нажмите R для рестарта\033[0m" << endl;
        }
    }

    void run() {
        cout << "\033[36mТанки 8-бит - Управление: WASD/Стрелки, Space - стрельба, Q - выход\033[0m" << endl;
        cout << "Нажмите Enter для начала...";
        cin.get();

        while (running
