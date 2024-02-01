package world

import (
	"encoding/json"
	"fmt"
	"os"
	"sao/battle"
	"sao/player"
	"sao/world/calendar"
	"sao/world/location"
	"sao/world/npc"
	"sao/world/party"
	"time"

	"github.com/google/uuid"
)

type FloorMap map[string]location.Floor

type World struct {
	Players  map[uuid.UUID]*player.Player
	NPCs     map[uuid.UUID]npc.NPC
	Floors   FloorMap
	Fights   map[uuid.UUID]battle.Fight
	Entities map[uuid.UUID]*battle.Entity
	Time     *calendar.Calendar
	Parties  map[uuid.UUID]*party.Party
	TestMode bool
}

func CreateWorld(testMode bool) World {
	floorMap := make(FloorMap, 1)

	pathToRes := "./data/release/world/floors/"

	if testMode {
		pathToRes = "./data/test/world/floors/"
	}

	files, err := os.ReadDir(pathToRes)

	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		f, err := os.Open(pathToRes + file.Name())

		if err != nil {
			panic(err)
		}

		var floor location.Floor

		err = json.NewDecoder(f).Decode(&floor)

		if err != nil {
			panic(err)
		}

		floorMap[floor.Name] = floor

		fmt.Printf("Loaded floor %v\n", floor)
	}

	return World{
		make(map[uuid.UUID]*player.Player),
		make(map[uuid.UUID]npc.NPC),
		floorMap,
		make(map[uuid.UUID]battle.Fight),
		make(map[uuid.UUID]*battle.Entity),
		calendar.StartCalendar(),
		make(map[uuid.UUID]*party.Party),
		testMode,
	}
}

func (w *World) RegisterNewPlayer(name, uid string) player.Player {
	newPlayer := player.NewPlayer(name, uid)

	w.Players[newPlayer.GetUUID()] = &newPlayer

	return newPlayer
}

func (w *World) MovePlayer(pUuid uuid.UUID, floorName, locationName, reason string) {
	if len(reason) == 0 {
		fmt.Println("No reason for move, it was by player wish")
	}

	player := w.Players[pUuid]

	player.Meta.Location.FloorName = floorName
	player.Meta.Location.LocationName = locationName
}

func (w *World) PlayerEncounter(uuid uuid.UUID) {
	//TODO: Implement actual encounters
	// 	player := w.Players[uuid]

	// 	floor := w.Floors[player.Meta.Location.FloorName]
	// 	location := floor.FindLocation(player.Meta.Location.LocationName)

	// 	randomEnemy := utils.RandomElement[mobs.MobType](location.Enemies)

	// 	enemies := mobs.MobEncounter(randomEnemy)

	// 	entityMap := make(battle.EntityMap)

	// 	entityMap[player.GetUUID()] = battle.EntityEntry{Entity: player, Side: 0}

	// 	for _, enemy := range enemies {
	// 		entityMap[enemy.GetUUID()] = battle.EntityEntry{Entity: enemy, Side: 1}
	// 	}

	// 	fight := battle.Fight{Entities: entityMap}

	// 	fight.Init(w.Time.Copy())

	// 	fightUUID := w.RegisterFight(fight)

	// 	player.Meta.FightInstance = &fightUUID

	// 	go w.ListenForFight(fightUUID)
}

func (w *World) StartClock() {
	for range time.Tick(1 * time.Minute) {
		w.Time.Tick()

		for pUuid, player := range w.Players {
			//Not in fight
			if player.Meta.FightInstance != nil {
				continue
			}

			//Dead
			if player.GetCurrentHP() == 0 {
				continue
			}

			//Missing mana
			if player.GetCurrentMana() < player.GetMaxMana() {
				player.Stats.CurrentMana += 1
			}

			//Can be healed
			if player.GetCurrentHP() < player.GetMaxHP() {
				healRatio := 50

				if w.PlayerInCity(pUuid) {
					healRatio = 100
				}

				player.Heal(player.GetMaxHP() / healRatio)
			}
		}
	}
}

func (w *World) PlayerInCity(uuid uuid.UUID) bool {
	player := w.Players[uuid]

	floor := w.Floors[player.Meta.Location.FloorName]
	location := floor.FindLocation(player.Meta.Location.LocationName)

	return location.CityPart
}

// Fight stuff
func (w *World) RegisterFight(fight battle.Fight) uuid.UUID {
	uuid := uuid.New()

	w.Fights[uuid] = fight

	for _, entity := range fight.Entities {
		if entity.Entity.IsAuto() {
			w.Entities[entity.Entity.GetUUID()] = &entity.Entity
		}
	}

	return uuid
}

func (w *World) ListenForFight(fightUuid uuid.UUID) {
	fight, ok := w.Fights[fightUuid]

	if !ok {
		return
	}

	go fight.Run()

	for {
		payload, ok := <-fight.ExternalChannel

		if !ok {
			w.DeregisterFight(fightUuid)
			break
		}

		msgType := payload[0]

		switch battle.FightMessage(msgType) {
		case battle.MSG_FIGHT_END:
			wonSideIDX := fight.Entities.SidesLeft()[0]

			allLoot := make([]battle.Loot, 0)

			for _, entity := range fight.Entities {
				if entity.Side == wonSideIDX {
					continue
				}

				allLoot = append(allLoot, entity.Entity.GetLoot()...)
			}

			wonEntities := fight.Entities.FromSide(wonSideIDX)

			for _, entity := range wonEntities {
				if !entity.IsAuto() {
					entity.(battle.PlayerEntity).ReceiveMultipleLoot(allLoot)
				}
			}

		default:
			panic("Unhandled event")
		}

		//Fallback for when the channel is closed
		if fight.IsFinished() {
			w.DeregisterFight(fightUuid)
			break
		}
	}
}

func (w *World) DeregisterFight(uuid uuid.UUID) {
	tmp := w.Fights[uuid]

	for _, entity := range tmp.Entities {
		if !entity.Entity.IsAuto() {
			entity.Entity.(*player.Player).Meta.FightInstance = nil
		} else {
			delete(w.Entities, entity.Entity.GetUUID())
		}
	}

	delete(w.Fights, uuid)
}
