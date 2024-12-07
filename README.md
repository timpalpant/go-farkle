# go-farkle
Solver for optimal play in the Farkle dice game

## How to run

### Solve the game
```bash
cd cmd/solve-farkle
go build
./solve-farkle -logtostderr -num_players 2 -db 2player.db
```

### Play the game using optimal solution
```bash
cd cmd/play-farkle
go build
./play-farkle -num_players 2 -db ../solve-farkle/2player.db
```

## Solution size

Scores are capped at 12,750 (255 * 50) to make the game play finite.

- 2 player: 99,488,250 states, 1.5 GiB
- 3 player: 25,369,503,750 states, 567 GiB
- 4 player: 6.4692235e+12 states, 188 TiB