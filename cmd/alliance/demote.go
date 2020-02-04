package alliance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/dgwidgets"
	"github.com/cfi2017/s157-bot/internal"
	"github.com/spf13/cobra"
)

func demoteCmdFactory(s *discordgo.Session, e *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demote",
		Short: "Demote a user in your alliance.",
		Long: `
Demote a user to the previous level. Available levels are:
	- Member
	- User is kicked out
`,
		Run: func(cmd *cobra.Command, args []string) {
			e.Member.GuildID = e.GuildID
			isLeader := internal.HasRole(e.Member, internal.LeaderRoleId)
			if !isLeader {
				_, _ = s.ChannelMessageSend(e.ChannelID, "You are not the representative of your alliance.")
				return
			}
			if len(e.Mentions) != 1 {
				_, _ = s.ChannelMessageSend(e.ChannelID, "Please mention exactly one user.")
				return
			}
			target := e.Mentions[0]
			if target.ID == e.Author.ID {
				_, _ = s.ChannelMessageSend(e.ChannelID, "You cannot demote yourself.")
				return
			}
			targetMember, _ := s.GuildMember(e.GuildID, target.ID)
			if !internal.HasRole(targetMember, internal.MemberRoleId) {
				_, _ = s.ChannelMessageSend(e.ChannelID, "Your target is not in an alliance.")
				return
			}

			if !strings.HasPrefix(targetMember.Nick, strings.Split(e.Member.Nick, " ")[0]) {
				_, _ = s.ChannelMessageSend(e.ChannelID, "Your target is not in the same alliance as you.")
				return
			}

			if internal.HasRole(targetMember, internal.CommodoreRoleId) {
				err := s.GuildMemberRoleRemove(e.GuildID, target.ID, internal.CommodoreRoleId)
				if err != nil {
					_, _ = s.ChannelMessageSend(e.ChannelID, "Could not demote user. No permission to remove role.")
				} else {
					_, _ = s.ChannelMessageSend(e.ChannelID, "User Demoted.")
				}
			} else {

				embed := &discordgo.MessageEmbed{
					Title:       "Authorising...",
					Description: fmt.Sprintf("Are you sure you want to kick %s?", targetMember.Nick),
				}

				w := dgwidgets.NewWidget(s, e.ChannelID, embed)
				w.DeleteReactions = true
				w.Timeout = 1 * time.Minute
				w.DeleteOnTimeout = true
				w.UserWhitelist = []string{e.Author.ID}
				_ = w.Handle("✅", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
					_ = s.GuildMemberNickname(e.GuildID, e.Mentions[0].ID, strings.Split(e.Mentions[0].Username, " ")[1])
					_ = s.GuildMemberRoleRemove(e.GuildID, e.Mentions[0].ID, internal.MemberRoleId)

					_, _ = s.ChannelMessageSend(e.ChannelID, fmt.Sprintf("Kicked %s.", targetMember.Nick))
				})

				_ = w.Handle("❌", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
					_ = s.ChannelMessageDelete(reaction.ChannelID, reaction.MessageID)

					_, _ = s.ChannelMessageSend(e.ChannelID, "Cancelled that.")
				})

				err := w.Spawn()
				if err != nil && err == context.DeadlineExceeded {
					_, _ = s.ChannelMessageSend(e.ChannelID, "Timed out.")
				}
			}
		},
	}
	return cmd
}
