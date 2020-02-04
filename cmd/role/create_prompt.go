package role

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/dgwidgets"
	"github.com/cfi2017/s157-bot/internal"
	"github.com/cfi2017/s157-bot/internal/state"
	"github.com/spf13/cobra"
)

func createPromptCmdFactory(s *discordgo.Session, e *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new role prompt.",
		Long: `
Sets your nickname.
`,
		Args: cobra.MinimumNArgs(1),
	}
	channel := cmd.Flags().StringP("channel", "c", e.ChannelID, "channel id to create prompt in")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if e.ChannelID != *channel {
			if ok, _ := internal.IsChannelInGuild(s, e.GuildID, *channel); !ok {
				return errors.New("invalid channel id")
			}
		}

		return nil
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		gs, err := state.GetGuildState(s, e.GuildID)
		if err != nil {
			_, _ = s.ChannelMessageSend(e.ChannelID, err.Error())
			return
		}

		roles, err := s.GuildRoles(e.GuildID)
		if err != nil {
			_, _ = s.ChannelMessageSend(e.ChannelID, err.Error())
			return
		}

		roleID := args[0]
		message := strings.Join(args[1:], " ")
		if message == "" {
			message = "Click the below emote to give yourself this role."
		}

		var role discordgo.Role
		for _, r := range roles {
			if r.ID == roleID {
				role = *r
			}
		}

		rp := state.RolePrompt{
			ChannelID: *channel,
			MessageID: "",
			RoleID:    roleID,
			GuildID:   e.GuildID,
			Emote:     "✅",
		}
		rand.Seed(time.Now().UnixNano())

		bot, _ := s.GuildMember(e.GuildID, s.State.User.ID)
		name := bot.User.Username
		if bot.Nick != "" {
			name = bot.Nick
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Toggle %s", role.Name),
			Description: message,
			Color:       int((((rand.Int31n(255) << 8) + rand.Int31n(255)) << 8) + (rand.Int31n(255))),
			Timestamp:   time.Now().Format(time.RFC3339),
			Author:      &discordgo.MessageEmbedAuthor{Name: name},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "s157-bot",
			},
		}
		wg := dgwidgets.NewWidget(s, *channel, embed)
		wg.DeleteReactions = true
		_ = wg.Handle("✅", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
			s := widget.Ses
			member, err := s.GuildMember(rp.GuildID, reaction.UserID)
			if err != nil {
				log.Println(err)
				return
			}
			if internal.HasRole(member, rp.RoleID) {
				err = s.GuildMemberRoleRemove(rp.GuildID, reaction.UserID, rp.RoleID)
			} else {
				err = s.GuildMemberRoleAdd(rp.GuildID, reaction.UserID, rp.RoleID)
			}
			if err != nil {
				log.Println(err)
			}
		})
		go wg.Spawn()
		for wg.Message == nil {
			time.Sleep(200 * time.Millisecond)
		} // wait for message to be sent
		rp.MessageID = wg.Message.ID
		gs.RolePrompts = append(gs.RolePrompts, rp)
		err = state.SetGuildState(s, e.GuildID, gs)
		if err != nil {
			_, _ = s.ChannelMessageSend(e.ChannelID, err.Error())
		}
	}

	return cmd
}
