package main

import "github.com/bwmarrin/discordgo"

func init() {
	for k := range commands {
		commands[k].IntegrationTypes = &[]discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationGuildInstall,
			discordgo.ApplicationIntegrationUserInstall,
		}
		commands[k].Contexts = &[]discordgo.InteractionContextType{
			discordgo.InteractionContextBotDM,
			discordgo.InteractionContextGuild,
			discordgo.InteractionContextPrivateChannel,
		}
	}
}
