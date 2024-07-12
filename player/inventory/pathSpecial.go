package inventory

import (
	"sao/battle"
	"sao/types"
	"sao/utils"

	"github.com/google/uuid"
)

type SpecialSkill struct{}

func (skill SpecialSkill) Execute(owner, target, fightInstance, meta interface{}) interface{} {
	return nil
}

func (skill SpecialSkill) GetPath() types.SkillPath {
	return types.PathSpecial
}

func (skill SpecialSkill) GetUUID() uuid.UUID {
	return uuid.Nil
}

func (skill SpecialSkill) IsLevelSkill() bool {
	return true
}

type SPC_LVL_1 struct {
	SpecialSkill
	DefaultCost
	NoEvents
	NoStats
	DefaultActiveTrigger
}

func (skill SPC_LVL_1) GetName() string {
	return "Poziom 1 - Specjalista"
}

func (skill SPC_LVL_1) UpgradableExecute(owner, target, fightInstance, meta interface{}, upgrades int) interface{} {
	baseIncrease := 10
	baseDuration := 1

	if HasUpgrade(upgrades, 2) {
		baseIncrease = 12
	}

	if HasUpgrade(upgrades, 3) {
		baseDuration++
	}

	randomStat := utils.RandomElement(
		[]types.Stat{types.STAT_DEF, types.STAT_MR, types.STAT_SPD, types.STAT_AD, types.STAT_AP},
	)

	fightInstance.(*battle.Fight).HandleAction(battle.Action{
		Event:  battle.ACTION_EFFECT,
		Target: target.(battle.Entity).GetUUID(),
		Source: owner.(battle.PlayerEntity).GetUUID(),
		Meta: battle.ActionEffect{
			Effect:   battle.EFFECT_STAT_INC,
			Value:    baseIncrease,
			Duration: baseDuration,
			Caster:   owner.(battle.PlayerEntity).GetUUID(),
			Meta: battle.ActionEffectStat{
				Stat:      randomStat,
				Value:     baseIncrease,
				IsPercent: true,
			},
		},
	})

	return nil
}

func (skill SPC_LVL_1) GetCD() int {
	return BaseCooldowns[skill.GetLevel()]
}

func (skill SPC_LVL_1) GetCooldown(upgrades int) int {
	baseCD := skill.GetCD()

	if HasUpgrade(upgrades, 1) {
		return baseCD - 1
	}

	return baseCD
}

func (skill SPC_LVL_1) GetDescription() string {
	return "Zwiększa losowy atrybut o 10% na jedną turę"
}

func (skill SPC_LVL_1) GetLevel() int {
	return 1
}

func (skill SPC_LVL_1) GetUpgrades() []PlayerSkillUpgrade {
	return []PlayerSkillUpgrade{
		{
			Name:        "Ulepszenie 1",
			Id:          "Cooldown",
			Events:      nil,
			Description: "Zmniejsza czas odnowienia o 1 turę",
		},
		{
			Name:        "Ulepszenie 2",
			Id:          "Percent",
			Events:      nil,
			Description: "Zwiększa wartość atrybutu do 12%",
		},
		{
			Name:        "Ulepszenie 3",
			Id:          "Duration",
			Events:      nil,
			Description: "Zwiększa czas trwania o 1 turę",
		},
	}
}

type SPC_LVL_2 struct {
	SpecialSkill
	NoExecute
	NoEvents
	NoTrigger
}

func (skill SPC_LVL_2) GetName() string {
	return "Poziom 2 - Specjalista"
}

func (skill SPC_LVL_2) GetDescription() string {
	return "Dostajesz 5 kradzieży życia"
}

func (skill SPC_LVL_2) GetLevel() int {
	return 2
}

func (skill SPC_LVL_2) GetStats(upgrades int) map[types.Stat]int {
	stats := map[types.Stat]int{
		types.STAT_ATK_VAMP: 5,
	}

	vampValue := 5
	vampType := types.STAT_ATK_VAMP

	if HasUpgrade(upgrades, 1) {
		vampType = types.STAT_OMNI_VAMP
	}

	if HasUpgrade(upgrades, 2) {
		vampValue = 10
	}

	if HasUpgrade(upgrades, 3) {
		stats[types.STAT_HEAL_SELF] = 20
	}

	stats[vampType] = vampValue

	return stats
}

func (skill SPC_LVL_2) GetUpgrades() []PlayerSkillUpgrade {
	return []PlayerSkillUpgrade{
		{
			Name:        "Ulepszenie 1",
			Id:          "Skill",
			Events:      nil,
			Description: "Kradzież życia działa na umiejętności",
		},
		{
			Name:        "Ulepszenie 2",
			Id:          "Increase",
			Events:      nil,
			Description: "Zwiększa wartości dwukrotnie",
		},
		{
			Name:        "Ulepszenie 3",
			Id:          "ShieldInc",
			Events:      nil,
			Description: "Moc leczenia i tarcz (na sobie) zwiększona o 20%",
		},
	}
}

type SPC_LVL_3 struct {
	SpecialSkill
	DefaultCost
	NoEvents
	NoStats
	DefaultActiveTrigger
}

func (skill SPC_LVL_3) GetName() string {
	return "Poziom 3 - Specjalista"
}

func (skill SPC_LVL_3) UpgradableExecute(owner, target, fightInstance, meta interface{}, upgrades int) interface{} {
	baseDmg := 25
	baseHeal := 20

	if HasUpgrade(upgrades, 2) {
		baseDmg = utils.PercentOf(owner.(battle.PlayerEntity).GetStat(types.STAT_AP), 2) + utils.PercentOf(owner.(battle.PlayerEntity).GetStat(types.STAT_AD), 2)
	}

	if HasUpgrade(upgrades, 3) {
		baseHeal = 25
	}

	healValue := utils.PercentOf(baseDmg, baseHeal)

	fightInstance.(*battle.Fight).HandleAction(battle.Action{
		Event:  battle.ACTION_DMG,
		Target: target.(battle.Entity).GetUUID(),
		Source: owner.(battle.PlayerEntity).GetUUID(),
		Meta: battle.ActionDamage{
			Damage: []battle.Damage{
				{
					Type:  battle.DMG_TRUE,
					Value: baseDmg,
				},
			},
		},
	})

	fightInstance.(*battle.Fight).HandleAction(battle.Action{
		Event:  battle.ACTION_EFFECT,
		Target: owner.(battle.PlayerEntity).GetUUID(),
		Source: owner.(battle.PlayerEntity).GetUUID(),
		Meta: battle.ActionEffect{
			Effect:   battle.EFFECT_HEAL_SELF,
			Value:    healValue,
			Duration: 0,
			Caster:   owner.(battle.PlayerEntity).GetUUID(),
		},
	})

	return nil
}

func (skill SPC_LVL_3) GetCD() int {
	return BaseCooldowns[skill.GetLevel()]
}

func (skill SPC_LVL_3) GetCooldown(upgrades int) int {
	baseCD := skill.GetCD()

	if HasUpgrade(upgrades, 1) {
		return baseCD - 1
	}

	return baseCD
}

func (skill SPC_LVL_3) GetDescription() string {
	return "Zadaje 25 obrażeń i leczy o 20% tej wartości"
}

func (skill SPC_LVL_3) GetLevel() int {
	return 3
}

func (skill SPC_LVL_3) GetUpgrades() []PlayerSkillUpgrade {
	return []PlayerSkillUpgrade{
		{
			Name:        "Ulepszenie 1",
			Id:          "Cooldown",
			Events:      nil,
			Description: "Zmniejsza czas odnowienia o 1 turę",
		},
		{
			Name:        "Ulepszenie 2",
			Id:          "Damage",
			Events:      nil,
			Description: "Zwiększa obrażenia o 2%AP + 2%AD",
		},
		{
			Name:        "Ulepszenie 3",
			Id:          "Heal",
			Events:      nil,
			Description: "Zwiększa leczenie do 25%",
		},
	}
}

type SPC_LVL_4 struct {
	SpecialSkill
	NoEvents
	NoStats
	DefaultActiveTrigger
}

func (skill SPC_LVL_4) GetName() string {
	return "Poziom 4 - Specjalista"
}

func (skill SPC_LVL_4) UpgradableExecute(owner, target, fightInstance, meta interface{}, upgrades int) interface{} {
	tempSkill := target.(battle.PlayerEntity).GetLvlSkill(meta.(types.SkillChoice).Choice)

	owner.(battle.PlayerEntity).AppendTempSkill(types.WithExpire[types.PlayerSkill]{
		Value:      tempSkill,
		Expire:     1,
		AfterUsage: HasUpgrade(upgrades, 3),
	})

	return nil
}

func (skill SPC_LVL_4) GetCD() int {
	return BaseCooldowns[skill.GetLevel()] + 1
}

func (skill SPC_LVL_4) GetCooldown(upgrades int) int {
	baseCD := skill.GetCD()

	if HasUpgrade(upgrades, 1) {
		return baseCD - 1
	}

	return baseCD
}

func (skill SPC_LVL_4) GetDescription() string {
	return "Pożycza umiejętność sojusznika"
}

func (skill SPC_LVL_4) GetLevel() int {
	return 4
}

func (skill SPC_LVL_4) GetUpgrades() []PlayerSkillUpgrade {
	return []PlayerSkillUpgrade{
		{
			Name:        "Ulepszenie 1",
			Id:          "Cooldown",
			Events:      nil,
			Description: "Zmniejsza czas odnowienia o 1 turę",
		},
		{
			Name:        "Ulepszenie 2",
			Id:          "Cost",
			Events:      nil,
			Description: "Zmniejsza koszt o 1",
		},
		{
			Name:        "Ulepszenie 3",
			Id:          "Duration",
			Events:      nil,
			Description: "Umiejętność wygasa do końca walki",
		},
	}
}

func (skill SPC_LVL_4) GetUpgradableCost(upgrades int) int {
	if HasUpgrade(upgrades, 2) {
		return 1
	}

	return 2
}

func (skill SPC_LVL_4) GetCost() int {
	return 2
}