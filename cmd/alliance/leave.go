package alliance

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/s157-bot/internal"
	"github.com/spf13/cobra"
)

func leaveCmdFactory(s *discordgo.Session, e *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "leave",
		Short: "Leave your current alliance.",
		Long: `
Leaves your alliance.
- If you are a member or commodore, this will remove your guild tag and your Alliance role.
- If you are a leader, this will disband your guild if you are the only member. 
  If there are other members, you will need to promote someone else first.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if internal.IsDM(s, e.ChannelID) {
				_, _ = s.ChannelMessageSend(e.ChannelID, "This command can only be run from a guild.")
				return
			}

			if !internal.HasRole(e.Member, internal.MemberRoleId) {
				_, _ = s.ChannelMessageSend(e.ChannelID, "You are not a member of any alliance.")
				return
			}

			if internal.HasRole(e.Member, internal.LeaderRoleId) {
				prefix := strings.Split(e.Member.Nick, " ")[0]
				members := internal.GetMembers(s, e.GuildID)
				m := make([]*discordgo.Member, 0)
				for _, member := range members {
					if member.User.ID == e.Author.ID {
						continue
					}
					if !strings.HasPrefix(member.Nick, prefix) {
						continue
					}
					for _, r := range member.Roles {
						if r == internal.MemberRoleId {
							m = append(m, member)
						}
					}
				}
				if len(m) > 0 {
					_, _ = s.ChannelMessageSend(e.ChannelID, "Your current alliance has members. Promote someone before leaving.")
					return
				}

				_ = s.GuildMemberRoleRemove(e.GuildID, e.Author.ID, internal.LeaderRoleId)
			}

			_ = s.GuildMemberRoleRemove(e.GuildID, e.Author.ID, internal.MemberRoleId)
			_ = s.GuildMemberNickname(e.GuildID, e.Author.ID, strings.Split(e.Member.Nick, "] ")[1])

		},
	}
	return cmd
}
