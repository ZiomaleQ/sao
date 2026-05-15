# SAO: Discord RPG Simulation Engine

A high-performance, event-driven RPG world simulation built in Go, designed for Discord-based roleplay communities. This engine handles complex combat, player progression, and world state management with a focus on modularity and extensibility.

## Key Architectural Features

*   **Custom Scripting Integration:** Uses the `parts` VM to load game data (mobs, floors, items) via external `.pts` files. This allows for live-reloading game balance and content without recompiling the core engine.
*   **Event-Driven Combat System:** Features a non-blocking combat loop using Go channels. It supports complex mechanics like speed-based turn order (Speed Gauge), status effects (Stun, Taunt, Vampirism), and reactive triggers (Counter-attacks).
*   **Concurrent World Clock:** A centralized clock manages player regeneration, status expirations, and automated backups every 15 minutes to prevent data loss.
*   **Dynamic Stat Engine:** Implements a sophisticated stat calculation system that handles base stats, level-based scaling, item modifiers, and party-role buffs (DPS/Tank/Support) in real-time.
*   **Automated Tournaments:** A built-in tournament manager that handles registration, bracket generation (including "lucky player" byes), and automated match orchestration.

## Tech Stack

*   **Language:** Go 1.22
*   **Framework:** [Disgo](https://github.com/disgoorg/disgo) (Discord API Wrapper)
*   **State Management:** In-memory with JSON-based persistence and automated backups.
*   **Data Injection:** Custom VM-based scripting via `github.com/tfo-dot/parts`.

## Game Mechanics Included

- **Party System:** 6-player parties with specialized roles and scaling bonuses.
- **Skill Trees:** Level-dependent skill unlocking and multi-tier upgrades.
- **Inventory System:** Item consumption, cooldown management, and hidden quest items.
- **Exploration:** Channel-specific encounter tables with randomized mob counts.

## Project Structure

- `/battle`: Core combat state machine and event handlers.
- `/world`: Global state management, including players, parties, and tournaments.
- `/game`: The "Data" layer—contains the `.pts` scripts for game content.
- `/player`: Inventory, skill-tree logic, and stat derivation.
- `/discord`: The UI/UX layer using Discord buttons, select menus, and modals.
