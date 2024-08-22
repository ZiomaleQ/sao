package discord

import (
	"bytes"
	"context"
	"fmt"
	"sao/config"
	"sao/data"
	"sao/player"
	"sao/player/inventory"
	"sao/types"
	"sao/utils"
	"sao/world"
	"sao/world/location"
	"sao/world/npc"
	"sao/world/party"
	"sao/world/tournament"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

var World *world.World
var Client *bot.Client
var Choices = make([]types.DiscordChoice, 0)

func StartClient() {
	client, err := disgo.New(config.Config.Token,
		bot.WithEventListenerFunc(func(e *events.Ready) {
			println("Discord connection is here!")
			e.Client().SetPresence(context.Background(), gateway.WithWatchingActivity("SAO"))
		}),
		bot.WithEventListenerFunc(func(e *events.MessageCreate) {
			if e.Message.Content == "sao:dump" && e.Message.Author.ID.String() == config.Config.Owner {
				data := World.CreateBackup()

				e.Client().Rest().AddReaction(e.Message.ChannelID, e.Message.ID, "02Calc:1159034005493129228")

				chanel, err := e.Client().Rest().CreateDMChannel(snowflake.MustParse(config.Config.Owner))

				if err != nil {
					return
				}

				_, err = e.Client().Rest().CreateMessage(chanel.ID(), discord.MessageCreate{
					Files: []*discord.File{
						{
							Name:   "backup.json",
							Reader: bytes.NewReader(data),
						},
					},
				})

				if err != nil {
					return
				}
			}
		}),
		bot.WithEventListenerFunc(commandListener),
		bot.WithEventListenerFunc(AutocompleteHandler),
		bot.WithEventListenerFunc(ComponentHandler),
		bot.WithEventListenerFunc(ModalSubmitHandler),
		bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentGuildMessages, gateway.IntentMessageContent)),
	)

	if err != nil {
		panic(err)
	}

	Client = &client

	if _, err = (*Client).Rest().SetGuildCommands((*Client).ApplicationID(), snowflake.MustParse(config.Config.GuildID), DISCORD_COMMANDS); err != nil {
		fmt.Println("error while registering commands")
	}

	if err = (*Client).OpenGateway(context.Background()); err != nil {
		panic(err)
	}

	go worldMessageListener()
}

func worldMessageListener() {
	for {
		msg, ok := <-World.DChannel

		if !ok {
			return
		}

		snowflake := snowflake.MustParse(msg.ChannelID)

		if msg.DM {
			ch, err := (*Client).Rest().CreateDMChannel(snowflake)

			if err != nil {
				return
			}

			snowflake = ch.ID()
		}

		_, err := (*Client).Rest().CreateMessage(snowflake, msg.MessageContent)

		if err != nil {
			panic(err)
		}
	}
}

func commandListener(event *events.ApplicationCommandInteractionCreate) {
	interactionData := event.SlashCommandInteractionData()

	user := event.User()
	member := event.Member()

	var playerChar *player.Player

	for _, pl := range World.Players {
		if pl.Meta.UserID == user.ID.String() {
			playerChar = pl
		}
	}

	if interactionData.CommandName() != "create" && interactionData.CommandName() != "turniej" && playerChar == nil {
		event.CreateMessage(noCharMessage)
		return
	}

	switch interactionData.CommandName() {
	case "create":
		if !isAdmin(member) {
			event.CreateMessage(MessageContent("Nie masz uprawnień do tej komendy", true))
			return
		}

		charName := interactionData.String("nazwa")
		charUser := interactionData.User("gracz")

		if World.GetPlayer(charUser.ID.String()) != nil {
			event.CreateMessage(MessageContent("Gracz ma już postać", true))
			return
		}

		newPlayer := player.NewPlayer(charName, charUser.ID.String())

		World.Players[newPlayer.GetUUID()] = &newPlayer

		err := event.CreateMessage(MessageContent("Zarejestrowano postać "+charName, false))

		if err != nil {
			fmt.Println("error while sending message")
		} else {
			gid := event.GuildID()

			(*Client).Rest().AddMemberRole(*gid, charUser.ID, snowflake.MustParse(config.Config.RoleID))
		}
		return
	case "ruch":
		locationName := interactionData.String("nazwa")

		err := World.MovePlayer(playerChar.GetUUID(), playerChar.Meta.Location.Floor, locationName, "")

		if err == nil {
			event.CreateMessage(MessageContent("Przeszedłeś do "+locationName, false))
		} else {
			event.CreateMessage(MessageContent("Nie możesz przejść do "+locationName, true))
		}

		return
	case "tp":
		floorName := interactionData.String("nazwa")

		currentLocation := World.Floors[playerChar.Meta.Location.Floor].FindLocation(playerChar.Meta.Location.Location)

		if !currentLocation.TP {
			event.CreateMessage(
				MessageContent("Nie możesz się stąd teleportować (idź do miasta lub lokacji z tp)", true),
			)
			return
		}

		err := World.MovePlayer(playerChar.GetUUID(), floorName, World.Floors[floorName].Default, "")

		if err == nil {
			event.CreateMessage(MessageContent("Teleportowałeś się na "+floorName, false))
		} else {
			event.CreateMessage(MessageContent("Nie możesz się teleportować"+err.Error(), true))
		}

		return
	case "info":
		if mentionedUser, exists := interactionData.OptUser("gracz"); exists {
			user = mentionedUser

			playerChar = nil
			playerChar = World.GetPlayer(user.ID.String())

			if playerChar == nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Użytkownik nie ma postaci").
						SetEphemeral(true).
						Build(),
				)

				return
			}
		}

		if playerChar == nil {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Nie znaleziono postaci").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		inFightText := utils.BoolToText(playerChar.Meta.FightInstance != nil, "Tak", "Nie")
		inPartyText := utils.BoolToText(playerChar.Meta.Party != nil, "Tak", "Nie")

		lvlText := fmt.Sprint(playerChar.XP.Level)

		if playerChar.XP.Level >= World.GetUnlockedFloorCount()*5 {
			lvlText += " MAX"
		} else {
			lvlText += fmt.Sprintf(" %d/%d", playerChar.XP.Exp, (playerChar.XP.Level*100)+100)
		}

		derivedStatsText := ""

		for _, stat := range playerChar.DynamicStats {
			derivedStatsText += fmt.Sprintf("- +%d%s %% jako %s\n", stat.Percent, types.StatToString[stat.Base], types.StatToString[stat.Derived])
		}

		if derivedStatsText == "" {
			derivedStatsText = "Brak"
		}

		rawLocation := World.Floors[playerChar.Meta.Location.Floor].FindLocation(playerChar.Meta.Location.Location)

		messageBuilder := discord.NewMessageCreateBuilder().
			AddEmbeds(
				discord.NewEmbedBuilder().
					AddField("Nazwa", playerChar.GetName(), true).
					AddField("Gracz", fmt.Sprintf("<@%s>", playerChar.Meta.UserID), true).
					AddField("Lokacja", fmt.Sprintf("<#%s>", rawLocation.CID), true).
					AddField("HP", fmt.Sprintf("%d/%d", playerChar.Stats.HP, playerChar.GetStat(types.STAT_HP)), true).
					AddField("Mana", fmt.Sprintf("%d/%d", playerChar.GetCurrentMana(), playerChar.GetStat(types.STAT_MANA)), true).
					AddField("Atak", fmt.Sprintf("%d", playerChar.GetStat(types.STAT_AD)), true).
					AddField("AP", fmt.Sprintf("%d", playerChar.GetStat(types.STAT_AP)), true).
					AddField("DEF/RES", fmt.Sprintf("%d/%d", playerChar.GetStat(types.STAT_DEF), playerChar.GetStat(types.STAT_MR)), true).
					AddField("Lvl", lvlText, true).
					AddField("SPD/AGL", fmt.Sprintf("%d/%d", playerChar.GetStat(types.STAT_SPD), playerChar.GetStat(types.STAT_AGL)), true).
					AddField("W walce?", inFightText, true).
					AddField("W party?", inPartyText, true).
					AddField("Dynamiczne statystyki", derivedStatsText, true).
					Build(),
			)

		if playerChar.Stats.HP > 0 && playerChar.Stats.HP < playerChar.GetStat(types.STAT_HP) {
			messageBuilder.AddActionRow(discord.NewPrimaryButton("Czekaj do 100% HP", "utils|wait"))
		}

		event.CreateMessage(messageBuilder.Build())
	case "skill":
		switch *interactionData.SubCommandName {
		case "pokaż":
			if mentionedUser, exists := interactionData.OptUser("gracz"); exists {
				user = mentionedUser

				playerChar = nil
				playerChar = World.GetPlayer(user.ID.String())

				if playerChar == nil {
					event.CreateMessage(
						discord.
							NewMessageCreateBuilder().
							SetContent("Użytkownik nie ma postaci").
							SetEphemeral(true).
							Build(),
					)
					return
				}
			}

			embed := discord.NewEmbedBuilder()

			if len(playerChar.Inventory.LevelSkills) == 0 {
				embed.AddField("Umiejętności za lvl", "Brak", false)
			}

			for _, skill := range playerChar.Inventory.LevelSkills {
				embed.AddField(
					fmt.Sprintf("%s (LVL: %d)", skill.GetName(), skill.GetLevel()),
					fmt.Sprintf("Ścieżka: %s\n\n%s", types.PathToString[skill.GetPath()], skill.GetDescription()),
					false,
				)
			}

			event.CreateMessage(MessageEmbed(embed.Build()))

			return
		case "odblokuj":
			lvl := interactionData.Int("lvl")

			if _, exists := playerChar.Inventory.LevelSkills[lvl]; exists {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Odblokowano już umiejętność na tym poziomie").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			embed := discord.NewEmbedBuilder()

			buttons := make([]discord.InteractiveComponent, 0)

			for _, skillTree := range inventory.AVAILABLE_SKILLS {
				skills, exists := skillTree[lvl]

				if !exists {
					continue
				}

				for idx, skill := range skills {
					upgradeMsg := ""

					for _, upgrade := range skill.GetUpgrades() {
						upgradeMsg += fmt.Sprintf("\n- %s", upgrade.Description)
					}

					embed.AddField(
						skill.GetName(),
						fmt.Sprintf("%s\nUlepszenia:%s", skill.GetDescription(), upgradeMsg),
						false,
					)

					buttons = append(buttons, discord.NewPrimaryButton(
						skill.GetName(),
						fmt.Sprintf("su|%d|%d|%d", skill.GetPath(), skill.GetLevel(), idx),
					))
				}
			}

			if len(buttons) == 0 {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie znaleziono umiejętności dla tego poziomu").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			containers := make([]discord.ContainerComponent, 0)
			containers = append(containers, discord.NewActionRow())

			tempButtons := make([]discord.InteractiveComponent, len(buttons))
			copy(tempButtons, buttons)

			for len(tempButtons) > 0 {
				if len(containers[len(containers)-1].(discord.ActionRowComponent).Components()) == 5 {
					containers = append(containers, discord.NewActionRow())
				}

				containers[len(containers)-1] = containers[len(containers)-1].(discord.ActionRowComponent).AddComponents(tempButtons[0])
				tempButtons = tempButtons[1:]
			}

			if len(containers[len(containers)-1].Components()) == 0 {
				containers = containers[:len(containers)-1]
			}

			if len(containers) > 5 {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Za dużo umiejętności na tym poziomie.").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					AddEmbeds(embed.Build()).
					AddContainerComponents(containers...).
					Build(),
			)

			return
		}
	case "plecak":
		switch *interactionData.SubCommandName {
		case "pokaż":
			embed := discord.NewEmbedBuilder()

			if len(playerChar.Inventory.Items) == 0 {
				embed.AddField("Przedmioty", fmt.Sprintf("%d/10", 0), false)
			} else {
				count := 0

				for _, item := range playerChar.Inventory.Items {
					if !item.Hidden {
						count++
					}
				}

				embed.AddField("Przedmioty", fmt.Sprintf("%d/10", count), false)
			}

			for _, item := range playerChar.Inventory.Items {

				if item.Hidden {
					continue
				}

				embed.AddField(item.Name, item.Description, false)
			}

			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					AddEmbeds(embed.Build()).
					Build(),
			)

			return
		}
	case "szukaj":
		dChannel, error := (*Client).Rest().GetChannel(event.Channel().ID())

		if error != nil {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Nie można znaleźć kanału").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		textChannel := dChannel.(discord.GuildTextChannel)

		textChannel.ParentID()

		var dFloor *location.Floor

		for _, floor := range World.Floors {
			if floor.CID == textChannel.ParentID().String() {
				dFloor = &floor
				break
			}
		}

		if dFloor == nil {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Ta kategoria nie wygląda jak z SAO...").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		newLocation := dFloor.FindLocation(event.Channel().ID().String())

		if newLocation == nil {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Nie mam tej lokacji w bazie...").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		if playerChar.Meta.Location.Location == newLocation.Name && playerChar.Meta.Location.Floor == dFloor.Name {

			loc := World.Floors[playerChar.Meta.Location.Floor].FindLocation(playerChar.Meta.Location.Location)

			if loc.CityPart {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie ma czego tu szukać...").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			go World.PlayerSearch(playerChar.GetUUID())

			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Szukanie...").
					SetEphemeral(true).
					Build(),
			)
		} else {
			pFloor := playerChar.Meta.Location.Floor

			if playerChar.Meta.Location.Floor != dFloor.Name {
				pFloor = dFloor.Name
			}

			err := World.MovePlayer(playerChar.GetUUID(), pFloor, newLocation.Name, "")

			if err != nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie możesz tam iść").
						SetEphemeral(true).
						Build(),
				)

				return
			}

			loc := World.Floors[playerChar.Meta.Location.Floor].FindLocation(playerChar.Meta.Location.Location)

			if loc.CityPart {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie ma czego tu szukać...").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			go World.PlayerSearch(playerChar.GetUUID())

			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Szukanie (automatycznie przeniosłam cię do lokacji)...").
					SetEphemeral(true).
					Build(),
			)
		}

		return
	case "party":
		if playerChar.Meta.Party == nil && *interactionData.SubCommandName != "stwórz" {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Nie jesteś w party").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		switch *interactionData.SubCommandName {
		case "pokaż":
			embed := discord.NewEmbedBuilder()

			partyMembersText := ""

			partyObj := World.Parties[*playerChar.Meta.Party]

			for _, member := range partyObj.Players {
				memberObj := World.Players[member.PlayerUuid]

				partyMembersText += fmt.Sprintf("<@%s> - %s\n", memberObj.Meta.UserID, memberObj.GetName())
			}

			if partyMembersText[len(partyMembersText)-1] == '\n' {
				partyMembersText = partyMembersText[:len(partyMembersText)-1]
			}

			embed.AddField("Członkowie", partyMembersText, false)

			partyLeader := World.Players[partyObj.Leader]

			embed.AddField("Lider", fmt.Sprintf("<@%s> - %s\n", partyLeader.Meta.UserID, partyLeader.GetName()), false)

			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					AddEmbeds(embed.Build()).
					Build(),
			)

			return
		case "zapros":
			if len(World.Parties[*playerChar.Meta.Party].Players) >= 6 {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Party jest pełne").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			part := World.Parties[*playerChar.Meta.Party]

			if playerChar.GetUUID() != part.Leader {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie jesteś liderem").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			mentionedUser := interactionData.User("gracz")

			for _, pl := range World.Players {
				if pl.Meta.UserID == mentionedUser.ID.String() {
					if pl.Meta.Party != nil {
						event.CreateMessage(
							discord.
								NewMessageCreateBuilder().
								SetContent("Gracz jest już w party").
								SetEphemeral(true).
								Build(),
						)
						return
					}
				}
			}

			ch, error := event.Client().Rest().CreateDMChannel(mentionedUser.ID)

			if error != nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie można wysłać wiadomości do gracza").
						SetEphemeral(true).
						Build(),
				)
			}

			chID := ch.ID()

			_, error = event.Client().Rest().CreateMessage(chID, discord.NewMessageCreateBuilder().
				SetContent(fmt.Sprintf("<@%s> (%s) zaprasza cię do party", user.ID.String(), playerChar.GetName())).
				AddActionRow(
					discord.NewPrimaryButton("Akceptuj", "party/res|"+(*playerChar.Meta.Party).String()),
					discord.NewDangerButton("Odrzuć", "party/rej|"+(*playerChar.Meta.Party).String()),
				).
				Build(),
			)

			if error != nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie można wysłać wiadomości do gracza").
						SetEphemeral(true).
						Build(),
				)
			} else {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Wysłano zaproszenie do party").
						SetEphemeral(true).
						Build(),
				)
			}

			return
		case "wyrzuć":
			part := World.Parties[*playerChar.Meta.Party]

			if playerChar.GetUUID() != part.Leader {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie jesteś liderem").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			mentionedUser := interactionData.User("gracz")

			if mentionedUser.ID == user.ID {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie możesz wyrzucić samego siebie").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			for _, pl := range World.Players {
				if pl.Meta.UserID == mentionedUser.ID.String() {
					if pl.Meta.Party == nil {
						event.CreateMessage(
							discord.
								NewMessageCreateBuilder().
								SetContent("Gracz nie jest w party").
								SetEphemeral(true).
								Build(),
						)
						return
					}

					if *pl.Meta.Party != *playerChar.Meta.Party {
						event.CreateMessage(
							discord.
								NewMessageCreateBuilder().
								SetContent("Gracz nie jest w twoim party").
								SetEphemeral(true).
								Build(),
						)
						return
					}

					for i, partyMember := range World.Parties[*playerChar.Meta.Party].Players {
						if partyMember.PlayerUuid == pl.GetUUID() {
							pl.Meta.Party = nil

							World.Parties[*playerChar.Meta.Party].Players = append(World.Parties[*playerChar.Meta.Party].Players[:i], World.Parties[*playerChar.Meta.Party].Players[i+1:]...)
							break
						}
					}

				}
			}

			return
		case "opuść":
			part := World.Parties[*playerChar.Meta.Party]

			if playerChar.GetUUID() == part.Leader {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Jesteś liderem...").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			for i, partyMember := range World.Parties[*playerChar.Meta.Party].Players {
				if partyMember.PlayerUuid == playerChar.GetUUID() {
					World.Parties[*playerChar.Meta.Party].Players = append(World.Parties[*playerChar.Meta.Party].Players[:i], World.Parties[*playerChar.Meta.Party].Players[i+1:]...)
					break
				}
			}

			playerChar.Meta.Party = nil

			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Opuściłeś party").
					SetEphemeral(true).
					Build(),
			)

			return
		case "zmień":
			part := World.Parties[*playerChar.Meta.Party]

			if playerChar.GetUUID() != part.Leader {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie jesteś liderem").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			mentionedUser := interactionData.User("gracz")

			if mentionedUser.ID == user.ID {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie możesz zmienić roli samego siebie").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			for _, pl := range World.Players {
				if pl.Meta.UserID == mentionedUser.ID.String() {
					if pl.Meta.Party == nil {
						event.CreateMessage(
							discord.
								NewMessageCreateBuilder().
								SetContent("Gracz nie jest w party").
								SetEphemeral(true).
								Build(),
						)
						return
					}

					if *pl.Meta.Party != *playerChar.Meta.Party {
						event.CreateMessage(
							discord.
								NewMessageCreateBuilder().
								SetContent("Gracz nie jest w twoim party").
								SetEphemeral(true).
								Build(),
						)
						return
					}

					role := interactionData.String("rola")

					for i, partyMember := range World.Parties[*playerChar.Meta.Party].Players {
						if partyMember.PlayerUuid == pl.GetUUID() {
							switch role {
							case "Lider":
								World.Parties[*playerChar.Meta.Party].Leader = pl.GetUUID()
							case "DPS":
								World.Parties[*playerChar.Meta.Party].Players[i].Role = party.DPS
							case "Support":
								World.Parties[*playerChar.Meta.Party].Players[i].Role = party.Support
							case "Tank":
								World.Parties[*playerChar.Meta.Party].Players[i].Role = party.Tank
							}
							break
						}
					}

					event.CreateMessage(
						discord.
							NewMessageCreateBuilder().
							SetContent("Zmieniono rolę").
							SetEphemeral(true).
							Build(),
					)
				}
			}
		case "rozwiąż":
			part := World.Parties[*playerChar.Meta.Party]

			if playerChar.GetUUID() != part.Leader {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie jesteś liderem").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			uuid := *playerChar.Meta.Party

			for _, partyMember := range World.Parties[uuid].Players {
				World.Players[partyMember.PlayerUuid].Meta.Party = nil
			}

			delete(World.Parties, uuid)

			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Rozwiązano party").
					SetEphemeral(true).
					Build(),
			)

			return
		}
	case "ratuj":
		if playerChar.Meta.FightInstance != nil {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Już jesteś w walce!").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		options := make([]discord.StringSelectMenuOption, 0)

		cid := event.Channel().ID().String()

		for _, fight := range World.Fights {

			if fight.Location.CID != cid {
				continue
			} else {
				for entityUuid, entityEntry := range fight.Entities {
					if entityEntry.Entity.GetFlags()&types.ENTITY_AUTO != 0 {
						options = append(options, discord.NewStringSelectMenuOption(entityEntry.Entity.GetName(), entityUuid.String()))
					}
				}
			}

		}

		if len(options) == 0 {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Nie ma nikogo do ratowania").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		event.CreateMessage(
			discord.
				NewMessageCreateBuilder().
				SetContent("Wybierz kogo chcesz ratować").
				AddActionRow(
					discord.NewStringSelectMenu("f/save", "Wybierz", options...),
				).
				Build(),
		)
		return
	case "stwórz":
		itemUuid, valid := uuid.Parse(interactionData.String("nazwa"))

		if valid != nil {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Nie znaleziono receptury? (XD)").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		recipe, exists := data.Recipes[itemUuid]

		if !exists {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Nie znaleziono receptury? (XD)").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		err := playerChar.Inventory.Craft(recipe)

		if err.Error() == "MISSING_INGREDIENT" {
			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Brakuje składników").
					SetEphemeral(true).
					Build(),
			)
			return
		}

		event.CreateMessage(
			discord.
				NewMessageCreateBuilder().
				SetContentf("Stworzono przedmiot\n%s x%d", recipe.Name, recipe.Product.Count).
				SetEphemeral(true).
				Build(),
		)
	case "furia":
		switch *interactionData.SubCommandName {
		case "pokaż":
			if playerChar.Meta.Fury == nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie masz furii").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			fury := playerChar.Meta.Fury

			message := discord.NewMessageCreateBuilder()

			embed := discord.NewEmbedBuilder()

			embed.AddField("Nazwa", fury.Name, true)

			levelTxt := fmt.Sprint(fury.XP.LVL)

			if fury.XP.LVL == 10 {
				levelTxt += " - czas na kolejny tier!"
			} else {
				levelTxt += fmt.Sprintf(" %d/%d", fury.XP.XP, fury.NextLvlXPGauge())
			}

			embed.AddField("Poziom", levelTxt, true)
			embed.AddField("Umiejętności", "Brak", false)

			statsText := ""

			furyStats := fury.GetStats()

			if len(furyStats) == 0 {
				statsText = "Brak"
			} else {
				for stat, value := range furyStats {
					statsText += fmt.Sprintf("- %d %s\n", value, types.StatToString[stat])
				}
			}

			embed.AddField("Statystyki", statsText, false)

			embed.AddField("Aktualny tier", fmt.Sprint(fury.CurrentTier), true)

			if fury.CurrentTier == len(fury.Tiers) {
				embed.AddField("Kolejny tier?", "Brak", true)
			} else {
				nextTier := fury.Tiers[fury.CurrentTier]

				nextTierText := "Tier: " + fmt.Sprint(fury.CurrentTier+1) + "\nPotrzebne składniki: "

				if len(nextTier.Ingredients) == 0 {
					nextTierText += "Brak"
				} else {
					for _, ingredient := range nextTier.Ingredients {
						nextTierText += fmt.Sprintf("\n- %s x%d", ingredient.Name, ingredient.Count)
					}
				}

				nextTierText += "\n"

				if len(nextTier.Skills) == 0 {
					nextTierText += "Brak nowych umiejętności\n"
				} else {
					nextTierText += "Nowe umiejętności: \n"

					for _, skill := range nextTier.Skills {
						nextTierText += "-" + skill.GetName() + "\n"
					}
				}

				if len(nextTier.Stats) == 0 {
					nextTierText += "Brak nowych statystyk\n"
				} else {
					nextTierText += "Nowe statystyki: \n"

					for stat, value := range nextTier.Stats {
						nextTierText += fmt.Sprintf("- %d %s\n", value, types.StatToString[stat])
					}
				}

				embed.AddField("Kolejny tier?", nextTierText, false)

				message.AddEmbeds(embed.Build())
			}

			event.CreateMessage(message.Build())
		case "ulepsz":
			if playerChar.Meta.Fury == nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie masz furii").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			if playerChar.Meta.Fury != nil {
				playerFury := playerChar.Meta.Fury

				if playerFury.CurrentTier == len(playerFury.Tiers) {
					event.CreateMessage(
						discord.
							NewMessageCreateBuilder().
							SetContent("Osiągnięto maksymalny tier").
							SetEphemeral(true).
							Build(),
					)
					return
				}

				if playerFury.XP.LVL < 10 {
					event.CreateMessage(
						discord.
							NewMessageCreateBuilder().
							SetContent("Nie osiągnięto maksymalnego poziomu na danym tierze").
							SetEphemeral(true).
							Build(),
					)
					return
				}

				nextTier := playerFury.Tiers[playerFury.CurrentTier]

				if len(nextTier.Ingredients) == 0 {
					playerFury.CurrentTier++
					playerFury.XP.LVL = 1
					playerFury.XP.XP = 0

					event.CreateMessage(
						discord.
							NewMessageCreateBuilder().
							SetContent("Ulepszono furie na kolejny tier!").
							SetEphemeral(true).
							Build(),
					)
				} else {
					if playerChar.Inventory.HasIngredients(nextTier.Ingredients) {
						playerChar.Inventory.RemoveIngredients(nextTier.Ingredients)

						playerFury.CurrentTier++
						playerFury.XP.LVL = 1
						playerFury.XP.XP = 0

						event.CreateMessage(
							discord.
								NewMessageCreateBuilder().
								SetContent("Ulepszono furie na kolejny tier!").
								SetEphemeral(true).
								Build(),
						)
					} else {
						event.CreateMessage(
							discord.
								NewMessageCreateBuilder().
								SetContent("Brakuje składników").
								SetEphemeral(true).
								Build(),
						)
					}
				}
			}
		}
	case "sklep":
		switch *interactionData.SubCommandName {
		case "pokaż":
			playerLocation := playerChar.Meta.Location

			storesInLocation := make([]*npc.NPCStore, 0)

			for _, npcObject := range World.NPCs {
				if playerLocation == npcObject.Location {
					storesInLocation = append(storesInLocation, npcObject.Store)
				}
			}

			if len(storesInLocation) == 0 {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie ma sklepów w tej lokalizacji").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			if len(storesInLocation) > 25 {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("<@344048874656366592> incydent sklepowy").
						Build(),
				)
				return
			}

			messageBuilder := discord.NewMessageCreateBuilder()

			for i := 0; i < (len(storesInLocation)/5)+1; i++ {
				var stores []*npc.NPCStore

				if len(storesInLocation) < i*5+5 {
					stores = storesInLocation[i*5:]
				} else {
					stores = storesInLocation[i*5 : i*5+5]
				}

				buttonArray := make([]discord.InteractiveComponent, 0)

				for _, store := range stores {
					buttonArray = append(buttonArray, discord.NewPrimaryButton(store.Name, "shop/show/1/"+store.Uuid.String()))
				}

				messageBuilder.AddActionRow(buttonArray...)
			}

			event.CreateMessage(messageBuilder.Build())
		}
	case "turniej":
		switch *interactionData.SubCommandName {
		case "stwórz":
			if !isAdmin(member) {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie masz uprawnień do tej komendy").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			tournamentName := interactionData.String("nazwa")
			maxCount, isMaxCountPresent := interactionData.OptInt("max")
			tournamentType := tournament.TournamentType(interactionData.Int("typ"))

			if !isMaxCountPresent {
				maxCount = -1
			}

			tournament := tournament.Tournament{
				Uuid:         uuid.New(),
				Name:         tournamentName,
				Type:         tournamentType,
				MaxPlayers:   maxCount,
				Participants: make([]uuid.UUID, 0),
				State:        tournament.Waiting,
			}

			World.RegisterTournament(tournament)

			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Turniej stworzony").
					SetEphemeral(true).
					Build(),
			)
		case "rozpocznij":
			if !isAdmin(member) {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie masz uprawnień do tej komendy").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			tournamentName := interactionData.String("nazwa")

			var actualTournament *tournament.Tournament

			for _, t := range World.Tournaments {
				if t.Name == tournamentName {
					actualTournament = t
					break
				}
			}

			if actualTournament == nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie znaleziono turnieju").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			World.StartTournament(actualTournament.Uuid)

			event.CreateMessage(
				discord.
					NewMessageCreateBuilder().
					SetContent("Rozpoczynam turniej!").
					SetEphemeral(true).
					Build(),
			)
			return
		}
	case "handel":
		switch *interactionData.SubCommandName {
		case "nowy":
			if playerChar.Meta.Transaction != nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Już masz otwartą ofertę!").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			secondUser := interactionData.User("gracz")

			secondPlayer := World.GetPlayer(secondUser.ID.String())

			if secondPlayer == nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Gracz nie ma postaci").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			if secondPlayer.Meta.Transaction != nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Gracz ma otwartą ofertę").
						SetEphemeral(true).
						Build(),
				)
				return
			}

			ch, error := event.Client().Rest().CreateDMChannel(secondUser.ID)

			if error != nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie można wysłać wiadomości do gracza").
						SetEphemeral(true).
						Build(),
				)
			}

			chID := ch.ID()

			tempTrans := World.CreatePendingTransaction(playerChar.GetUUID(), secondPlayer.GetUUID())

			_, error = event.Client().Rest().CreateMessage(chID, discord.NewMessageCreateBuilder().
				SetContent(fmt.Sprintf("<@%s> (%s) zaprasza cię handlu!", user.ID.String(), playerChar.GetName())).
				AddActionRow(
					discord.NewPrimaryButton("Akceptuj", "trade/res|"+tempTrans.Uuid.String()),
					discord.NewDangerButton("Odrzuć", "trade/rej|"+tempTrans.Uuid.String()),
				).
				Build(),
			)

			if error != nil {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Nie można wysłać wiadomości do gracza").
						SetEphemeral(true).
						Build(),
				)
			} else {
				event.CreateMessage(
					discord.
						NewMessageCreateBuilder().
						SetContent("Wysłano zaproszenie do handlu").
						SetEphemeral(true).
						Build(),
				)
			}
		}
	}
}
