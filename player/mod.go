package player

import (
	"encoding/json"
	"errors"
	"os"
	"sao/battle"
	"sao/battle/mobs"
	"sao/config"
	"sao/data"
	"sao/player/inventory"
	"sao/types"
	"sao/utils"
	"sao/world/fury"
	"strconv"

	"github.com/google/uuid"
)

type PlayerStats struct {
	HP          int
	Effects     mobs.EffectList
	Defending   bool
	CurrentMana int
}

type PlayerXP struct {
	Level int
	Exp   int
}

type PlayerMeta struct {
	Location      types.EntityLocation
	OwnUUID       uuid.UUID
	UserID        string
	FightInstance *uuid.UUID
	Party         *uuid.UUID
	Transaction   *uuid.UUID
	Fury          *fury.Fury
	WaitToHeal    bool
}

func (pM *PlayerMeta) SerializeFuries() map[string]interface{} {
	if pM.Fury == nil {
		return nil
	}

	return pM.Fury.Serialize()
}

func (pM *PlayerMeta) Serialize() map[string]interface{} {
	party := ""
	if pM.Party != nil {
		party = pM.Party.String()
	}

	return map[string]interface{}{
		"location": []string{pM.Location.Floor, pM.Location.Location},
		"uuid":     pM.OwnUUID.String(),
		"uid":      pM.UserID,
		"fury":     pM.SerializeFuries(),
		"party":    party,
	}
}

type Player struct {
	Name         string
	XP           PlayerXP
	Stats        PlayerStats
	Meta         PlayerMeta
	Inventory    inventory.PlayerInventory
	DynamicStats []types.DerivedStat
	LevelStats   map[types.Stat]int
	DefaultStats map[types.Stat]int
}

func (p *Player) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"name": p.Name,
		"xp":   []int{p.XP.Level, p.XP.Exp},
		"stats": map[string]interface{}{
			"hp":           p.Stats.HP,
			"current_mana": p.Stats.CurrentMana,
			"effects":      p.Stats.Effects,
		},
		"dynamic_stats": p.DynamicStats,
		"level_stats":   p.LevelStats,
		"default_stats": p.DefaultStats,
		"meta":          p.Meta.Serialize(),
		"inventory":     p.Inventory.Serialize(),
	}
}

func DeserializeDerivedStats(data []interface{}) []types.DerivedStat {
	tempArray := make([]types.DerivedStat, 0)

	for _, stat := range data {
		stat := stat.(map[string]interface{})

		temp := types.DerivedStat{
			Derived: types.Stat(stat["derived"].(float64)),
			Base:    types.Stat(stat["base"].(float64)),
			Percent: int(stat["percent"].(float64)),
			Source:  uuid.MustParse(stat["source"].(string)),
		}

		tempArray = append(tempArray, temp)
	}

	return tempArray
}

func DeserializeLevelStats(data map[string]interface{}) map[types.Stat]int {
	tempMap := make(map[types.Stat]int, 0)

	for key, value := range data {
		val, _ := strconv.Atoi(key)

		tempMap[types.Stat(val)] = int(value.(float64))
	}

	return tempMap
}

func DeserializeDefaultStats(data map[string]interface{}) map[types.Stat]int {
	tempMap := make(map[types.Stat]int, 0)

	for key, value := range data {
		val, _ := strconv.Atoi(key)

		tempMap[types.Stat(val)] = int(value.(float64))
	}

	return tempMap
}

func Deserialize(data map[string]interface{}) *Player {
	return &Player{
		data["name"].(string),
		PlayerXP{
			Level: int(data["xp"].([]interface{})[0].(float64)),
			Exp:   int(data["xp"].([]interface{})[1].(float64)),
		},
		PlayerStats{
			int(data["stats"].(map[string]interface{})["hp"].(float64)),
			DeserializeEffects(data["stats"].(map[string]interface{})["effects"].([]interface{})),
			false,
			int(data["stats"].(map[string]interface{})["current_mana"].(float64)),
		},
		*DeserializeMeta(data["meta"].(map[string]interface{})),
		inventory.DeserializeInventory(data["inventory"].(map[string]interface{})),
		DeserializeDerivedStats(data["dynamic_stats"].([]interface{})),
		DeserializeLevelStats(data["level_stats"].(map[string]interface{})),
		DeserializeDefaultStats(data["default_stats"].(map[string]interface{})),
	}
}

func DeserializeEffects(data []interface{}) mobs.EffectList {
	temp := make(mobs.EffectList, 0)

	if len(data) == 0 {
		return temp
	}

	for _, effect := range data {
		effect := effect.(map[string]interface{})

		temp = append(temp, battle.ActionEffect{
			Effect: battle.Effect(effect["effect"].(float64)),
			Value:  int(effect["value"].(float64)),
			Meta:   effect["meta"],
		})
	}

	return temp
}

func DeserializeMeta(data map[string]interface{}) *PlayerMeta {

	pLocation := types.EntityLocation{
		Floor:    data["location"].([]interface{})[0].(string),
		Location: data["location"].([]interface{})[1].(string),
	}

	var furyData *fury.Fury
	if data["fury"] != nil {
		furyData = fury.Deserialize(data["fury"].(map[string]interface{}))
	}

	var party *uuid.UUID

	if data["party"] != "" {
		temp := uuid.MustParse(data["party"].(string))

		party = &temp
	}

	return &PlayerMeta{
		pLocation,
		uuid.MustParse(data["uuid"].(string)),
		data["uid"].(string),
		nil,
		party,
		nil,
		furyData,
		false,
	}
}

func (p *Player) AppendTempSkill(skill types.WithExpire[types.PlayerSkill]) {
	p.Inventory.AddTempSkill(skill)
}

func (p *Player) GetCurrentMana() int {
	return p.Stats.CurrentMana
}

func (p *Player) GetUUID() uuid.UUID {
	return p.Meta.OwnUUID
}

func (p *Player) Heal(val int) {
	p.Stats.HP += val

	if p.Stats.HP > p.GetStat(types.STAT_HP) {
		p.Stats.HP = p.GetStat(types.STAT_HP)
	}
}

func (p *Player) GetFlags() types.EntityFlag {
	return 0
}

func (p *Player) SetDefendingState(state bool) {
	p.Stats.Defending = state
}

func (p *Player) GetDefendingState() bool {
	return p.Stats.Defending
}

func (p *Player) Action(f *battle.Fight) []battle.Action { return []battle.Action{} }

func (p *Player) TakeDMG(dmgList battle.ActionDamage) []battle.Damage {
	dmgStats := []battle.Damage{
		{Value: 0, Type: types.DMG_PHYSICAL},
		{Value: 0, Type: types.DMG_MAGICAL},
		{Value: 0, Type: types.DMG_TRUE},
	}

	for _, dmg := range dmgList.Damage {
		//Skip shield and such
		if dmg.Type == types.DMG_TRUE {
			p.Stats.HP -= dmg.Value
			dmgStats[2].Value += dmg.Value
			continue
		}

		rawDmg := dmg.Value

		switch dmg.Type {
		case types.DMG_PHYSICAL:
			rawDmg = utils.CalcReducedDamage(dmg.Value, p.GetStat(types.STAT_DEF))
		case types.DMG_MAGICAL:
			rawDmg = utils.CalcReducedDamage(dmg.Value, p.GetStat(types.STAT_MR))
		}

		actualDmg := p.DamageShields(rawDmg)

		dmgStats[dmg.Type].Value += actualDmg

		p.Stats.HP -= actualDmg
	}

	return dmgStats
}

func (p *Player) TakeDMGOrDodge(dmg battle.ActionDamage) ([]battle.Damage, bool) {
	if utils.RandomNumber(0, 100) <= p.GetStat(types.STAT_AGL) && dmg.CanDodge {
		return []battle.Damage{
			{Value: 0, Type: types.DMG_PHYSICAL},
			{Value: 0, Type: types.DMG_MAGICAL},
			{Value: 0, Type: types.DMG_TRUE},
		}, true
	}

	return p.TakeDMG(dmg), false
}

func (p *Player) DamageShields(dmg int) int {
	leftOverDmg := dmg
	idxToRemove := make([]int, 0)

	for idx, effect := range p.Stats.Effects {
		if effect.Effect == battle.EFFECT_SHIELD {
			newShieldValue := effect.Value - leftOverDmg

			if newShieldValue <= 0 {
				leftOverDmg = newShieldValue * -1

				idxToRemove = append(idxToRemove, idx)
			} else {
				effect.Value = newShieldValue
				leftOverDmg = 0
			}
		}
	}

	for _, idx := range idxToRemove {
		p.Stats.Effects = append(p.Stats.Effects[:idx], p.Stats.Effects[idx+1:]...)
	}

	return leftOverDmg
}

func (p *Player) AddEXP(maxFloor, value int) {
	p.XP.Exp += value

	if p.XP.Level >= maxFloor*5 {
		p.XP.Level = maxFloor * 5
		p.XP.Exp = 0
		return
	}

	if p.Meta.Fury != nil {
		p.Meta.Fury.AddXP(utils.PercentOf(value, 20))
	}

	for p.XP.Exp >= ((p.XP.Level * 100) + 100) {
		if p.XP.Level >= maxFloor*5 {
			p.XP.Level = maxFloor * 5
			p.XP.Exp = 0
			return
		}

		p.XP.Exp -= ((p.XP.Level * 100) + 100)
		p.LevelUP()
	}
}

func (p *Player) LevelUP() {
	p.XP.Level++

	if p.Stats.HP < p.GetStat(types.STAT_HP) {
		p.Stats.HP = utils.PercentOf(p.GetStat(types.STAT_HP), 20) + p.GetCurrentHP()
	}

	if p.Stats.HP > p.GetStat(types.STAT_HP) {
		p.Stats.HP = p.GetStat(types.STAT_HP)
	}
}

func (p *Player) GetCurrentHP() int {
	return p.Stats.HP
}

func (p *Player) AddGold(value int) {
	p.Inventory.Gold += value
}

func (p *Player) GetLoot() []battle.Loot {
	return nil
}

func (p *Player) GetName() string {
	return p.Name
}

func (p *Player) CanDodge() bool {
	return !p.Stats.Defending
}

func (p *Player) ApplyEffect(e battle.ActionEffect) {
	p.Stats.Effects = append(p.Stats.Effects, e)
}

func (p *Player) GetEffectByType(effect battle.Effect) *battle.ActionEffect {
	return p.Stats.Effects.GetEffectByType(effect)
}

func (p *Player) GetEffectByUUID(uuid uuid.UUID) *battle.ActionEffect {
	return p.Stats.Effects.GetEffectByUUID(uuid)
}

func (p *Player) TriggerAllEffects() []battle.ActionEffect {
	effects, expiredEffects := p.Stats.Effects.TriggerAllEffects(p)

	p.Stats.Effects = effects

	return expiredEffects
}

func (p *Player) RemoveEffect(uuid uuid.UUID) {
	p.Stats.Effects = p.Stats.Effects.RemoveEffect(uuid)
}

func (p *Player) GetAllEffects() []battle.ActionEffect {
	return p.Stats.Effects
}

func (p *Player) CanAttack() bool {
	return p.GetEffectByType(battle.EFFECT_STUN) == nil
}

func (p *Player) CanDodgeNow() bool {
	return p.GetEffectByType(battle.EFFECT_STUN) == nil
}

func (p *Player) CanDefend() bool {
	return p.GetEffectByType(battle.EFFECT_STUN) == nil
}

func (p *Player) GetTempSkills() []*types.WithExpire[types.PlayerSkill] {
	return p.Inventory.TempSkills
}

func (p *Player) RemoveTempByUUID(uuid uuid.UUID) {
	tempList := make([]*types.WithExpire[types.PlayerSkill], 0)

	for _, skill := range p.Inventory.TempSkills {
		if skill.Value.GetUUID() != uuid {
			tempList = append(tempList, skill)
		}
	}

	p.Inventory.TempSkills = tempList
}

func (p *Player) CanUseSkill(skill types.PlayerSkill) bool {
	if skill.IsLevelSkill() {
		lvl := skill.(types.PlayerSkillLevel).GetLevel()

		skillTrigger := skill.(types.PlayerSkillUpgradable).GetUpgradableTrigger(p.Inventory.LevelSkillsUpgrades[lvl])

		if skillTrigger.Type == types.TRIGGER_PASSIVE {
			return false
		}

		if p.GetEffectByType(battle.EFFECT_STUN) != nil {
			return skillTrigger.Flags&types.FLAG_IGNORE_CC != 0
		}

		if currentCD, onCooldown := p.Inventory.LevelSkillsCDS[lvl]; onCooldown && currentCD != 0 {
			return false
		}
	} else {
		skillTrigger := skill.GetTrigger()

		if skillTrigger.Type == types.TRIGGER_PASSIVE {
			return false
		}

		if p.GetEffectByType(battle.EFFECT_STUN) != nil {
			return skillTrigger.Flags&types.FLAG_IGNORE_CC != 0
		}

		if currentCD, onCooldown := p.Inventory.ItemSkillCD[skill.GetUUID()]; onCooldown && currentCD != 0 {
			return false
		}

		if currentCD, onCooldown := p.Inventory.FurySkillsCD[skill.GetUUID()]; onCooldown && currentCD != 0 {
			return false
		}
	}

	if p.GetCurrentMana() < skill.GetCost() {
		return false
	}

	return true
}

func (p *Player) AddItem(item *types.PlayerItem) {

	for _, effect := range item.Effects {
		effectEvents := effect.GetEvents()

		effectEvents[types.CUSTOM_TRIGGER_UNLOCK](p)
	}

	p.Inventory.Items = append(p.Inventory.Items, item)
}

func (p *Player) GetAllItems() []*types.PlayerItem {
	return p.Inventory.Items
}

func (p *Player) GetSkills() []types.PlayerSkill {
	arr := make([]types.PlayerSkill, 0)

	for _, skill := range p.Inventory.LevelSkills {
		arr = append(arr, skill)
	}

	return arr
}

func (p *Player) GetSkill(uuid uuid.UUID) types.PlayerSkill {
	for _, skill := range p.Inventory.TempSkills {
		if skill.Value.GetUUID() == uuid {
			return skill.Value
		}
	}

	return nil
}

func (p *Player) RemoveItem(item int) {
	p.Inventory.Items = append(p.Inventory.Items[:item], p.Inventory.Items[item+1:]...)
}

func (p *Player) RestoreMana(value int) {
	p.Stats.CurrentMana += value

	if p.Stats.CurrentMana > p.GetStat(types.STAT_MANA) {
		p.Stats.CurrentMana = p.GetStat(types.STAT_MANA)
	}
}

func (p *Player) GetDefaultStat(stat types.Stat) int {
	if value, ok := p.DefaultStats[stat]; ok {
		return value
	}
	return 0
}

func (p *Player) GetStat(stat types.Stat) int {
	switch stat {
	case types.STAT_MANA_PLUS:
		return p.GetDefaultStat(types.STAT_MANA) - p.GetStat(types.STAT_MANA)
	case types.STAT_HP_PLUS:
		return p.GetDefaultStat(types.STAT_HP) + ((p.XP.Level - 1) * 10) - p.GetStat(types.STAT_HP)
	}

	statValue := p.GetDefaultStat(stat)
	percentValue := 0

	for _, effect := range p.GetAllEffects() {
		if effect.Effect == battle.EFFECT_STAT_INC {

			if value, ok := effect.Meta.(battle.ActionEffectStat); ok {
				if value.Stat != stat {
					continue
				}

				if value.IsPercent {
					percentValue += value.Value
				} else {
					statValue += value.Value
				}
			}
		}

		if effect.Effect == battle.EFFECT_STAT_DEC {

			if value, ok := effect.Meta.(battle.ActionEffectStat); ok {
				if value.Stat != stat {
					continue
				}

				if value.IsPercent {
					percentValue -= value.Value
				} else {
					statValue -= value.Value
				}
			}
		}
	}

	if value, ok := p.LevelStats[stat]; ok {
		statValue += ((p.XP.Level - 1) * value)
	}

	statValue += p.Inventory.GetStat(stat)

	if p.Meta.Fury != nil {
		statValue += p.Meta.Fury.GetStat(stat)
	}

	for _, effect := range p.DynamicStats {
		if effect.Derived == stat {
			statValue += utils.PercentOf(p.GetStat(effect.Base), effect.Percent)
		}
	}

	if stat == types.STAT_AD || stat == types.STAT_AP {
		adaptive := p.GetStat(types.STAT_ADAPTIVE)

		if adaptive > 0 {
			adaptiveType := p.GetAdaptiveAttackType()

			if adaptiveType == types.ADAPTIVE_ATK && stat == types.STAT_AD {
				statValue += adaptive
			}

			if adaptiveType == types.ADAPTIVE_AP && stat == types.STAT_AP {
				statValue += adaptive
			}
		}
	}

	baseStat := statValue + (statValue * percentValue / 100)

	if stat == types.STAT_AD || stat == types.STAT_AP {
		adaptive := p.GetStat(types.STAT_ADAPTIVE_PERCENT)

		if adaptive > 0 {
			adaptiveType := p.GetAdaptiveAttackType()

			if adaptiveType == types.ADAPTIVE_ATK && stat == types.STAT_AD {
				baseStat += utils.PercentOf(baseStat, adaptive)
			}

			if adaptiveType == types.ADAPTIVE_AP && stat == types.STAT_AP {
				baseStat += utils.PercentOf(baseStat, adaptive)
			}
		}
	}
	return baseStat
}

func (p *Player) GetRawStat(stat types.Stat) int {
	switch stat {
	case types.STAT_MANA_PLUS:
		return p.GetDefaultStat(types.STAT_MANA) - p.GetStat(types.STAT_MANA)
	case types.STAT_HP_PLUS:
		return p.GetDefaultStat(types.STAT_HP) + ((p.XP.Level - 1) * 10) - p.GetStat(types.STAT_HP)
	}

	statValue := p.GetDefaultStat(stat)
	percentValue := 0

	for _, effect := range p.GetAllEffects() {
		if effect.Effect == battle.EFFECT_STAT_INC {

			if value, ok := effect.Meta.(battle.ActionEffectStat); ok {
				if value.Stat != stat {
					continue
				}

				if value.IsPercent {
					percentValue += value.Value
				} else {
					statValue += value.Value
				}
			}
		}

		if effect.Effect == battle.EFFECT_STAT_DEC {

			if value, ok := effect.Meta.(battle.ActionEffectStat); ok {
				if value.Stat != stat {
					continue
				}

				if value.IsPercent {
					percentValue -= value.Value
				} else {
					statValue -= value.Value
				}
			}
		}
	}

	if value, ok := p.LevelStats[stat]; ok {
		statValue += ((p.XP.Level - 1) * value)
	}

	statValue += p.Inventory.GetStat(stat)

	if p.Meta.Fury != nil {
		statValue += p.Meta.Fury.GetStat(stat)
	}

	for _, effect := range p.DynamicStats {
		if effect.Derived == stat {
			statValue += utils.PercentOf(p.GetStat(effect.Base), effect.Percent)
		}
	}

	return statValue + (statValue * percentValue / 100)
}

func (p *Player) GetAdaptiveAttackType() types.AdaptiveAttackType {
	adStat := p.GetRawStat(types.STAT_AD)
	apStat := p.GetRawStat(types.STAT_AP)

	if apStat > adStat {
		return types.ADAPTIVE_AP
	}

	return types.ADAPTIVE_ATK
}

func (p *Player) Cleanse() {
	p.Stats.Effects = p.Stats.Effects.Cleanse()
}

func (p *Player) GetUpgrades(lvl int) int {
	return p.Inventory.LevelSkillsUpgrades[lvl]
}

func (p *Player) GetLvlSkill(lvl int) types.PlayerSkill {
	skill, skillExists := p.Inventory.LevelSkills[lvl]

	if !skillExists {
		return nil
	}

	return skill
}

func (p *Player) GetUID() string {
	return p.Meta.UserID
}

func (p *Player) GetLvlCD(lvl int) int {
	return p.Inventory.LevelSkillsCDS[lvl]
}

func (p *Player) SetLvlCD(lvl int, value int) {
	if value == 0 {
		delete(p.Inventory.LevelSkillsCDS, lvl)
	} else {
		p.Inventory.LevelSkillsCDS[lvl] = value
	}
}

func (p *Player) GetLevelSkillsCD() map[int]int {
	mapTemp := make(map[int]int, 0)

	for skill, cd := range p.Inventory.LevelSkillsCDS {
		mapTemp[skill] = cd
	}

	return mapTemp
}

func (p *Player) AppendDerivedStat(stat types.DerivedStat) {
	p.DynamicStats = append(p.DynamicStats, stat)
}

func (p *Player) SetLevelStat(stat types.Stat, value int) {
	p.LevelStats[stat] = value
}

func (p *Player) ReduceCooldowns(event types.SkillTrigger) {
	for skill := range p.Inventory.ItemSkillCD {
		item := data.Items[utils.SkillUUIDToItemUUID(skill)]

		for _, effect := range item.Effects {
			cdMeta := effect.GetTrigger().Cooldown

			if cdMeta == nil && event == types.TRIGGER_TURN {
				p.Inventory.ItemSkillCD[skill]--
			}

			if cdMeta != nil && event != cdMeta.PassEvent {
				continue
			}
		}
	}

	for skillLevel := range p.Inventory.LevelSkillsCDS {
		skillData := p.Inventory.LevelSkills[skillLevel]
		cdMeta := skillData.GetUpgradableTrigger(skillLevel).Cooldown

		if cdMeta == nil && event == types.TRIGGER_TURN {
			p.Inventory.LevelSkillsCDS[skillLevel]--
		}

		if cdMeta != nil && event != cdMeta.PassEvent {
			continue
		}
	}

	for skillUuid := range p.Inventory.FurySkillsCD {
		if p.Meta.Fury == nil {
			break
		}

		for _, skillStruct := range p.Meta.Fury.GetSkills() {
			if skillUuid == skillStruct.GetUUID() {
				cdMeta := skillStruct.GetTrigger().Cooldown

				if cdMeta == nil && event == types.TRIGGER_TURN {
					p.Inventory.FurySkillsCD[skillUuid]--
				}

				if cdMeta != nil && event != cdMeta.PassEvent {
					continue
				}
			}
		}
	}
}

func (p *Player) GetLvl() int {
	return p.XP.Level
}

func (p *Player) TriggerEvent(event types.SkillTrigger, data types.EventData, meta interface{}) []interface{} {
	returnMeta := make([]interface{}, 0)

	for _, item := range p.Inventory.Items {
		for _, effect := range item.Effects {
			trigger := effect.GetTrigger()

			if cd := effect.GetCD(); cd != 0 {
				currentCD, onCooldown := p.Inventory.ItemSkillCD[effect.GetUUID()]

				if onCooldown {
					if currentCD == 0 {
						p.Inventory.ItemSkillCD[effect.GetUUID()] = cd
					} else {
						continue
					}
				}
			}

			if trigger.Type == types.TRIGGER_PASSIVE && trigger.Event == event {
				if cost := effect.GetCost(); cost != 0 {
					if cost > p.GetCurrentMana() {
						continue
					} else {
						p.Stats.CurrentMana -= cost
					}
				}

				temp := effect.Execute(p, data.Target, data.Fight, meta)

				if temp != nil {
					returnMeta = append(returnMeta, temp)
				}
			}
		}
	}

	for skillLevel, skillStruct := range p.Inventory.LevelSkills {
		trigger := skillStruct.GetUpgradableTrigger(p.Inventory.LevelSkillsUpgrades[skillLevel])

		if cd := skillStruct.GetUpgradableCost(p.Inventory.LevelSkillsUpgrades[skillLevel]); cd != 0 {
			currentCD, onCooldown := p.Inventory.LevelSkillsCDS[skillLevel]

			if onCooldown {
				if currentCD == 0 {
					p.Inventory.LevelSkillsCDS[skillLevel] = cd
				} else {
					continue
				}
			}
		}

		if trigger.Type == types.TRIGGER_PASSIVE && trigger.Event == event {
			if cost := skillStruct.GetCost(); cost != 0 {
				if cost > p.GetCurrentMana() {
					continue
				} else {
					p.Stats.CurrentMana -= cost
				}
			}

			temp := skillStruct.Execute(p, data.Target, data.Fight, meta)

			if temp != nil {
				returnMeta = append(returnMeta, temp)
			}
		}
	}

	if p.Meta.Fury != nil {
		for _, skill := range p.Meta.Fury.GetSkills() {
			trigger := skill.GetTrigger()

			if cd := skill.GetCost(); cd != 0 {
				currentCD, onCooldown := p.Inventory.FurySkillsCD[skill.GetUUID()]

				if onCooldown {
					if currentCD == 0 {
						p.Inventory.FurySkillsCD[skill.GetUUID()] = cd
					} else {
						continue
					}
				}
			}

			if trigger.Type == types.TRIGGER_PASSIVE && trigger.Event == event {
				if cost := skill.GetCost(); cost != 0 {
					if cost > p.GetCurrentMana() {
						continue
					} else {
						p.Stats.CurrentMana -= cost
					}
				}

				temp := skill.Execute(p, data.Target, data.Fight, meta)

				if temp != nil {
					returnMeta = append(returnMeta, temp)
				}
			}
		}
	}

	for _, effect := range p.Inventory.TempSkills {
		trigger := effect.Value.GetTrigger()

		if trigger.Type == types.TRIGGER_PASSIVE && trigger.Event == event {
			if cost := effect.Value.GetCost(); cost != 0 {
				if cost > p.GetCurrentMana() {
					continue
				} else {
					p.Stats.CurrentMana -= cost
				}
			}

			temp := effect.Value.Execute(p, data.Target, data.Fight, meta)

			if temp != nil {
				returnMeta = append(returnMeta, temp)
			}
		}
	}

	return returnMeta
}

func (p *Player) UnlockSkill(path types.SkillPath, lvl, choice int) error {
	if lvl > p.GetLvl() {
		return errors.New("PLAYER_LVL_TOO_LOW")
	}

	if _, exists := p.Inventory.LevelSkills[lvl]; exists {
		return errors.New("SKILL_ALREADY_UNLOCKED")
	}

	if actions := p.GetAvailableSkillActions(); actions < 1 {
		return errors.New("NO_ACTIONS_AVAILABLE")
	}

	skill, skillExists := inventory.AVAILABLE_SKILLS[path][lvl]

	if !skillExists {
		return errors.New("SKILL_NOT_FOUND")
	}

	if choice >= len(skill) {
		return errors.New("INVALID_CHOICE")
	}

	p.Inventory.LevelSkills[lvl] = skill[choice]

	skillEvents := p.Inventory.LevelSkills[lvl].GetEvents()

	if effect, effectExists := skillEvents[types.CUSTOM_TRIGGER_UNLOCK]; effectExists {
		effect(p)
	}

	return nil
}

func (p *Player) UpgradeSkill(lvl int, upgradeIdx int) error {
	skill, exists := p.Inventory.LevelSkills[lvl]

	if !exists {
		return errors.New("SKILL_NOT_FOUND")
	}

	if upgradeIdx >= len(skill.GetUpgrades()) {
		return errors.New("INVALID_UPGRADE")
	}

	upgradeValue := 1 << upgradeIdx

	if p.Inventory.LevelSkillsUpgrades[lvl]&upgradeValue != 0 {
		return errors.New("UPGRADE_ALREADY_UNLOCKED")
	}

	if actions := p.GetAvailableSkillActions(); actions < 1 {
		return errors.New("NO_ACTIONS_AVAILABLE")
	}

	p.Inventory.LevelSkillsUpgrades[lvl] |= upgradeValue

	println(p.Inventory.LevelSkillsUpgrades[lvl])

	return nil
}

func (p *Player) TriggerTempSkills() {
	list := make([]*types.WithExpire[types.PlayerSkill], 0)

	for _, skill := range p.Inventory.TempSkills {
		if !skill.AfterUsage {
			skill.Expire--

			if skill.Expire > 0 {
				list = append(list, skill)
			} else {
				continue
			}
		}
	}

	p.Inventory.TempSkills = list
}

func (p *Player) ClearFight() {
	p.Meta.FightInstance = nil
}

func (p *Player) GetAvailableSkillActions() int {
	overall := 0

	if p.XP.Level < 6 {
		overall = p.XP.Level
	}

	{
		temp := p.XP.Level - 6

		for temp >= 2 {
			temp -= 2
			overall++
		}
	}

	used := len(p.Inventory.LevelSkills)

	for lvl, upgrades := range p.Inventory.LevelSkillsUpgrades {
		skill := p.Inventory.LevelSkills[lvl]

		if skill == nil {
			continue
		}

		skillUpgrades := skill.GetUpgrades()

		for i := 0; i < len(skillUpgrades); i++ {
			if upgrades&i != 0 {
				used++
			}
		}
	}

	return overall - used
}

func NewPlayer(name string, uid string) Player {
	return Player{
		name,
		PlayerXP{Level: 1, Exp: 0},
		PlayerStats{
			Default.StartingStats[types.STAT_HP],
			make(mobs.EffectList, 0),
			false,
			Default.StartingStats[types.STAT_MANA],
		},
		PlayerMeta{Default.Location, uuid.New(), uid, nil, nil, nil, nil, false},
		inventory.GetDefaultInventory(),
		make([]types.DerivedStat, 0),
		Default.LevelStats,
		Default.StartingStats,
	}
}

var Default PlayerDefaults = GetPlayerDefaults()

type PlayerDefaults struct {
	StartingStats map[types.Stat]int
	LevelStats    map[types.Stat]int
	Location      types.EntityLocation
}

func GetPlayerDefaults() PlayerDefaults {
	rawData, err := os.ReadFile(config.Config.GameDataLocation + "/players/default.json")

	if err != nil {
		panic(err)
	}

	var parsedData map[string]interface{}

	json.Unmarshal([]byte(rawData), &parsedData)

	pDefaults := PlayerDefaults{
		Location: types.EntityLocation{
			Floor:    parsedData["Location"].(map[string]interface{})["Floor"].(string),
			Location: parsedData["Location"].(map[string]interface{})["Location"].(string),
		},
	}

	pDefaults.StartingStats = make(map[types.Stat]int, 0)

	for key, value := range parsedData["Stats"].(map[string]interface{}) {
		pDefaults.StartingStats[utils.StringToStat[key]] = int(value.(float64))
	}

	pDefaults.LevelStats = make(map[types.Stat]int, 0)

	for key, value := range parsedData["LevelStats"].(map[string]interface{}) {
		pDefaults.LevelStats[utils.StringToStat[key]] = int(value.(float64))
	}

	return pDefaults
}
