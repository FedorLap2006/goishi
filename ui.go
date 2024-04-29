package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/fogleman/gg"
)

const movePass = "pass"
const moveResign = "resign"

func moveCustomID(x int, y int, white bool) (r string) {
	r = "b"
	if white {
		r = "w"
	}

	r += strconv.Itoa(x)
	if y >= 0 {
		r += ":" + strconv.Itoa(y)
	}

	return
}

func parseMoveCustomID(customID string) (x, y int, white bool) {
	white = customID[0] == 'w'
	parts := strings.Split(customID[1:], ":")
	rx, _ := strconv.ParseInt(parts[0], 10, 8)
	x = int(rx)
	y = -1
	if len(parts) > 1 {
		ry, _ := strconv.ParseInt(parts[1], 10, 8)
		y = int(ry)
	}

	return
}

func boardToFilename(board *Board) (r string) {
	whiteToMove := 0
	if board.WhiteToMove {
		whiteToMove = 1
	}
	// TODO: ko & current move
	return fmt.Sprintf("%dx%d_%d_%dx%d_%dx%d_%s", board.sizeX, board.sizeY, whiteToMove, -1, -1, -1, -1, base64.StdEncoding.EncodeToString(board.Encode()))
}

func parseBoardFromFilename(filename string) (board *Board) {
	parts := strings.SplitN(filename, "_", 5)
	// fmt.Println(len(filename))
	size := strings.Split(parts[0], "x")
	sizeX, _ := strconv.ParseInt(size[0], 10, 8)
	sizeY, _ := strconv.ParseInt(size[1], 10, 8)

	whiteToMove, _ := strconv.ParseBool(parts[1])
	// TODO: ko & current move (2 & 3)
	data, _ := base64.StdEncoding.DecodeString(parts[4])
	board, _ = DecodeBoard(int(sizeX), int(sizeY), data)
	board.WhiteToMove = whiteToMove
	return
}

func makeMoveTable(board *Board, white bool, prefix, revertPrefix string, col int) (rows []discordgo.MessageComponent) {
	sx, sy := board.Size()
	dim := sx
	if col >= 0 {
		dim = sy
	}
	player := "b"
	if white {
		player = "w"
	}

	rowCount := int(math.Ceil(float64(dim) / 5.0))
	rows = make([]discordgo.MessageComponent, rowCount+1)
	for i := 0; i < rowCount+1; i++ {
		// c := make([]discordgo.MessageComponent, int(math.Ceil(float64(dim)/5.0)))
		rows[i] = &discordgo.ActionsRow{Components: make([]discordgo.MessageComponent, 0, 5)}
	}

	controlRow := rows[0].(*discordgo.ActionsRow)
	controlRow.Components = []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Pass",
			CustomID: prefix + "_" + movePass + "_" + player,
			Style:    discordgo.SuccessButton,
		},
		discordgo.Button{
			Label:    "Resign",
			Style:    discordgo.DangerButton,
			CustomID: prefix + "_" + moveResign + "_" + player,
		},
	}
	if col >= 0 {
		controlRow.Components = append(controlRow.Components, discordgo.Button{
			Label:    "Go back",
			Style:    discordgo.PrimaryButton,
			CustomID: revertPrefix + "_" + player,
		})
	}

	for i := 0; i < dim; i++ {
		arow := rows[1+i/5].(*discordgo.ActionsRow)
		label := string('A' + rune(i))
		disabled := false
		coords := moveCustomID(i, -1, white)
		if col >= 0 {
			label = string('A'+rune(col)) + strconv.Itoa(i+1)
			disabled = board.GetCell(col, i) != BoardCellEmpty
			coords = moveCustomID(col, i, white)
		}

		arow.Components = append(arow.Components, &discordgo.Button{
			CustomID: prefix + "_" + coords,
			Label:    label,
			Style:    discordgo.SecondaryButton,
			Disabled: disabled,
			// TODO: illegal moves
		})
	}

	return
}

func handleMove(board *Board, move string) (data *discordgo.InteractionResponseData, ok bool) {
	resigned := strings.HasPrefix(move, moveResign+"_")
	passed := strings.HasPrefix(move, movePass+"_")
	var content string
	var components []discordgo.MessageComponent

	if resigned {
		move = strings.TrimPrefix(move, moveResign+"_")

		winner := "Black"
		if move[0] == 'b' {
			winner = "White"
		}

		content = "Opponent has resigned. " + winner + " has won."
	} else {
		var white bool
		x, y := -1, -1
		if passed { // TODO: double pass = end
			move = strings.TrimPrefix(move, movePass+"_")
			white = move[0] == 'w'
		} else {
			x, y, white = parseMoveCustomID(move)
		}

		var playerToMove string
		if y == -1 && !passed {
			playerToMove = "Black"
			if white {
				playerToMove = "White"
			}

			components = makeMoveTable(board, white, "mc", "mp", x)
		} else {
			if !passed {
				// TODO: legality check
        ok = board.MakeMoveWithPlayer(x, y, white)
        if !ok {
          return nil, ok
        }
			}

			// Since the other player will make the next move we have inverted logic.
			playerToMove = "White"
			if white {
				playerToMove = "Black"
			}
			components = makeMoveTable(board, !white, "m", "mp", -1)
		}

		content = "**" + playerToMove + "**" + " " + "to move."
	}

	cnv := gg.NewContext(1000, 1000)
	board.Draw(cnv, 100.0, true)
	buffer := &bytes.Buffer{}
	png.Encode(buffer, cnv.Image())

	// if !resigned {
	//
	// }

	return &discordgo.InteractionResponseData{
		Content:     content,
    Attachments: &[]*discordgo.MessageAttachment{},
		Components:  components,
		Files: []*discordgo.File{
			{
				Name:        boardToFilename(board) + ".png",
				ContentType: "image/png",
				Reader:      buffer,
			},
		},
	}, true
}

func componentHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	data := i.MessageComponentData()
	customID := strings.SplitN(data.CustomID, "_", 2)
	switch customID[0] {
	case "m", "mc":
		// board := unpackBoardFromCustomIDV2(customID[2])
		filename := strings.TrimSuffix(i.Message.Attachments[0].Filename, ".png")
		board := parseBoardFromFilename(filename)
		response, ok := handleMove(board, customID[1])

    if !ok {
      s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
          Content: ":x: **Illegal move!**",
          Flags: discordgo.MessageFlagsEphemeral,
        },
      })
      return
    }

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: response,
		})
		if err != nil {
			log.Printf("could not respond: %s", err)
		}

	case "mp":
		customID = strings.SplitN(customID[1], "_", 2)
		whiteMove := customID[0] == "w"
		filename := strings.TrimSuffix(i.Message.Attachments[0].Filename, ".png")
		board := parseBoardFromFilename(filename)

		// board := unpackBoardFromCustomIDV2(customID[1])

		content := ""
		if whiteMove {
			content = "**White** to move."
		} else {
			content = "**Black** to move."
		}
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    content,
				Components: makeMoveTable(board, whiteMove, "mc", "mp", -1),
			},
		})
		if err != nil {
			log.Printf("could not respond: %s", err)
		}
	}

}
