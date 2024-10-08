package inventory

import (
	"fmt"
	"sao/types"
	"sao/utils"

	"github.com/google/uuid"
)

type EnduranceSkill struct{}

func (skill EnduranceSkill) Execute(owner types.PlayerEntity, target types.Entity, fightInstance types.FightInstance, meta interface{}) interface{} {
	return nil
}

func (skill EnduranceSkill) GetPath() types.SkillPath {
	return types.PathEndurance
}

func (skill EnduranceSkill) GetUUID() uuid.UUID {
	return uuid.Nil
}

func (skill EnduranceSkill) IsLevelSkill() bool {
	return true
}

func (skill EnduranceSkill) CanUse(owner types.PlayerEntity, fightInstance types.FightInstance) bool {
	return true
}

type END_LVL_1 struct {
	EnduranceSkill
	DefaultCost
	DefaultActiveTrigger
	NoEvents
	NoStats
}

func (skill END_LVL_1) GetName() string {
	return "Poziom 1 - wytrzymałość"
}

func (skill END_LVL_1) UpgradableExecute(owner types.PlayerEntity, target types.Entity, fightInstance types.FightInstance, meta interface{}) interface{} {
	baseIncrease := 10
	baseDuration := 2

	if HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 1) {
		baseIncrease = 20
		baseIncrease += utils.PercentOf(owner.GetStat(types.STAT_HP), 3)
		baseIncrease += utils.PercentOf(owner.GetStat(types.STAT_AD), 2)
	}

	if HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 2) {
		baseDuration++
	}

	fightInstance.HandleAction(types.Action{
		Event:  types.ACTION_EFFECT,
		Target: owner.GetUUID(),
		Source: owner.GetUUID(),
		Meta: types.ActionEffect{
			Effect:   types.EFFECT_SHIELD,
			Value:    baseIncrease,
			Duration: baseDuration,
			Caster:   owner.GetUUID(),
		},
	})

	return nil
}

func (skill END_LVL_1) GetCD() int {
	return BaseCooldowns[skill.GetLevel()]
}

func (skill END_LVL_1) GetCooldown(upgrades int) int {
	baseCD := skill.GetCD()

	if HasUpgrade(upgrades, 1) {
		return baseCD - 1
	}

	return baseCD
}

func (skill END_LVL_1) GetDescription() string {
	return "Otrzymujesz tarczę o wartości 10 na jedną turę"
}

func (skill END_LVL_1) GetLevel() int {
	return 1
}

func (skill END_LVL_1) GetUpgrades() []types.PlayerSkillUpgrade {
	return []types.PlayerSkillUpgrade{
		{
			Id:          "Cooldown",
			Description: "Zmniejsza czas odnowienia o 1 turę",
		},
		{
			Id:          "Damage",
			Description: "Zwiększa wartość tarczy do 20 + 3%HP + 2%ATK",
		},
		{
			Id:          "Duration",
			Description: "Zwiększa czas trwania o 1 turę",
		},
	}
}

func (skill END_LVL_1) GetUpgradableDescription(upgrades int) string {
	baseIncrease := "10"
	baseDuration := 1

	if HasUpgrade(upgrades, 1) {
		baseIncrease = "20 + 3%HP + 2%ATK"
	}

	if HasUpgrade(upgrades, 2) {
		baseDuration = 2
	}

	return fmt.Sprintf("Otrzymujesz tarczę o wartości %s na %d tur", baseIncrease, baseDuration)
}

type END_LVL_2 struct {
	EnduranceSkill
	NoExecute
	NoStats
	NoTrigger
}

func (skill END_LVL_2) GetEvents() map[types.CustomTrigger]func(owner types.PlayerEntity) {
	return map[types.CustomTrigger]func(owner types.PlayerEntity){
		types.CUSTOM_TRIGGER_UNLOCK: func(owner types.PlayerEntity) {
			owner.SetLevelStat(types.STAT_HP, owner.GetLevelStat(types.STAT_HP)+5)
		},
	}
}

func (skill END_LVL_2) GetUpgrades() []types.PlayerSkillUpgrade {
	return []types.PlayerSkillUpgrade{
		{
			Id: "Increase",
			Events: &map[types.CustomTrigger]func(owner types.PlayerEntity){
				types.CUSTOM_TRIGGER_UNLOCK: func(owner types.PlayerEntity) {
					owner.SetLevelStat(types.STAT_HP, owner.GetLevelStat(types.STAT_HP)+5)
				},
			},
			Description: "Zwiększa zdrowie otrzymywane co poziom o dodatkowe 5 (łączenie 10)",
		},
		{
			Id: "DEFIncrease",
			Events: &map[types.CustomTrigger]func(owner types.PlayerEntity){
				types.CUSTOM_TRIGGER_UNLOCK: func(owner types.PlayerEntity) {
					owner.AppendDerivedStat(types.DerivedStat{
						Base:    types.STAT_HP_PLUS,
						Derived: types.STAT_DEF,
						Percent: 1,
					})
				},
			},
			Description: "Otrzymujesz 1% dodatkowego HP jako pancerz",
		},
		{
			Id: "RESIncrease",
			Events: &map[types.CustomTrigger]func(owner types.PlayerEntity){
				types.CUSTOM_TRIGGER_UNLOCK: func(owner types.PlayerEntity) {
					owner.AppendDerivedStat(types.DerivedStat{
						Base:    types.STAT_HP_PLUS,
						Derived: types.STAT_MR,
						Percent: 1,
					})
				},
			},
			Description: "Otrzymujesz 1% dodatkowego HP jako odporność na magię",
		},
	}
}

func (skill END_LVL_2) GetDescription() string {
	return "Zwiększa zdrowie zyskiwane co poziom o 5"
}

func (skill END_LVL_2) GetLevel() int {
	return 2
}

func (skill END_LVL_2) GetName() string {
	return "Poziom 2 - wytrzymałość"
}

func (skill END_LVL_2) GetUpgradableDescription(upgrades int) string {
	baseIncrease := 5

	if HasUpgrade(upgrades, 1) {
		baseIncrease = 10
	}

	defUpgrade := ""

	if HasUpgrade(upgrades, 2) {
		defUpgrade = "\nOtrzymujesz pancerz równy 1% dodatkowego HP."
	}

	mrUpgrade := ""

	if HasUpgrade(upgrades, 3) {
		mrUpgrade = "\nOtrzymujesz odporność na magię równą 1% dodatkowego HP."
	}

	return fmt.Sprintf("Zwiększa zdrowie zyskiwane co poziom do %d.%s%s", baseIncrease, defUpgrade, mrUpgrade)
}

type END_LVL_3 struct {
	EnduranceSkill
	DefaultCost
	NoEvents
	NoStats
}

func (skill END_LVL_3) GetTrigger() types.Trigger {
	return types.Trigger{Type: types.TRIGGER_ACTIVE, Target: &types.TargetTrigger{Target: types.TARGET_ENEMY, MaxTargets: 1}}
}

func (skill END_LVL_3) GetUpgradableTrigger(upgrades int) types.Trigger {
	return skill.GetTrigger()
}

type END_LVL_3_EFFECT struct {
	NoCost
	NoCooldown
	NoEvents
	NoLevel
	Owner      uuid.UUID
	HealFactor int
	OnlyOwner  bool
}

func (effect END_LVL_3_EFFECT) Execute(owner types.PlayerEntity, target types.Entity, fightInstance types.FightInstance, meta interface{}) interface{} {
	if target.GetUUID() != effect.Owner && !effect.OnlyOwner {
		return nil
	}

	og := fightInstance.GetEntity(effect.Owner)

	fightInstance.HandleAction(types.Action{
		Event:  types.ACTION_EFFECT,
		Target: effect.Owner,
		Source: effect.Owner,
		Meta: types.ActionEffect{
			Effect: types.EFFECT_HEAL,
			Value:  utils.PercentOf(og.GetStat(types.STAT_HP), effect.HealFactor),
		},
	})

	return nil
}

func (effect END_LVL_3_EFFECT) GetDescription() string {
	return "Leczy o 5% maksymalnego HP celu"
}

func (effect END_LVL_3_EFFECT) GetName() string {
	return "Efekt wytrzymałości - poziom 3"
}

func (effect END_LVL_3_EFFECT) GetTrigger() types.Trigger {
	return types.Trigger{
		Type:  types.TRIGGER_PASSIVE,
		Event: types.TRIGGER_ATTACK_GOT_HIT,
	}
}

func (skill END_LVL_3) UpgradableExecute(owner types.PlayerEntity, target types.Entity, fightInstance types.FightInstance, meta interface{}) interface{} {
	healFactor := 5

	if HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 3) {
		healFactor += 5
	}

	target.AppendTempSkill(types.WithExpire[types.PlayerSkill]{
		Value: END_LVL_3_EFFECT{
			Owner:      owner.GetUUID(),
			HealFactor: healFactor,
			OnlyOwner:  HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 1),
		},
		AfterUsage: false,
		Expire:     1,
	})

	return nil
}

func (skill END_LVL_3) GetName() string {
	return "Poziom 3 - wytrzymałość"
}

func (skill END_LVL_3) GetLevel() int {
	return 3
}

func (skill END_LVL_3) GetCD() int {
	return BaseCooldowns[skill.GetLevel()]
}

func (skill END_LVL_3) GetCooldown(upgrades int) int {
	baseCD := skill.GetCD()

	if HasUpgrade(upgrades, 2) {
		return baseCD - 1
	}

	return baseCD
}

func (skill END_LVL_3) GetDescription() string {
	return "Oznacza cel na turę, atak rozbije oznaczenie i wyleczy o 5% maksymalnego HP celu"
}

func (skill END_LVL_3) GetUpgrades() []types.PlayerSkillUpgrade {
	return []types.PlayerSkillUpgrade{
		{
			Id:          "Ally",
			Description: "Sojusznik może rozbić oznaczenie",
		},
		{
			Id:          "Cooldown",
			Description: "Zmniejsza czas odnowienia o 1 turę",
		},
		{
			Id:          "Increase",
			Description: "Zwiększa leczenie o dodatkowe 5%",
		},
	}
}

func (skill END_LVL_3) GetUpgradableDescription(upgrades int) string {
	baseHeal := 5

	if HasUpgrade(upgrades, 3) {
		baseHeal = 10
	}

	allyUpgrade := ""

	if HasUpgrade(upgrades, 1) {
		allyUpgrade = " (także sojusznika)"
	}

	return fmt.Sprintf("Oznacza cel na turę, kolejny atak%s rozbije oznaczenie i wyleczy o %d%%HP", allyUpgrade, baseHeal)
}

type END_LVL_4 struct {
	EnduranceSkill
	DefaultCost
	DefaultActiveTrigger
	NoEvents
	NoStats
}

type END_LVL_4_EFFECT struct {
	NoCost
	NoCooldown
	NoEvents
	NoLevel
	Reduce int
	Event  types.SkillTrigger
}

func (skill END_LVL_4_EFFECT) Execute(owner types.PlayerEntity, target types.Entity, fightInstance types.FightInstance, meta interface{}) interface{} {
	if skill.Event == types.TRIGGER_ATTACK_GOT_HIT {
		tempMeta := meta.(types.AttackTriggerMeta)

		for idx, dmg := range tempMeta.Effects {
			tempMeta.Effects[idx].Value = -utils.PercentOf(dmg.Value, skill.Reduce)
		}

		return tempMeta
	}

	tempMeta := meta.(types.DamageTriggerMeta)

	for idx, dmg := range tempMeta.Effects {
		tempMeta.Effects[idx].Value = -utils.PercentOf(dmg.Value, skill.Reduce)
	}

	return tempMeta
}

func (skill END_LVL_4_EFFECT) GetDescription() string {
	return "Zmniejszy obrażenia kolejnego otrzymanego ataku o 20%"
}

func (skill END_LVL_4_EFFECT) GetName() string {
	return "Efekt wytrzymałości - poziom 4"
}

func (skill END_LVL_4_EFFECT) GetTrigger() types.Trigger {
	return types.Trigger{
		Type:  types.TRIGGER_PASSIVE,
		Event: skill.Event,
	}
}

func (skill END_LVL_4) UpgradableExecute(owner types.PlayerEntity, target types.Entity, fightInstance types.FightInstance, meta interface{}) interface{} {
	baseReduce := 20

	if HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 1) {
		baseReduce += 5
	}

	baseEvent := types.TRIGGER_ATTACK_GOT_HIT

	if HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 3) {
		baseEvent = types.TRIGGER_DAMAGE
	}

	owner.AppendTempSkill(types.WithExpire[types.PlayerSkill]{
		Value: END_LVL_4_EFFECT{
			Reduce: baseReduce,
			Event:  baseEvent,
		},
		AfterUsage: true,
		Expire:     1,
		Either:     HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 3),
	})

	return nil
}

func (skill END_LVL_4) GetName() string {
	return "Poziom 4 - wytrzymałość"
}

func (skill END_LVL_4) GetCD() int {
	return BaseCooldowns[skill.GetLevel()]
}

func (skill END_LVL_4) GetCooldown(upgrades int) int {
	return skill.GetCD()
}

func (skill END_LVL_4) GetDescription() string {
	return "Zmniejszy obrażenia kolejnego otrzymanego ataku o 20%"
}

func (skill END_LVL_4) GetLevel() int {
	return 4
}

func (skill END_LVL_4) GetUpgrades() []types.PlayerSkillUpgrade {
	return []types.PlayerSkillUpgrade{
		{
			Id:          "Reduce",
			Description: "Redukcja obrażeń zwiększona do 25%",
		},
		{
			Id:          "Duration",
			Description: "Efekt działa przez całą ture",
		},
		{
			Id:          "Type",
			Description: "Efekt działa na wszystkie obrażenia (wcześniej tylko ataki)",
		},
	}
}

func (skill END_LVL_4) GetUpgradableDescription(upgrades int) string {
	baseReduce := "20%"

	if HasUpgrade(upgrades, 1) {
		baseReduce = "25%"
	}

	baseDuration := "kolejny"

	if HasUpgrade(upgrades, 2) {
		baseDuration = "wszystkie"
	}

	baseEvent := "otrzymany atak"

	if HasUpgrade(upgrades, 3) {
		if HasUpgrade(upgrades, 2) {
			baseDuration = "kolejne"
		}
		baseEvent = "otrzymane obrażenia"
	}

	return fmt.Sprintf("Przez jedną turę zmniejsza %s %s o %s", baseDuration, baseEvent, baseReduce)
}

type END_LVL_5 struct {
	EnduranceSkill
	DefaultCost
	NoEvents
	NoStats
	DefaultActiveTrigger
}

func (skill END_LVL_5) GetName() string {
	return "Poziom 5 - wytrzymałość"
}

func (skill END_LVL_5) GetDescription() string {
	return "Otrzymujesz 25 tarczy za każdego (żywego) przeciwnika w walce i prowokujesz ich wszystkich"
}

func (skill END_LVL_5) GetLevel() int {
	return 5
}

func (skill END_LVL_5) GetCD() int {
	return BaseCooldowns[skill.GetLevel()]
}

func (skill END_LVL_5) GetCooldown(upgrades int) int {
	return skill.GetCD()
}

func (skill END_LVL_5) GetUpgrades() []types.PlayerSkillUpgrade {
	return []types.PlayerSkillUpgrade{
		{
			Id:          "Duration",
			Description: "Tarcza i prowokacja działają turę dłużej",
		},
		{
			Id:          "Increase",
			Description: "Zwiększa wartość tarczy o 2% HP + 20% AP",
		},
		{
			Id:          "Heal",
			Description: "Po wygaśnięciu tarczy leczy o 50% pozostałej wartości",
		},
	}
}

func (skill END_LVL_5) UpgradableExecute(owner types.PlayerEntity, target types.Entity, fightInstance types.FightInstance, meta interface{}) interface{} {
	enemiesCount := len(fightInstance.GetEnemiesFor(owner.GetUUID()))

	baseShield := 25

	if HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 2) {
		baseShield += utils.PercentOf(owner.GetStat(types.STAT_HP), 2)
		baseShield += utils.PercentOf(owner.GetStat(types.STAT_AP), 20)
	}

	duration := 1

	if HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 1) {
		duration++
	}

	shieldUuid := uuid.New()

	fightInstance.HandleAction(types.Action{
		Event:  types.ACTION_EFFECT,
		Target: owner.GetUUID(),
		Source: owner.GetUUID(),
		Meta: types.ActionEffect{
			Effect:   types.EFFECT_SHIELD,
			Value:    enemiesCount * baseShield,
			Duration: duration,
			Caster:   owner.GetUUID(),
			Uuid:     shieldUuid,
			OnExpire: func(owner types.Entity, fight types.FightInstance, meta types.ActionEffect) {
				if HasUpgrade(owner.(types.PlayerEntity).GetUpgrades(skill.GetLevel()), 3) {
					if meta.Value > 0 {
						healValue := utils.PercentOf(meta.Value, 50)

						fight.HandleAction(types.Action{
							Event:  types.ACTION_EFFECT,
							Target: owner.GetUUID(),
							Source: owner.GetUUID(),
							Meta: types.ActionEffect{
								Effect:   types.EFFECT_HEAL,
								Value:    healValue,
								Duration: 0,
							},
						})
					}
				}
			},
		},
	})

	fightInstance.HandleAction(types.Action{
		Event:  types.ACTION_EFFECT,
		Target: owner.GetUUID(),
		Source: owner.GetUUID(),
		Meta: types.ActionEffect{
			Effect:   types.EFFECT_TAUNT,
			Value:    0,
			Duration: duration,
			Caster:   owner.GetUUID(),
		},
	})

	return nil
}

func (skill END_LVL_5) GetUpgradableDescription(upgrades int) string {
	baseShield := "25"

	if HasUpgrade(upgrades, 2) {
		baseShield = "25 + 2% HP + 10% AP"
	}

	duration := "1"

	if HasUpgrade(upgrades, 1) {
		duration = "2"
	}

	heal := ""

	if HasUpgrade(upgrades, 3) {
		heal = " Po wygaśnięciu tarczy leczysz się o 50% pozostałej wartości"
	}

	return fmt.Sprintf("Otrzymujesz %s tarczy za każdego przeciwnika prowokując wszystkich wrogów. Tarcza i prowokacja działają przez %s tur.%s", baseShield, duration, heal)
}

type END_LVL_6 struct {
	EnduranceSkill
	NoEvents
	NoStats
	DefaultActiveTrigger
}

func (skill END_LVL_6) GetName() string {
	return "Poziom 6 - wytrzymałość"
}

func (skill END_LVL_6) GetDescription() string {
	return "Zmniejsza wszystkie obrażenia otrzymywane przez sojuszników i niego samego o 10%, przez jedną turę"
}

func (skill END_LVL_6) GetLevel() int {
	return 6
}

func (skill END_LVL_6) GetCD() int {
	return BaseCooldowns[skill.GetLevel()]
}

func (skill END_LVL_6) GetCooldown(upgrades int) int {
	return skill.GetCD()
}

func (skill END_LVL_6) GetCost() int {
	return 1
}

func (skill END_LVL_6) GetUpgradableCost(upgrades int) int {
	if HasUpgrade(upgrades, 1) {
		return 0
	}

	return 1
}

func (skill END_LVL_6) GetUpgrades() []types.PlayerSkillUpgrade {
	return []types.PlayerSkillUpgrade{
		{
			Id:          "Cost",
			Description: "Umiejętność nie kosztuje many",
		},
		{
			Id:          "Increase",
			Description: "Zwiększa redukcję obrażeń do 15%",
		},
	}
}

func (skill END_LVL_6) GetUpgradableDescription(upgrades int) string {
	baseReduce := 10

	if HasUpgrade(upgrades, 2) {
		baseReduce = 15
	}

	return fmt.Sprintf("Zmniejsza wszystkie obrażenia otrzymywane przez sojuszników i niego samego o %d przez jedną turę.", baseReduce)
}

func (skill END_LVL_6) UpgradableExecute(owner types.PlayerEntity, target types.Entity, fightInstance types.FightInstance, meta interface{}) interface{} {
	baseReduce := 10

	if HasUpgrade(owner.GetUpgrades(skill.GetLevel()), 2) {
		baseReduce = 15
	}

	targets := fightInstance.GetAlliesFor(owner.GetUUID())

	for _, target := range targets {
		fightInstance.HandleAction(types.Action{
			Event:  types.ACTION_EFFECT,
			Target: target.GetUUID(),
			Source: owner.GetUUID(),
			Meta: types.ActionEffect{
				Effect:   types.EFFECT_RESIST,
				Value:    baseReduce,
				Duration: 1,
				Caster:   owner.GetUUID(),
				Uuid:     uuid.New(),
				Meta: types.ActionEffectResist{
					Value:     baseReduce,
					IsPercent: true,
					DmgType:   4,
				},
				Target: target.GetUUID(),
				Source: types.SOURCE_ND,
			},
		})
	}

	return nil
}
