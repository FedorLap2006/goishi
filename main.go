package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"image/png"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/fogleman/gg"
)

var (
	Token   = flag.String("token", "", "Bot token")
	AppID   = flag.String("appid", "", "Application ID")
	GuildID = flag.String("guildid", "", "Guild ID")
	Revert  = flag.Bool("rmcmd", false, "Revert commands")
)

func makeOptionMap(opts []*discordgo.ApplicationCommandInteractionDataOption) (m map[string]*discordgo.ApplicationCommandInteractionDataOption) {
	m = make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, v := range opts {
		m[v.Name] = v
	}
	return
}

func commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	acdata := i.ApplicationCommandData()
	switch acdata.Name {
	case "play-go":
		options := makeOptionMap(acdata.Options)
		boardSize := int(options["size"].IntValue())
		cnv := gg.NewContext(1000, 1000)
		var board *Board
		var err error
		if options["data"] != nil {
			var d []byte
			d, err = base64.StdEncoding.DecodeString(options["data"].StringValue())
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Error: could not decode data",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			board, err = DecodeBoard(boardSize, boardSize, d)
		} else {
			board, err = NewBoard(boardSize, boardSize)

      // if err == nil {
      //   board.SetCell(9, 10, BoardCellBlack)
      //   board.SetCell(10, 9, BoardCellBlack)
      //   board.SetCell(11, 10, BoardCellBlack)
      //   board.SetCell(10, 11, BoardCellBlack)
      // }
		}
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Error: " + err.Error(),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		board.Draw(cnv, 100.0, true)
		buffer := &bytes.Buffer{}
		png.Encode(buffer, cnv.Image())
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "**Black** to move.",
				Files: []*discordgo.File{
					{
						Name:        boardToFilename(board) + ".png",
						ContentType: "image/png",
						Reader:      buffer,
					},
				},
				Components: makeMoveTable(board, false, "mc", "mp", -1),
			},
		})
		if err != nil {
			panic(err)
		}
	}
}

var minBoardSize float64 = 9
var maxBoardSize float64 = 19

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "play-go",
		Description: "Play Go with friends!",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "size",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
				Description: "NxN board size",
				MinValue:    &minBoardSize,
				MaxValue:    maxBoardSize,
			},
			{
				Name:        "data",
				Required:    false,
				Type:        discordgo.ApplicationCommandOptionString,
				Description: "Board data",
			},
		},
	},
}

func main() {
	flag.Parse()
	session, _ := discordgo.New("Bot " + *Token)
	// session.Debug = true
	// session.LogLevel = discordgo.LogDebug
	session.AddHandler(commandHandler)
	session.AddHandler(componentHandler)
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Logged in as " + r.User.String())
	})

	savedCommands, err := session.ApplicationCommands(*AppID, *GuildID)
	if err != nil {
		log.Fatal("Could not fetch existing commands: " + err.Error())
	}

	log.Println("Creating commands...")

	commands, err = session.ApplicationCommandBulkOverwrite(*AppID, *GuildID, commands)
	if err != nil {
		log.Fatal("Could not overwrite commands: " + err.Error())
	}
	log.Println("registered commands: ", commands)

	err = session.Open()
	if err != nil {
		log.Fatal("could not open session: " + err.Error())
	}

	defer session.Close()

	defer func() {
		if !*Revert {
			return
		}
		log.Println("Reverting commands...")
		_, err = session.ApplicationCommandBulkOverwrite(*AppID, *GuildID, commands)
		if err != nil {
			log.Println("Could not revert commands: " + err.Error())
			j, _ := json.Marshal(savedCommands)
			log.Println("Saved commands: " + string(j))
			os.Exit(1)
		}
	}()

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch
}
