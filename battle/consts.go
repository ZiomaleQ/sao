package battle

import (
	"sao/types"

	"github.com/google/uuid"
)

type FightMessage byte

const (
	MSG_ACTION_NEEDED FightMessage = iota
	MSG_FIGHT_START
	MSG_FIGHT_END
	MSG_ENTITY_RESCUE
	MSG_ENTITY_DIED
)

type LootType int

const (
	LOOT_ITEM LootType = iota
	LOOT_EXP
	LOOT_GOLD
)

type Loot struct {
	Type  LootType
	Count int
	Meta  *LootMeta
}

// Only for items
type LootMeta struct {
	Type types.ItemType
	Uuid uuid.UUID
}

type DamageType int

const (
	DMG_PHYSICAL DamageType = iota
	DMG_MAGICAL
	DMG_TRUE
)

type ActionEnum int

const (
	ACTION_ATTACK ActionEnum = iota
	ACTION_DEFEND
	ACTION_SKILL
	ACTION_ITEM
	ACTION_RUN
	//Helper events
	ACTION_COUNTER
	ACTION_EFFECT
	ACTION_DMG
	ACTION_SUMMON
)

type Effect int

const (
	EFFECT_POISON Effect = iota
	EFFECT_FEAR
	EFFECT_HEAL_SELF
	EFFECT_HEAL_OTHER
	EFFECT_HEAL_REDUCE
	EFFECT_MANA_RESTORE
	EFFECT_VAMP
	EFFECT_LIFESTEAL
	EFFECT_SHIELD
	EFFECT_BLIND
	EFFECT_DISARM
	EFFECT_GROUND
	EFFECT_ROOT
	EFFECT_SILENCE
	EFFECT_STUN
	EFFECT_IMMUNE
	EFFECT_STAT_INC
	EFFECT_STAT_DEC
	EFFECT_RESIST
	EFFECT_FASTEN
	EFFECT_TAUNT
	EFFECT_TAUNTED
	EFFECT_ON_HIT
)

type Action struct {
	Event  ActionEnum
	Target uuid.UUID
	Source uuid.UUID
	Meta   any
}

type ActionSummon struct {
	Flags       SummonFlags
	ExpireTimer int
	Entity      Entity
}

type SummonFlags int

const (
	SUMMON_FLAG_NONE SummonFlags = 1 << iota
	SUMMON_FLAG_ATTACK
	SUMMON_FLAG_EXPIRE
)

type ActionDamage struct {
	Damage   []Damage
	CanDodge bool
}

type ActionEffect struct {
	Effect   Effect
	Value    int
	Duration int
	Uuid     uuid.UUID
	Meta     any
	Caster   uuid.UUID
	Source   types.EffectSource
}

type ActionEffectHeal struct {
	Value int
}

type ActionEffectStat struct {
	Stat      types.Stat
	Value     int
	IsPercent bool
}

type ActionEffectResist struct {
	Value     int
	IsPercent bool
}

type ActionEffectOnHit struct {
	Skill     bool
	Attack    bool
	IsPercent bool
}

type ActionSkillMeta struct {
	Lvl        int
	IsForLevel bool
	SkillUuid  uuid.UUID
	Targets    []uuid.UUID
}

type ActionItemMeta struct {
	Item    uuid.UUID
	Targets []uuid.UUID
}

type Damage struct {
	Value int
	Type  DamageType
	//Its ignored when []Damage is of 1
	IsPercent bool
	CanDodge  bool
}

const SPEED_GAUGE = 100

type Entity interface {
	GetCurrentHP() int
	GetCurrentMana() int

	GetStat(types.Stat) int

	Action(*Fight) []Action
	TakeDMG(ActionDamage) []Damage
	DamageShields(int) int

	Heal(int)
	RestoreMana(int)
	Cleanse()

	GetLoot() []Loot
	CanDodge() bool

	GetFlags() types.EntityFlag

	GetName() string
	GetUUID() uuid.UUID

	ApplyEffect(ActionEffect)
	GetEffectByType(Effect) *ActionEffect
	GetEffectByUUID(uuid.UUID) *ActionEffect
	GetAllEffects() []ActionEffect
	RemoveEffect(uuid.UUID)
	TriggerAllEffects() []ActionEffect

	AppendTempSkill(types.WithExpire[types.PlayerSkill])
}

type DodgeEntity interface {
	Entity

	TakeDMGOrDodge(ActionDamage) ([]Damage, bool)
}

type PlayerEntity interface {
	DodgeEntity

	GetUID() string

	GetAllSkills() []types.PlayerSkill
	GetUpgrades(int) int
	GetLvlSkill(int) types.PlayerSkill
	GetSkill(uuid.UUID) types.PlayerSkill

	SetLvlCD(int, int)
	GetLvlCD(int) int

	SetCD(uuid.UUID, int)
	GetCD(uuid.UUID) int

	GetLevelSkillsCD() map[int]int
	GetSkillsCD() map[uuid.UUID]int

	SetDefendingState(bool)
	GetDefendingState() bool

	GetAllItems() []*types.PlayerItem
	AddItem(*types.PlayerItem)
	RemoveItem(int)

	GetParty() *uuid.UUID

	AppendDerivedStat(types.DerivedStat)
	SetLevelStat(types.Stat, int)
	GetDefaultStat(types.Stat) int
	ReduceCooldowns()
}
