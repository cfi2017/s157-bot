package alliance

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/dgwidgets"
	"github.com/cfi2017/s157-bot/internal"
	"github.com/spf13/cobra"
)

func promoteCmdFactory(s *discordgo.Session, e *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Promote a user in your alliance.",
		Long: `Promotes a user to the next level. Available levels are:
	- Commodore
    - Admiral/Representative
`,
		Run: func(cmd *cobra.Command, args []string) {
			if internal.IsDM(s, e.ChannelID) {
				_, _ = s.ChannelMessageSend(e.ChannelID, "Use this in a guild.")
				_ = cmd.Usage()
				return
			}
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
				_, _ = s.ChannelMessageSend(e.ChannelID, "You cannot promote yourself.")
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

				embed := &discordgo.MessageEmbed{
					Title:       "Authorising...",
					Description: fmt.Sprintf("Are you sure you want to make %s your successor?", targetMember.Nick),
				}

				w := dgwidgets.NewWidget(s, e.ChannelID, embed)
				w.DeleteReactions = true
				w.Timeout = time.Minute
				w.UserWhitelist = []string{e.Author.ID}
				_ = w.Handle("✅", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
					// adding new user to alliance
					_ = s.GuildMemberRoleAdd(e.GuildID, e.Mentions[0].ID, internal.LeaderRoleId)
					_ = s.GuildMemberRoleRemove(e.GuildID, e.Mentions[0].ID, internal.CommodoreRoleId)
					_ = s.GuildMemberRoleRemove(e.GuildID, e.Author.ID, internal.LeaderRoleId)
					_, _ = s.ChannelMessageSend(e.ChannelID, fmt.Sprintf("Made %s the new alliance leader.", targetMember.Nick))
				})

				_ = w.Handle("❌", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
					_ = s.ChannelMessageDelete(reaction.ChannelID, reaction.MessageID)
					_, _ = s.ChannelMessageSend(e.ChannelID, "Cancelled that.")
				})

				_ = w.Spawn()
			} else {
				_ = s.GuildMemberRoleAdd(e.GuildID, e.Mentions[0].ID, internal.CommodoreRoleId)

				_, _ = s.ChannelMessageSend(e.ChannelID, fmt.Sprintf("Made %s a commodore.", targetMember.Nick))
			}

		},
	}
	return cmd
}
