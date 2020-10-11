package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/dgcobra"
	"github.com/cfi2017/dgwidgets"
	"github.com/cfi2017/s157-bot/cmd"
	"github.com/cfi2017/s157-bot/internal"
	state2 "github.com/cfi2017/s157-bot/internal/state"
)

var (
	token   = flag.String("token", "", "bot token")
	fPrefix = flag.String("p", "!", "bot prefix")
	session *discordgo.Session
)

/*
verification flow:

*/
func main() {
	flag.Parse()

	var err error
	session, err = discordgo.New("Bot " + *token)
	if err != nil {
		panic(err)
	}

	// add handlers
	session.AddHandlerOnce(onReady)
	session.AddHandler(onMessageCreate) // announcements
	session.AddHandler(onGuildLeave)    // notify admins if admiral leaves

	handler := dgcobra.NewHandler(session)
	handler.RootFactory = cmd.RootCmdFactory
	handler.AddPrefix(*fPrefix)
	handler.AddPrefix("<@!667001702687440896> ")
	handler.Start()

	initRolePrompts()

	// open websocket
	err = session.Open()
	if err != nil {
		log.Fatal("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("STFC is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	logErr(session.Close)

}

func initRolePrompts() {
	state, err := state2.GetGuildState(session, "665976752992157706")
	if err != nil {
		panic(err)
	}
	for _, prompt := range state.RolePrompts {
		w := &dgwidgets.Widget{
			Keys: []string{
				prompt.Emote,
			},
			Handlers: map[string]dgwidgets.WidgetHandler{
				prompt.Emote: func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
					s := widget.Ses
					member, err := s.GuildMember(prompt.GuildID, reaction.UserID)
					if err != nil {
						log.Println(err)
						return
					}
					if internal.HasRole(member, prompt.RoleID) {
						err = s.GuildMemberRoleRemove(prompt.GuildID, reaction.UserID, prompt.RoleID)
					} else {
						err = s.GuildMemberRoleAdd(prompt.GuildID, reaction.UserID, prompt.RoleID)
					}
					if err != nil {
						log.Println(err)
					}
				},
			},
			DeleteReactions: true,
		}
		go func() {
			err := w.Hook(session, prompt.ChannelID, prompt.MessageID)
			if err != nil {
				sendMessage(prompt.ChannelID, fmt.Sprintf("Could not hook widget: %s.", err.Error()))
			}
		}()
	}
}

func onReady(_ *discordgo.Session, _ *discordgo.Ready) {

	// Set the playing status.
	logErr(func() error {
		return session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Game: &discordgo.Game{
				Name: "!alliance",
				Type: 2,
			},
		})
	})
}

func onMessageCreate(_ *discordgo.Session, event *discordgo.MessageCreate) {
	if event.WebhookID != "" {
		return
	}
	if event.Author.Bot {
		return
	}

	if !strings.HasPrefix(event.Content, "IGNORE") {
		var roleId string
		if event.ChannelID == internal.GameNewsQueueId {
			roleId = internal.GameNewsRoleId
		} else if event.ChannelID == internal.ServerNewsQueueId {
			roleId = internal.ServerNewsRoleId
		}
		if roleId != "" {
			roles, _ := session.GuildRoles(event.GuildID)
			var role *discordgo.Role
			for _, r := range roles {
				if r.ID == roleId {
					role = r
				}
			}
			if role == nil {
				return
			}

			role, err := session.GuildRoleEdit(event.GuildID, role.ID, role.Name, role.Color, role.Hoist, role.Permissions, true)
			if err != nil {
				sendMessage(event.ChannelID, err.Error())
				return
			}
			_, err = session.ChannelMessageSend(internal.NewsChannelId, fmt.Sprintf("%s\n\n%s", role.Mention(), event.Content))
			if err != nil {
				sendMessage(event.ChannelID, err.Error())
				return
			}
			_, err = session.GuildRoleEdit(event.GuildID, role.ID, role.Name, role.Color, role.Hoist, role.Permissions, false)
			if err != nil {
				sendMessage(event.ChannelID, err.Error())
				return
			}
		}
	}

}

func onGuildLeave(_ *discordgo.Session, event *discordgo.GuildMemberRemove) {
	if event.Roles != nil {
		if internal.HasRole(event.Member, internal.LeaderRoleId) {
			sendMessage(internal.AdminChannelId,
				fmt.Sprintf("[warning]: Representative %s has left the discord guild. Please assign a new representative manually.", event.Nick))
		}
	}
}

func sendMessage(channelId string, message string) {
	_, err := session.ChannelMessageSend(channelId, message)
	if err != nil {
		log.Println(err)
	}
	return
}

func logErr(some func() error) {
	if err := some(); err != nil {
		log.Println(err)
	}
}
