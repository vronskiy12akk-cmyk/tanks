#!/usr/bin/env python3
# tanks.py
import random
import sys
import os
import time
import json
import threading
from collections import deque
from colorama import init, Fore, Style, Back

init(autoreset=True)

WIDTH = 40
HEIGHT = 20
TANK_SIZE = 3
BULLET_CHAR = '•'
WALL_CHAR = '█'
PLAYER_CHAR = '▲'
ENEMY_CHAR = '▼'
EXPLOSION_CHARS = ['💥', '🔥', '💫']
BONUS_TYPES = ['❤️', '⚡', '🛡️']

class TankGame:
    def __init__(self):
        self.reset_game()
        self.high_score = self.load_high_score()
        self.running = True

    def reset_game(self):
        self.width = WIDTH
        self.height = HEIGHT
        self.player = {
            'x': self.width // 2,
            'y': self.height - 3,
            'dir': 'up',
            'hp': 3,
            'score': 0,
            'speed': 1,
            'shield': False,
            'cooldown': 0,
            'alive': True
        }
        self.bullets = []
        self.enemies = []
        self.bonuses = []
        self.explosions = []
        self.walls = self.generate_walls()
        self.enemy_spawn_timer = 0
        self.enemy_spawn_interval = 60
        self.game_over = False
        self.win = False
        self.level = 1
        self.frame = 0

    def generate_walls(self):
        walls = []
        # Границы
        for x in range(self.width):
            walls.append((x, 0))
            walls.append((x, self.height - 1))
        for y in range(self.height):
            walls.append((0, y))
            walls.append((self.width - 1, y))
        # Случайные препятствия
        for _ in range(8):
            x = random.randint(3, self.width - 4)
            y = random.randint(3, self.height - 4)
            for dx in range(2):
                for dy in range(2):
                    if (x+dx, y+dy) not in walls:
                        walls.append((x+dx, y+dy))
        return walls

    def load_high_score(self):
        try:
            with open('tanks_score.json', 'r') as f:
                return json.load(f).get('high_score', 0)
        except:
            return 0

    def save_high_score(self):
        try:
            with open('tanks_score.json', 'w') as f:
                json.dump({'high_score': self.high_score}, f)
        except:
            pass

    def is_collision(self, x, y, size):
        for dx in range(size):
            for dy in range(size):
                px, py = x + dx, y + dy
                if px < 0 or px >= self.width or py < 0 or py >= self.height:
                    return True
                if (px, py) in self.walls:
                    return True
        # Проверка столкновений с врагами
        for e in self.enemies:
            if e['alive']:
                for dx in range(TANK_SIZE):
                    for dy in range(TANK_SIZE):
                        ex, ey = e['x'] + dx, e['y'] + dy
                        for dx2 in range(TANK_SIZE):
                            for dy2 in range(TANK_SIZE):
                                if x + dx2 == ex and y + dy2 == ey:
                                    return True
        return False

    def spawn_enemy(self):
        side = random.randint(0, 3)
        if side == 0:  # верх
            x = random.randint(1, self.width - TANK_SIZE - 1)
            y = 1
        elif side == 1:  # низ
            x = random.randint(1, self.width - TANK_SIZE - 1)
            y = self.height - TANK_SIZE - 1
        elif side == 2:  # лево
            x = 1
            y = random.randint(1, self.height - TANK_SIZE - 1)
        else:  # право
            x = self.width - TANK_SIZE - 1
            y = random.randint(1, self.height - TANK_SIZE - 1)
        if not self.is_collision(x, y, TANK_SIZE):
            dirs = ['up', 'down', 'left', 'right']
            self.enemies.append({
                'x': x,
                'y': y,
                'dir': random.choice(dirs),
                'hp': 1,
                'speed': 0.5 + self.level * 0.1,
                'alive': True,
                'cooldown': random.randint(0, 30)
            })

    def spawn_bonus(self, x, y):
        if random.random() < 0.15:
            self.bonuses.append({
                'x': x,
                'y': y,
                'type': random.choice(BONUS_TYPES),
                'active': True
            })

    def shoot(self, x, y, direction, is_player=True):
        self.bullets.append({
            'x': x + TANK_SIZE // 2,
            'y': y + TANK_SIZE // 2,
            'dir': direction,
            'is_player': is_player,
            'speed': 2 if is_player else 1.5
        })

    def move_enemy(self, enemy):
        if not enemy['alive']:
            return
        # Простая логика движения к игроку
        dx = self.player['x'] - enemy['x']
        dy = self.player['y'] - enemy['y']
        if abs(dx) > abs(dy):
            if dx > 0 and not self.is_collision(enemy['x'] + 1, enemy['y'], TANK_SIZE):
                enemy['x'] += enemy['speed']
                enemy['dir'] = 'right'
            elif dx < 0 and not self.is_collision(enemy['x'] - 1, enemy['y'], TANK_SIZE):
                enemy['x'] -= enemy['speed']
                enemy['dir'] = 'left'
        else:
            if dy > 0 and not self.is_collision(enemy['x'], enemy['y'] + 1, TANK_SIZE):
                enemy['y'] += enemy['speed']
                enemy['dir'] = 'down'
            elif dy < 0 and not self.is_collision(enemy['x'], enemy['y'] - 1, TANK_SIZE):
                enemy['y'] -= enemy['speed']
                enemy['dir'] = 'up'

    def update(self):
        if self.game_over:
            return

        self.frame += 1

        # Спавн врагов
        if self.frame % max(20, 60 - self.level * 4) == 0 and len([e for e in self.enemies if e['alive']]) < 3 + self.level:
            self.spawn_enemy()

        # Движение врагов
        for e in self.enemies:
            if e['alive']:
                self.move_enemy(e)
                # Стрельба врагов
                e['cooldown'] -= 1
                if e['cooldown'] <= 0:
                    if random.random() < 0.02 + self.level * 0.005:
                        self.shoot(e['x'], e['y'], e['dir'], False)
                        e['cooldown'] = 20 + random.randint(0, 20)

        # Движение пуль
        for b in self.bullets[:]:
            if b['dir'] == 'up':
                b['y'] -= b['speed']
            elif b['dir'] == 'down':
                b['y'] += b['speed']
            elif b['dir'] == 'left':
                b['x'] -= b['speed']
            elif b['dir'] == 'right':
                b['x'] += b['speed']

            # Проверка попаданий
            hit = False
            bx, by = int(b['x']), int(b['y'])
            if bx < 0 or bx >= self.width or by < 0 or by >= self.height:
                self.bullets.remove(b)
                continue

            # Столкновение со стенами
            if (bx, by) in self.walls:
                self.bullets.remove(b)
                self.explosions.append({'x': bx, 'y': by, 'life': 10})
                continue

            if b['is_player']:
                # Попадание во врага
                for e in self.enemies:
                    if e['alive'] and e['x'] <= bx < e['x'] + TANK_SIZE and e['y'] <= by < e['y'] + TANK_SIZE:
                        e['hp'] -= 1
                        if e['hp'] <= 0:
                            e['alive'] = False
                            self.player['score'] += 10
                            self.explosions.append({'x': e['x'] + 1, 'y': e['y'] + 1, 'life': 15})
                            self.spawn_bonus(e['x'], e['y'])
                        hit = True
                        break
            else:
                # Попадание в игрока
                if (self.player['x'] <= bx < self.player['x'] + TANK_SIZE and 
                    self.player['y'] <= by < self.player['y'] + TANK_SIZE):
                    if not self.player['shield']:
                        self.player['hp'] -= 1
                        self.explosions.append({'x': self.player['x'] + 1, 'y': self.player['y'] + 1, 'life': 20})
                        if self.player['hp'] <= 0:
                            self.player['alive'] = False
                            self.game_over = True
                            if self.player['score'] > self.high_score:
                                self.high_score = self.player['score']
                                self.save_high_score()
                    hit = True

            if hit:
                self.bullets.remove(b)

        # Проверка победы
        if self.player['score'] >= 100 + self.level * 20:
            self.win = True
            self.game_over = True

        # Обновление бонусов
        for b in self.bonuses[:]:
            if b['active']:
                # Проверка подбора бонуса
                if (self.player['x'] <= b['x'] < self.player['x'] + TANK_SIZE and
                    self.player['y'] <= b['y'] < self.player['y'] + TANK_SIZE):
                    if b['type'] == '❤️':
                        self.player['hp'] = min(5, self.player['hp'] + 1)
                    elif b['type'] == '⚡':
                        self.player['speed'] = min(3, self.player['speed'] + 0.5)
                    elif b['type'] == '🛡️':
                        self.player['shield'] = True
                    self.bonuses.remove(b)
                    continue
                # Время жизни бонуса
                b['life'] = b.get('life', 100) - 1
                if b.get('life', 0) <= 0:
                    self.bonuses.remove(b)

        # Обновление взрывов
        for e in self.explosions[:]:
            e['life'] -= 1
            if e['life'] <= 0:
                self.explosions.remove(e)

        # Обновление щита
        if self.player['shield']:
            self.player['shield_timer'] = self.player.get('shield_timer', 0) - 1
            if self.player.get('shield_timer', 0) <= 0:
                self.player['shield'] = False

    def draw(self):
        os.system('cls' if os.name == 'nt' else 'clear')
        
        # Создаём сетку
        grid = [[' ' for _ in range(self.width)] for _ in range(self.height)]

        # Стены
        for x, y in self.walls:
            if 0 <= y < self.height and 0 <= x < self.width:
                grid[y][x] = Fore.WHITE + WALL_CHAR + Style.RESET_ALL

        # Враги
        for e in self.enemies:
            if e['alive']:
                ch = Fore.RED + '▼' + Style.RESET_ALL
                for dy in range(TANK_SIZE):
                    for dx in range(TANK_SIZE):
                        ex, ey = e['x'] + dx, e['y'] + dy
                        if 0 <= ey < self.height and 0 <= ex < self.width:
                            if dx == 1 and dy == 1:
                                grid[ey][ex] = ch
                            else:
                                grid[ey][ex] = Fore.RED + '█' + Style.RESET_ALL

        # Игрок
        if self.player['alive']:
            dir_chars = {'up': '▲', 'down': '▼', 'left': '◄', 'right': '►'}
            ch = Fore.GREEN + dir_chars.get(self.player['dir'], '▲') + Style.RESET_ALL
            for dy in range(TANK_SIZE):
                for dx in range(TANK_SIZE):
                    px, py = self.player['x'] + dx, self.player['y'] + dy
                    if 0 <= py < self.height and 0 <= px < self.width:
                        if dx == 1 and dy == 1:
                            grid[py][px] = ch
                        else:
                            grid[py][px] = Fore.GREEN + '█' + Style.RESET_ALL

        # Пули
        for b in self.bullets:
            bx, by = int(b['x']), int(b['y'])
            if 0 <= by < self.height and 0 <= bx < self.width:
                color = Fore.YELLOW if b['is_player'] else Fore.RED
                grid[by][bx] = color + BULLET_CHAR + Style.RESET_ALL

        # Бонусы
        for b in self.bonuses:
            if b['active'] and 0 <= b['y'] < self.height and 0 <= b['x'] < self.width:
                grid[b['y']][b['x']] = Fore.YELLOW + b['type'] + Style.RESET_ALL

        # Взрывы
        for e in self.explosions:
            ex, ey = int(e['x']), int(e['y'])
            if 0 <= ey < self.height and 0 <= ex < self.width:
                ch = random.choice(EXPLOSION_CHARS)
                grid[ey][ex] = Fore.RED + ch + Style.RESET_ALL

        # Рендеринг
        print('┌' + '─' * self.width + '┐')
        for row in grid:
            line = '│' + ''.join(row) + '│'
            print(line)
        print('└' + '─' * self.width + '┘')

        # UI
        hp_bar = '█' * self.player['hp'] + '░' * (5 - self.player['hp'])
        shield_str = '🛡️ ' if self.player['shield'] else ''
        print(f"HP: {Fore.GREEN}{hp_bar}{Style.RESET_ALL}  Score: {self.player['score']}  Level: {self.level}  {shield_str}  Best: {self.high_score}")

        if self.game_over:
            if self.win:
                print(Fore.CYAN + "🎉 ПОБЕДА! Вы прошли уровень! Нажмите R для рестарта" + Style.RESET_ALL)
            else:
                print(Fore.RED + "💀 ИГРА ОКОНЧЕНА! Нажмите R для рестарта" + Style.RESET_ALL)

    def handle_input(self):
        if self.game_over:
            return

        # Простой неблокирующий ввод
        import msvcrt if os.name == 'nt' else select, sys, tty, termios
        # Для простоты используем блокирующий ввод с проверкой
        # В реальном коде используйте библиотеку keyboard или pynput

    def run(self):
        print(Fore.CYAN + "Танки 8-бит - Управление: WASD/Стрелки, Space - стрельба, Q - выход" + Style.RESET_ALL)
        print("Нажмите любую клавишу для начала...")
        input()

        while self.running:
            self.update()
            self.draw()
            time.sleep(0.05)

if __name__ == "__main__":
    game = TankGame()
    game.run()
