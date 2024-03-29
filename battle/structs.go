package battle

import (
	"fmt"
	"sao/types"
	"sao/utils"
	"sao/world/calendar"
	"sao/world/location"
	"sort"

	"github.com/disgoorg/disgo/discord"
	"github.com/google/uuid"
)

type Fight struct {
	Entities        EntityMap
	SpeedMap        map[uuid.UUID]int
	StartTime       *calendar.Calendar
	ActionChannel   chan Action
	ExternalChannel chan []byte
	DiscordChannel  chan types.DiscordMessageStruct
	Effects         []ActionEffect
	Location        *location.Location
	EventHandlers   map[uuid.UUID]types.Skill
}

type EntityMap map[uuid.UUID]EntityEntry

type EntityEntry struct {
	Entity Entity
	Side   int
}

func (f Fight) GetEntity(uuid uuid.UUID) (Entity, int) {
	entry, ok := f.Entities[uuid]

	if ok {
		return entry.Entity, entry.Side
	}

	return nil, -1
}

func (mp EntityMap) SidesLeft() []int {
	sides := make([]int, 0)

	for _, entity := range mp {

		if entity.Entity.GetCurrentHP() <= 0 {
			continue
		}

		exists := false

		for _, side := range sides {
			if side == entity.Side {
				exists = true
			}
		}

		if !exists {
			sides = append(sides, entity.Side)
		}
	}

	return sides
}

func (mp EntityMap) FromSide(side int) []Entity {
	entities := make([]Entity, 0)

	for _, entity := range mp {
		if entity.Side == side {
			entities = append(entities, entity.Entity)
		}
	}

	return entities
}

func (f *Fight) TriggerPassive(entityUuid uuid.UUID, triggerType types.SkillTrigger, additionalCheck func(Entity, types.PlayerSkill) bool) {
	entityEntry, exists := f.Entities[entityUuid]

	if !exists {
		return
	}

	sourceEntity := entityEntry.Entity

	if sourceEntity.IsAuto() {
		return
	}

	for _, skill := range sourceEntity.(PlayerEntity).GetAllSkills() {
		if skill.Trigger.Type == types.TRIGGER_ACTIVE {
			continue
		}

		if skill.Trigger.Event.TriggerType != triggerType {
			continue
		}

		if !additionalCheck(sourceEntity, skill) {
			continue
		}

		if skill.CD != 0 {
			sourceEntity.(PlayerEntity).SetCD(skill.UUID, skill.CD)
		}

		if skill.Cost != 0 {
			sourceEntity.(PlayerEntity).RestoreMana(-skill.Cost)
		}

		targets := f.FindValidTargets(sourceEntity.GetUUID(), *skill.Trigger.Event)

		if skill.Trigger.Event.TargetCount != -1 {
			count := skill.Trigger.Event.TargetCount

			if count > len(targets) {
				count = len(targets)
			}

			targets = targets[:count]
		}

		for _, target := range targets {
			skill.Action(sourceEntity, f.Entities[target].Entity, f)
		}
	}
}

func (f *Fight) DispatchActionAttack(act Action) (int, bool) {
	sourceEntity := f.Entities[act.Source]

	tempMeta := act.Meta

	if _, ok := tempMeta.(ActionDamage); !ok {
		tempMeta = ActionDamage{
			Damage: []Damage{{
				Value:    sourceEntity.Entity.GetATK(),
				Type:     DMG_PHYSICAL,
				CanDodge: true,
			}},
			CanDodge: true,
		}
	}

	meta := tempMeta.(ActionDamage)

	canDodge := meta.CanDodge && f.Entities[act.Target].Entity.CanDodge()

	dodged := false
	atk := 0

	if canDodge {
		atk, dodged = f.Entities[act.Target].Entity.(DodgeEntity).TakeDMGOrDodge(meta)
	} else {
		atk = f.Entities[act.Target].Entity.TakeDMG(meta)
	}

	if !dodged {
		f.TriggerPassive(act.Source, types.TRIGGER_HIT, func(e Entity, ps types.PlayerSkill) bool { return true })
	}

	return atk, dodged
}

func (f *Fight) HandleAction(act Action) {
	switch act.Event {
	case ACTION_ATTACK:
		dmgDealt, dodged := f.DispatchActionAttack(act)

		sourceEntity := f.Entities[act.Source]

		tempMeta := act.Meta

		if _, ok := tempMeta.(ActionDamage); !ok {
			tempMeta = ActionDamage{
				Damage: []Damage{
					{
						Value:    sourceEntity.Entity.GetATK(),
						Type:     DMG_PHYSICAL,
						CanDodge: true,
					},
				},
				CanDodge: true,
			}

		}

		meta := tempMeta.(ActionDamage)

		f.TriggerPassive(act.Source, types.TRIGGER_ATTACK, func(e Entity, ps types.PlayerSkill) bool { return true })

		messageBuilder := discord.NewMessageCreateBuilder()

		tempEmbed := discord.NewEmbedBuilder().
			SetTitle("Atak")

		if len(meta.Damage) == 1 {
			dmgType := "fizycznych"

			switch meta.Damage[0].Type {
			case DMG_PHYSICAL:
				dmgType = "fizycznych"
			case DMG_MAGICAL:
				dmgType = "magicznych"
			case DMG_TRUE:
				dmgType = "prawdziwych"
			}

			if dodged {
				tempEmbed.SetDescriptionf("%s zaatakował %s, ale atak został uniknięty", sourceEntity.Entity.GetName(), f.Entities[act.Target].Entity.GetName())
			} else {
				tempEmbed.SetDescriptionf("%s zaatakował %s, zadając %d obrażeń %s", sourceEntity.Entity.GetName(), f.Entities[act.Target].Entity.GetName(), dmgDealt, dmgType)
			}

			targetEntity := f.Entities[act.Target]

			tempEmbed.AddField(
				"Stat check",
				fmt.Sprintf(
					"Nazwa: %s\nHP: %v/%v\nATK/AP: %v/%v\nDEF/RES: %v/%v",
					targetEntity.Entity.GetName(),
					targetEntity.Entity.GetCurrentHP(),
					targetEntity.Entity.GetMaxHP(),
					targetEntity.Entity.GetATK(),
					targetEntity.Entity.GetAP(),
					targetEntity.Entity.GetDEF(),
					targetEntity.Entity.GetMR(),
				),
				false,
			)
		}

		messageBuilder.AddEmbeds(tempEmbed.Build())

		if sourceEntity.Entity.HasEffect(EFFECT_VAMP) && !dodged {
			effect := sourceEntity.Entity.GetEffect(EFFECT_VAMP)

			value := utils.PercentOf(dmgDealt, effect.Value)

			sourceEntity.Entity.Heal(value)

			messageBuilder.AddEmbeds(
				discord.
					NewEmbedBuilder().
					SetTitle("Efekt!").
					AddField(
						"Wampiryzm",
						fmt.Sprintf(
							"%s wyleczył się za %d punktów zdrowia",
							sourceEntity.Entity.GetName(),
							value,
						),
						false,
					).
					Build(),
			)
		}

		targetEntity := f.Entities[act.Target]

		if !targetEntity.Entity.IsAuto() {
			if targetEntity.Entity.(PlayerEntity).GetDefendingState() {
				if utils.RandomNumber(0, 100) < targetEntity.Entity.GetAGL() {
					counterDmg := utils.PercentOf(targetEntity.Entity.GetATK(), 70)

					counterDmg += utils.PercentOf(targetEntity.Entity.GetDEF(), 15)
					counterDmg += utils.PercentOf(targetEntity.Entity.GetMR(), 15)

					f.HandleAction(Action{
						Event:  ACTION_COUNTER,
						Source: act.Target,
						Target: act.Source,
						Meta: ActionDamage{
							Damage: []Damage{
								{
									Value:    counterDmg,
									Type:     DMG_PHYSICAL,
									CanDodge: true,
								},
							},
							CanDodge: true,
						},
					})
				}
			}
		}

		f.DiscordChannel <- types.DiscordMessageStruct{
			ChannelID:      f.Location.CID,
			MessageContent: messageBuilder.Build(),
		}
	case ACTION_EFFECT:
		meta := act.Meta.(ActionEffect)

		if meta.Duration == 0 {
			switch meta.Effect {
			case EFFECT_HEAL:
				f.Entities[act.Target].Entity.Heal(meta.Value)
			case EFFECT_STAT_INC:
				f.Entities[act.Target].Entity.ApplyEffect(meta)
			}
		}

		f.Entities[act.Target].Entity.ApplyEffect(act.Meta.(ActionEffect))
	case ACTION_DEFEND:
		entity := f.Entities[act.Source]

		f.TriggerPassive(act.Source, types.TRIGGER_DEFEND, func(e Entity, ps types.PlayerSkill) bool { return true })

		if !entity.Entity.IsAuto() {
			entity.Entity.(PlayerEntity).SetDefendingState(true)

			f.HandleAction(Action{
				Event:  ACTION_EFFECT,
				Source: act.Source,
				Target: act.Source,
				Meta: ActionEffect{
					Effect:   EFFECT_STAT_INC,
					Duration: 1,
					Meta: &map[string]interface{}{
						"stat":     STAT_DEF,
						"percent":  20,
						"duration": 1,
					},
				},
			})

			f.HandleAction(Action{
				Event:  ACTION_EFFECT,
				Source: act.Source,
				Target: act.Source,
				Meta: ActionEffect{
					Effect:   EFFECT_STAT_INC,
					Duration: 0,
					Meta: &map[string]interface{}{
						"stat":     STAT_DEF,
						"percent":  20,
						"duration": 1,
					},
				},
			})
		}
	case ACTION_SKILL:
		sourceEntity := f.Entities[act.Source]

		if !sourceEntity.Entity.IsAuto() {
			for _, skill := range sourceEntity.Entity.(PlayerEntity).GetAllSkills() {
				if skill.Trigger.Type == types.TRIGGER_ACTIVE {
					continue
				}

				if skill.Trigger.Event.TriggerType != types.TRIGGER_MANA {
					continue
				}

				if skill.Trigger.Event.Meta["value"].(int) > sourceEntity.Entity.GetCurrentMana() {
					continue
				}

				targets := f.FindValidTargets(sourceEntity.Entity.GetUUID(), *skill.Trigger.Event)

				if skill.Trigger.Event.TargetCount != -1 {
					count := skill.Trigger.Event.TargetCount

					if count > len(targets) {
						count = len(targets)
					}

					targets = targets[:count]
				}

				for _, target := range targets {
					targetEntity := f.Entities[target]

					beforeSkillHP := targetEntity.Entity.GetCurrentHP()

					skill.Action(sourceEntity.Entity, targetEntity, f)

					//Check if it's dmg skill so it doesn't trigger on heal/barrier etc
					if !targetEntity.Entity.IsAuto() && beforeSkillHP > targetEntity.Entity.GetCurrentHP() {
						if targetEntity.Entity.(PlayerEntity).GetDefendingState() {
							if utils.RandomNumber(0, 100) < targetEntity.Entity.GetAGL() {
								f.HandleAction(Action{
									Event:  ACTION_ATTACK,
									Source: act.Target,
									Target: act.Source,
									Meta: ActionDamage{
										Damage: []Damage{{
											Value:    targetEntity.Entity.GetATK(),
											Type:     DMG_PHYSICAL,
											CanDodge: true,
										}},
										CanDodge: true,
									},
								})

							}
						}
					}
				}
			}

			skillUsageMeta := act.Meta.(ActionSkillMeta)

			if skillUsageMeta.IsForLevel {
				skill := sourceEntity.Entity.(PlayerEntity).GetLvlSkill(skillUsageMeta.Lvl)

				if skill.Trigger.Type != types.TRIGGER_ACTIVE {
					return
				}

				if act.Target == uuid.Nil {
					for _, target := range skillUsageMeta.Targets {
						skill.Action(sourceEntity.Entity, f.Entities[target].Entity, f)
					}
				} else {
					skill.Action(sourceEntity.Entity, f.Entities[act.Target].Entity, f)
				}

				f.DiscordChannel <- types.DiscordMessageStruct{
					ChannelID: f.Location.CID,
					MessageContent: discord.
						NewMessageCreateBuilder().
						AddEmbeds(
							discord.NewEmbedBuilder().
								SetTitle("Skill!").
								SetDescriptionf(
									"%s użył %s!\n",
									sourceEntity.Entity.GetName(),
									skill.Name,
								).
								Build(),
						).
						Build(),
				}
			}
		}
	case ACTION_DMG:
		targetEntity := f.Entities[act.Target]
		meta := act.Meta.(ActionDamage)

		if targetEntity.Entity.GetCurrentHP() <= 0 {
			return
		}

		if meta.CanDodge && targetEntity.Entity.CanDodge() {
			targetEntity.Entity.(DodgeEntity).TakeDMGOrDodge(meta)
		} else {
			targetEntity.Entity.TakeDMG(meta)
		}

		if targetEntity.Entity.GetCurrentHP() <= 0 {
			f.TriggerPassive(act.Source, types.TRIGGER_EXECUTE, func(e Entity, ps types.PlayerSkill) bool { return true })
		}

		f.TriggerPassive(act.Source, types.TRIGGER_HIT, func(e Entity, ps types.PlayerSkill) bool {
			hpValue := 0

			if ps.Trigger.Event.Meta["value"] != nil {
				hpValue = ps.Trigger.Event.Meta["value"].(int)
			} else {
				hpValue = (ps.Trigger.Event.Meta["percent"].(int) * e.GetMaxHP() / 100)
			}

			return hpValue < e.GetCurrentHP()
		})

		f.TriggerPassive(act.Source, types.TRIGGER_HIT, func(e Entity, ps types.PlayerSkill) bool {
			hpValue := 0

			if ps.Trigger.Event.Meta["value"] != nil {
				hpValue = ps.Trigger.Event.Meta["value"].(int)
			} else {
				hpValue = (ps.Trigger.Event.Meta["percent"].(int) * e.GetMaxHP() / 100)
			}

			return hpValue < e.GetCurrentHP()
		})
	case ACTION_COUNTER:
		dmgDealt, dodged := f.DispatchActionAttack(act)

		sourceEntity := f.Entities[act.Source]

		meta := act.Meta.(ActionDamage)

		messageBuilder := discord.NewMessageCreateBuilder()

		tempEmbed := discord.NewEmbedBuilder().
			SetTitle("Kontra!")

		if len(meta.Damage) == 1 {
			dmgType := "fizycznych"

			switch meta.Damage[0].Type {
			case DMG_PHYSICAL:
				dmgType = "fizycznych"
			case DMG_MAGICAL:
				dmgType = "magicznych"
			case DMG_TRUE:
				dmgType = "prawdziwych"
			}

			if dodged {
				tempEmbed.SetDescriptionf("%s skontrował %s, ale kontra została uniknięta", sourceEntity.Entity.GetName(), f.Entities[act.Target].Entity.GetName())
			} else {
				tempEmbed.SetDescriptionf("%s skontrował %s, zadając %d obrażeń %s", sourceEntity.Entity.GetName(), f.Entities[act.Target].Entity.GetName(), dmgDealt, dmgType)
			}
		}

		if sourceEntity.Entity.HasEffect(EFFECT_VAMP) && !dodged {
			effect := sourceEntity.Entity.GetEffect(EFFECT_VAMP)

			value := utils.PercentOf(dmgDealt, effect.Value)

			sourceEntity.Entity.Heal(value)

			messageBuilder.AddEmbeds(
				discord.
					NewEmbedBuilder().
					SetTitle("Efekt!").
					AddField(
						"Wampiryzm",
						fmt.Sprintf(
							"%s wyleczył się za %d punktów zdrowia",
							sourceEntity.Entity.GetName(),
							value,
						),
						false,
					).
					Build(),
			)
		}

		if !dodged {
			f.TriggerPassive(act.Source, types.TRIGGER_ATTACK, func(e Entity, ps types.PlayerSkill) bool { return true })
		}

		messageBuilder.AddEmbeds(tempEmbed.Build())

		f.DiscordChannel <- types.DiscordMessageStruct{
			ChannelID:      f.Location.CID,
			MessageContent: messageBuilder.Build(),
		}

		targetEntity := f.Entities[act.Target]

		if !targetEntity.Entity.IsAuto() {
			if targetEntity.Entity.(PlayerEntity).GetDefendingState() {
				if utils.RandomNumber(0, 100) < targetEntity.Entity.GetAGL() {

					counterDmg := utils.PercentOf(targetEntity.Entity.GetATK(), 70)

					counterDmg += utils.PercentOf(targetEntity.Entity.GetDEF(), 15)
					counterDmg += utils.PercentOf(targetEntity.Entity.GetMR(), 15)

					f.HandleAction(Action{
						Event:  ACTION_COUNTER,
						Source: act.Target,
						Target: act.Source,
						Meta: ActionDamage{
							Damage: []Damage{
								{
									Value:    counterDmg,
									Type:     DMG_PHYSICAL,
									CanDodge: true,
								},
							},
							CanDodge: true,
						},
					})
				}
			}
		}
	case ACTION_ITEM:
		sourceEntity := f.Entities[act.Source]

		itemMeta := act.Meta.(ActionItemMeta)

		var item *types.PlayerItem
		var itemIdx int

		for idx, invItem := range sourceEntity.Entity.(PlayerEntity).GetAllItems() {
			if invItem.UUID == itemMeta.Item {
				item = invItem
				itemIdx = idx
				break
			}
		}

		if act.Target == uuid.Nil {
			for _, target := range itemMeta.Targets {
				item.UseItem(sourceEntity.Entity, f.Entities[target].Entity, nil)
			}
		} else {
			item.UseItem(sourceEntity.Entity, f.Entities[act.Target].Entity, nil)
		}

		if item.Count == 0 && item.Consume {
			sourceEntity.Entity.(PlayerEntity).RemoveItem(itemIdx)
		}

		f.DiscordChannel <- types.DiscordMessageStruct{
			ChannelID: f.Location.CID,
			MessageContent: discord.
				NewMessageCreateBuilder().
				AddEmbeds(
					discord.NewEmbedBuilder().
						SetTitle("Przedmiot!").
						SetDescriptionf(
							"%s użył %s!\nEfekt: %s",
							sourceEntity.Entity.GetName(),
							item.Name, item.Description,
						).
						Build(),
				).Build(),
		}
	case ACTION_RUN:
		entity, side := f.GetEntity(act.Source)

		if utils.RandomNumber(0, 100) < entity.GetAGL() {
			f.DiscordChannel <- types.DiscordMessageStruct{
				ChannelID: f.Location.CID,
				MessageContent: discord.
					NewMessageCreateBuilder().
					AddEmbeds(
						discord.NewEmbedBuilder().
							SetTitle("Ucieczka!").
							SetDescriptionf("%s próbował uciec, ale mu się to nie udało", entity.GetName()).
							SetColor(0xff0000).
							Build(),
					).Build(),
			}
		} else {

			delete(f.Entities, act.Source)

			entities := f.Entities.FromSide(side)

			count := 0

			for _, entity := range entities {
				if entity.GetCurrentHP() > 0 && !entity.IsAuto() {
					count++
				}
			}

			f.DiscordChannel <- types.DiscordMessageStruct{
				ChannelID: f.Location.CID,
				MessageContent: discord.
					NewMessageCreateBuilder().
					AddEmbeds(
						discord.NewEmbedBuilder().
							SetTitle("Ucieczka!").
							SetDescriptionf("%s próbował uciec i mu się to udało", entity.GetName()).
							SetColor(0x00ff00).
							Build(),
					).Build(),
			}

			if count == 0 {
				f.ExternalChannel <- []byte{byte(MSG_FIGHT_END)}
			}
		}
	default:
		fmt.Printf("Unknown action %d\n", act.Event)
		panic("Not implemented (actions)")
	}
}

func (f *Fight) Init() {
	f.SpeedMap = make(map[uuid.UUID]int)
	f.ActionChannel = make(chan Action, 10)

	for _, entity := range f.Entities {
		f.SpeedMap[entity.Entity.GetUUID()] = entity.Entity.GetSPD()
	}

	f.ExternalChannel = make(chan []byte)

	//FIGHT START EVENT
	for entityUuid := range f.Entities {
		f.TriggerPassive(entityUuid, types.TRIGGER_FIGHT_START, func(e Entity, ps types.PlayerSkill) bool { return true })
	}
}

func (f *Fight) FindValidTargets(source uuid.UUID, trigger types.EventTriggerDetails) []uuid.UUID {
	sourceEntity, sourceSide := f.GetEntity(source)

	if len(trigger.TargetType) == 1 && trigger.TargetType[0] == types.TARGET_SELF {
		return []uuid.UUID{source}
	}

	targetEntities := make([]Entity, 0)

	for _, targetType := range trigger.TargetType {
		if targetType == types.TARGET_SELF {
			targetEntities = append(targetEntities, sourceEntity)
		}
	}

	isAllyValid := false

	for _, targetType := range trigger.TargetType {
		if targetType == types.TARGET_ALLY {
			isAllyValid = true
		}
	}

	isEnemyValid := false

	for _, targetType := range trigger.TargetType {
		if targetType == types.TARGET_ENEMY {
			isEnemyValid = true
		}
	}

	for _, entity := range f.Entities {
		if entity.Side == sourceSide && isAllyValid {
			targetEntities = append(targetEntities, entity.Entity)
		}

		if entity.Side != sourceSide && isEnemyValid {
			targetEntities = append(targetEntities, entity.Entity)
		}
	}

	sortInit := EntitySort{
		Entities: targetEntities,
		Order:    trigger.TargetDetails,
		Meta:     &trigger.Meta,
	}

	sort.Sort(sortInit)

	targets := make([]uuid.UUID, len(targetEntities))

	for i, entity := range sortInit.Entities {
		targets[i] = entity.GetUUID()
	}

	return targets
}

func (f *Fight) Run() {
	f.ExternalChannel <- []byte{byte(MSG_FIGHT_START)}

	for len(f.Entities.SidesLeft()) > 1 {
		turnList := make([]uuid.UUID, 0)

		for uuid, speed := range f.SpeedMap {
			entity, _ := f.GetEntity(uuid)

			f.SpeedMap[uuid] = speed + entity.GetSPD()

			turns := f.SpeedMap[uuid] / SPEED_GAUGE

			if turns == 0 {
				continue
			}

			f.SpeedMap[uuid] -= turns * SPEED_GAUGE

			for i := 0; i < turns; i++ {
				turnList = append(turnList, entity.GetUUID())
			}
		}

		for _, entityUuid := range turnList {
			entity, _ := f.GetEntity(entityUuid)

			if entity.GetCurrentHP() <= 0 {
				continue
			}

			f.TriggerPassive(entityUuid, types.TRIGGER_TURN, func(e Entity, ps types.PlayerSkill) bool { return true })

			if !entity.IsAuto() {
				bytes, _ := entity.GetUUID().MarshalBinary()
				packet := make([]byte, 1+len(bytes))
				packet[0] = byte(MSG_ACTION_NEEDED)
				copy(packet[1:], bytes)

				f.ExternalChannel <- packet

				f.HandleAction(<-f.ActionChannel)
			} else {
				if !(entity.HasEffect(EFFECT_DISARM) || entity.HasEffect(EFFECT_STUN) || entity.HasEffect(EFFECT_STUN) || entity.HasEffect(EFFECT_ROOT) || entity.HasEffect(EFFECT_GROUND) || entity.HasEffect(EFFECT_BLIND)) {
					actionNum := entity.Action(f)

					for i := 0; i < actionNum; i++ {
						f.HandleAction(<-f.ActionChannel)
					}
				} else {
					f.DiscordChannel <- types.DiscordMessageStruct{
						ChannelID: f.Location.CID,
						MessageContent: discord.
							NewMessageCreateBuilder().
							AddEmbeds(
								discord.NewEmbedBuilder().
									SetTitle("Efekt!").
									SetDescriptionf("%s jest unieruchomiony, pomijamy!", entity.GetName()).
									Build(),
							).Build(),
					}
				}

			}

			expired := entity.TriggerAllEffects()

			for _, effect := range expired {
				if effect.Uuid != uuid.Nil {
					effectHandler, exists := f.EventHandlers[effect.Uuid]

					if exists {
						delete(f.EventHandlers, effect.Uuid)

						caster := effectHandler.Trigger.Event.Meta["source"]

						effectHandler.Execute(f.Entities[caster.(uuid.UUID)].Entity, f.Entities[entityUuid].Entity, nil)

						continue
					}
				}
			}
		}

		for _, entry := range f.Entities {
			if entry.Entity.GetCurrentHP() <= 0 {
				continue
			}
		}
	}

	//FIGHT END EVENT
	for entityUuid := range f.Entities {
		f.TriggerPassive(entityUuid, types.TRIGGER_FIGHT_END, func(e Entity, ps types.PlayerSkill) bool { return true })
	}

	f.ExternalChannel <- []byte{byte(MSG_FIGHT_END)}
}

func (f *Fight) IsFinished() bool {
	return len(f.Entities.SidesLeft()) <= 1
}

func (f *Fight) GetEnemiesFor(uuid uuid.UUID) []Entity {
	_, userSide := f.GetEntity(uuid)

	enemiesList := make([]Entity, 0)

	for _, entry := range f.Entities {
		if entry.Side == userSide {
			continue
		}

		if entry.Entity.GetCurrentHP() <= 0 {
			continue
		}

		enemiesList = append(enemiesList, entry.Entity)
	}

	return enemiesList
}
