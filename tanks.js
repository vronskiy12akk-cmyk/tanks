#!/usr/bin/env node
// tanks.js
const readline = require('readline');
const chalk = require('chalk');
const fs = require('fs');

const WIDTH = 40;
const HEIGHT = 20;
const TANK_SIZE = 3;
const BULLET_CHAR = '•';
const WALL_CHAR = '█';
const EXPLOSION_CHARS = ['💥', '🔥', '💫'];
const BONUS_TYPES = ['❤️', '⚡', '🛡️'];

class TankGame {
    constructor() {
        this.resetGame();
        this.highScore = this.loadHighScore();
        this.running = true;
        this.keys = {};
        this.setupInput();
    }

    resetGame() {
        this.width = WIDTH;
        this.height = HEIGHT;
        this.player = {
            x: Math.floor(WIDTH / 2),
            y: HEIGHT - 3,
            dir: 'up',
            hp: 3,
            score: 0,
            speed: 1,
            shield: false,
            cooldown: 0,
            alive: true
        };
        this.bullets = [];
        this.enemies = [];
        this.bonuses = [];
        this.explosions = [];
        this.walls = this.generateWalls();
        this.enemySpawnTimer = 0;
        this.enemySpawnInterval = 60;
        this.gameOver = false;
        this.win = false;
        this.level = 1;
        this.frame = 0;
        this.shootCooldown = 0;
    }

    generateWalls() {
        const walls = [];
        for (let x = 0; x < this.width; x++) {
            walls.push([x, 0]);
            walls.push([x, this.height - 1]);
        }
        for (let y = 0; y < this.height; y++) {
            walls.push([0, y]);
            walls.push([this.width - 1, y]);
        }
        for (let i = 0; i < 8; i++) {
            const x = Math.floor(Math.random() * (this.width - 4)) + 2;
            const y = Math.floor(Math.random() * (this.height - 4)) + 2;
            for (let dx = 0; dx < 2; dx++) {
                for (let dy = 0; dy < 2; dy++) {
                    const wall = [x + dx, y + dy];
                    if (!walls.some(w => w[0] === wall[0] && w[1] === wall[1])) {
                        walls.push(wall);
                    }
                }
            }
        }
        return walls;
    }

    loadHighScore() {
        try {
            if (fs.existsSync('tanks_score.json')) {
                const data = JSON.parse(fs.readFileSync('tanks_score.json'));
                return data.high_score || 0;
            }
        } catch (e) {}
        return 0;
    }

    saveHighScore() {
        try {
            fs.writeFileSync('tanks_score.json', JSON.stringify({ high_score: this.highScore }));
        } catch (e) {}
    }

    isCollision(x, y, size) {
        for (let dx = 0; dx < size; dx++) {
            for (let dy = 0; dy < size; dy++) {
                const px = x + dx;
                const py = y + dy;
                if (px < 0 || px >= this.width || py < 0 || py >= this.height) return true;
                if (this.walls.some(w => w[0] === px && w[1] === py)) return true;
            }
        }
        for (const e of this.enemies) {
            if (e.alive) {
                for (let dx = 0; dx < TANK_SIZE; dx++) {
                    for (let dy = 0; dy < TANK_SIZE; dy++) {
                        const ex = e.x + dx;
                        const ey = e.y + dy;
                        for (let dx2 = 0; dx2 < TANK_SIZE; dx2++) {
                            for (let dy2 = 0; dy2 < TANK_SIZE; dy2++) {
                                if (x + dx2 === ex && y + dy2 === ey) return true;
                            }
                        }
                    }
                }
            }
        }
        return false;
    }

    spawnEnemy() {
        const side = Math.floor(Math.random() * 4);
        let x, y;
        if (side === 0) { x = Math.floor(Math.random() * (this.width - TANK_SIZE - 2)) + 1; y = 1; }
        else if (side === 1) { x = Math.floor(Math.random() * (this.width - TANK_SIZE - 2)) + 1; y = this.height - TANK_SIZE - 1; }
        else if (side === 2) { x = 1; y = Math.floor(Math.random() * (this.height - TANK_SIZE - 2)) + 1; }
        else { x = this.width - TANK_SIZE - 1; y = Math.floor(Math.random() * (this.height - TANK_SIZE - 2)) + 1; }
        if (!this.isCollision(x, y, TANK_SIZE)) {
            const dirs = ['up', 'down', 'left', 'right'];
            this.enemies.push({
                x, y,
                dir: dirs[Math.floor(Math.random() * dirs.length)],
                hp: 1,
                speed: 0.5 + this.level * 0.1,
                alive: true,
                cooldown: Math.floor(Math.random() * 30)
            });
        }
    }

    spawnBonus(x, y) {
        if (Math.random() < 0.15) {
            this.bonuses.push({
                x, y,
                type: BONUS_TYPES[Math.floor(Math.random() * BONUS_TYPES.length)],
                active: true,
                life: 100
            });
        }
    }

    shoot(x, y, dir, isPlayer = true) {
        this.bullets.push({
            x: x + Math.floor(TANK_SIZE / 2),
            y: y + Math.floor(TANK_SIZE / 2),
            dir: dir,
            isPlayer: isPlayer,
            speed: isPlayer ? 2 : 1.5
        });
    }

    moveEnemy(enemy) {
        if (!enemy.alive) return;
        const dx = this.player.x - enemy.x;
        const dy = this.player.y - enemy.y;
        if (Math.abs(dx) > Math.abs(dy)) {
            if (dx > 0 && !this.isCollision(enemy.x + 1, enemy.y, TANK_SIZE)) {
                enemy.x += enemy.speed;
                enemy.dir = 'right';
            } else if (dx < 0 && !this.isCollision(enemy.x - 1, enemy.y, TANK_SIZE)) {
                enemy.x -= enemy.speed;
                enemy.dir = 'left';
            }
        } else {
            if (dy > 0 && !this.isCollision(enemy.x, enemy.y + 1, TANK_SIZE)) {
                enemy.y += enemy.speed;
                enemy.dir = 'down';
            } else if (dy < 0 && !this.isCollision(enemy.x, enemy.y - 1, TANK_SIZE)) {
                enemy.y -= enemy.speed;
                enemy.dir = 'up';
            }
        }
    }

    update() {
        if (this.gameOver) return;

        this.frame++;
        this.shootCooldown = Math.max(0, this.shootCooldown - 1);

        // Player movement
        if (this.player.alive) {
            let dx = 0, dy = 0;
            if (this.keys['w'] || this.keys['arrowup']) { dy = -this.player.speed; this.player.dir = 'up'; }
            else if (this.keys['s'] || this.keys['arrowdown']) { dy = this.player.speed; this.player.dir = 'down'; }
            else if (this.keys['a'] || this.keys['arrowleft']) { dx = -this.player.speed; this.player.dir = 'left'; }
            else if (this.keys['d'] || this.keys['arrowright']) { dx = this.player.speed; this.player.dir = 'right'; }
            if (dx !== 0 || dy !== 0) {
                const newX = this.player.x + dx;
                const newY = this.player.y + dy;
                if (!this.isCollision(newX, newY, TANK_SIZE)) {
                    this.player.x = newX;
                    this.player.y = newY;
                }
            }
            if ((this.keys[' '] || this.keys['enter']) && this.shootCooldown === 0) {
                this.shoot(this.player.x, this.player.y, this.player.dir, true);
                this.shootCooldown = 5;
            }
        }

        // Enemy spawn
        const spawnRate = Math.max(20, 60 - this.level * 4);
        if (this.frame % spawnRate === 0 && this.enemies.filter(e => e.alive).length < 3 + this.level) {
            this.spawnEnemy();
        }

        // Enemy movement and shooting
        for (const e of this.enemies) {
            if (e.alive) {
                this.moveEnemy(e);
                e.cooldown--;
                if (e.cooldown <= 0 && Math.random() < 0.02 + this.level * 0.005) {
                    this.shoot(e.x, e.y, e.dir, false);
                    e.cooldown = 20 + Math.floor(Math.random() * 20);
                }
            }
        }

        // Bullets
        for (let i = this.bullets.length - 1; i >= 0; i--) {
            const b = this.bullets[i];
            if (b.dir === 'up') b.y -= b.speed;
            else if (b.dir === 'down') b.y += b.speed;
            else if (b.dir === 'left') b.x -= b.speed;
            else if (b.dir === 'right') b.x += b.speed;

            const bx = Math.round(b.x);
            const by = Math.round(b.y);
            let hit = false;

            if (bx < 0 || bx >= this.width || by < 0 || by >= this.height) {
                this.bullets.splice(i, 1);
                continue;
            }

            if (this.walls.some(w => w[0] === bx && w[1] === by)) {
                this.bullets.splice(i, 1);
                this.explosions.push({ x: bx, y: by, life: 10 });
                continue;
            }

            if (b.isPlayer) {
                for (const e of this.enemies) {
                    if (e.alive && e.x <= bx && bx < e.x + TANK_SIZE && e.y <= by && by < e.y + TANK_SIZE) {
                        e.hp--;
                        if (e.hp <= 0) {
                            e.alive = false;
                            this.player.score += 10;
                            this.explosions.push({ x: e.x + 1, y: e.y + 1, life: 15 });
                            this.spawnBonus(e.x, e.y);
                        }
                        hit = true;
                        break;
                    }
                }
            } else {
                if (this.player.alive && this.player.x <= bx && bx < this.player.x + TANK_SIZE &&
                    this.player.y <= by && by < this.player.y + TANK_SIZE) {
                    if (!this.player.shield) {
                        this.player.hp--;
                        this.explosions.push({ x: this.player.x + 1, y: this.player.y + 1, life: 20 });
                        if (this.player.hp <= 0) {
                            this.player.alive = false;
                            this.gameOver = true;
                            if (this.player.score > this.highScore) {
                                this.highScore = this.player.score;
                                this.saveHighScore();
                            }
                        }
                    }
                    hit = true;
                }
            }

            if (hit) {
                this.bullets.splice(i, 1);
            }
        }

        // Win condition
        if (this.player.score >= 100 + this.level * 20) {
            this.win = true;
            this.gameOver = true;
        }

        // Bonuses
        for (let i = this.bonuses.length - 1; i >= 0; i--) {
            const b = this.bonuses[i];
            if (b.active) {
                if (this.player.x <= b.x && b.x < this.player.x + TANK_SIZE &&
                    this.player.y <= b.y && b.y < this.player.y + TANK_SIZE) {
                    if (b.type === '❤️') this.player.hp = Math.min(5, this.player.hp + 1);
                    else if (b.type === '⚡') this.player.speed = Math.min(3, this.player.speed + 0.5);
                    else if (b.type === '🛡️') { this.player.shield = true; this.player.shieldTimer = 120; }
                    this.bonuses.splice(i, 1);
                    continue;
                }
                b.life--;
                if (b.life <= 0) this.bonuses.splice(i, 1);
            }
        }

        // Shield timer
        if (this.player.shield) {
            this.player.shieldTimer = (this.player.shieldTimer || 0) - 1;
            if (this.player.shieldTimer <= 0) this.player.shield = false;
        }

        // Explosions
        for (let i = this.explosions.length - 1; i >= 0; i--) {
            this.explosions[i].life--;
            if (this.explosions[i].life <= 0) this.explosions.splice(i, 1);
        }
    }

    draw() {
        console.clear();
        const grid = Array.from({ length: this.height }, () => Array(this.width).fill(' '));

        for (const [x, y] of this.walls) {
            if (y >= 0 && y < this.height && x >= 0 && x < this.width) {
                grid[y][x] = chalk.white(WALL_CHAR);
            }
        }

        for (const e of this.enemies) {
            if (e.alive) {
                for (let dy = 0; dy < TANK_SIZE; dy++) {
                    for (let dx = 0; dx < TANK_SIZE; dx++) {
                        const ex = e.x + dx;
                        const ey = e.y + dy;
                        if (ey >= 0 && ey < this.height && ex >= 0 && ex < this.width) {
                            grid[ey][ex] = dx === 1 && dy === 1 ? chalk.red('▼') : chalk.red('█');
                        }
                    }
                }
            }
        }

        if (this.player.alive) {
            const dirChars = { up: '▲', down: '▼', left: '◄', right: '►' };
            const ch = dirChars[this.player.dir] || '▲';
            for (let dy = 0; dy < TANK_SIZE; dy++) {
                for (let dx = 0; dx < TANK_SIZE; dx++) {
                    const px = this.player.x + dx;
                    const py = this.player.y + dy;
                    if (py >= 0 && py < this.height && px >= 0 && px < this.width) {
                        grid[py][px] = dx === 1 && dy === 1 ? chalk.green(ch) : chalk.green('█');
                    }
                }
            }
        }

        for (const b of this.bullets) {
            const bx = Math.round(b.x);
            const by = Math.round(b.y);
            if (by >= 0 && by < this.height && bx >= 0 && bx < this.width) {
                const color = b.isPlayer ? chalk.yellow : chalk.red;
                grid[by][bx] = color(BULLET_CHAR);
            }
        }

        for (const b of this.bonuses) {
            if (b.active && b.y >= 0 && b.y < this.height && b.x >= 0 && b.x < this.width) {
                grid[b.y][b.x] = chalk.yellow(b.type);
            }
        }

        for (const e of this.explosions) {
            const ex = Math.round(e.x);
            const ey = Math.round(e.y);
            if (ey >= 0 && ey < this.height && ex >= 0 && ex < this.width) {
                grid[ey][ex] = chalk.red(EXPLOSION_CHARS[Math.floor(Math.random() * EXPLOSION_CHARS.length)]);
            }
        }

        console.log('┌' + '─'.repeat(this.width) + '┐');
        for (const row of grid) {
            console.log('│' + row.join('') + '│');
        }
        console.log('└' + '─'.repeat(this.width) + '┘');

        const hpBar = '█'.repeat(this.player.hp) + '░'.repeat(5 - this.player.hp);
        const shieldStr = this.player.shield ? '🛡️ ' : '';
        console.log(`HP: ${chalk.green(hpBar)}  Score: ${this.player.score}  Level: ${this.level}  ${shieldStr}  Best: ${this.highScore}`);

        if (this.gameOver) {
            console.log(this.win ? chalk.cyan('🎉 ПОБЕДА! Нажмите R для рестарта') : chalk.red('💀 ИГРА ОКОНЧЕНА! Нажмите R для рестарта'));
        }
    }

    setupInput() {
        readline.emitKeypressEvents(process.stdin);
        process.stdin.setRawMode(true);
        process.stdin.on('keypress', (str, key) => {
            if (key && key.name === 'q') process.exit(0);
            if (key && key.name === 'r' && this.gameOver) this.resetGame();
            if (key) {
                this.keys[key.name] = true;
                if (key.name === 'space') this.keys[' '] = true;
                if (key.name === 'return') this.keys['enter'] = true;
            }
        });
        process.stdin.on('keypress', (str, key) => {
            if (key) {
                setTimeout(() => {
                    this.keys[key.name] = false;
                    if (key.name === 'space') this.keys[' '] = false;
                    if (key.name === 'return') this.keys['enter'] = false;
                }, 100);
            }
        });
    }

    run() {
        console.log(chalk.cyan('Танки 8-бит - Управление: WASD/Стрелки, Space - стрельба, Q - выход'));
        console.log('Нажмите любую клавишу для начала...');
        process.stdin.once('keypress', () => {
            setInterval(() => {
                this.update();
                this.draw();
            }, 50);
        });
    }
}

const game = new TankGame();
game.run();
