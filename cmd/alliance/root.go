package alliance

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/dgwidgets"
	"github.com/cfi2017/s157-bot/internal"
	"github.com/spf13/cobra"
)

func RootCmdFactory(s *discordgo.Session, e *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alliance",
		Short: "Alliance related commands.",
		Long:  `Returns the bots gateway ping.`,
		Example: `
!alliance <tag> -- join an alliance
!alliance leave -- leave an alliance
!alliance setnick <nick> -- set your own nickname
!alliance promote @username -- promote another user
!alliance demote @username -- demote (or kick) another user
!alliance kick @username
`,
		Args: cobra.ExactArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if e.ChannelID != internal.CrewAssignmentChannelId {
				_, _ = s.ChannelMessageSend(e.ChannelID, "Please run this command on a server.")
				return errors.New("is private channel")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if utf8.RuneCountInString(args[0]) > 4 {
				_, _ = s.ChannelMessageSend(e.ChannelID, "your alliance tag must be at most four characters")
				return
			}
			tag := strings.ToUpper(args[0])
			if internal.HasRole(e.Member, internal.MemberRoleId) {
				_, _ = s.ChannelMessageSend(e.ChannelID, "You are already in an alliance.")
				return
			}

			// check for members in same guild
			guildMembers := internal.GetMembers(s, e.GuildID)
			var leader *discordgo.Member
			// if none - add and make rep
			for _, member := range guildMembers {
				for _, role := range member.Roles {
					if role == internal.LeaderRoleId {
						if strings.HasPrefix(member.Nick, "["+tag+"]") {
							leader = member
						}
					}
				}
			}

			if leader == nil {
				// add and make rep
				err := s.GuildMemberNickname(e.GuildID, e.Author.ID, "["+tag+"] "+e.Author.Username)
				if err != nil {
					_, _ = s.ChannelMessageSend(e.ChannelID, "couldn't change your nickname. aborting.")
					log.Println(err)
					return
				}
				err = s.GuildMemberRoleAdd(e.GuildID, e.Author.ID, internal.MemberRoleId)
				if err != nil {
					log.Println(err)
					_, _ = s.ChannelMessageSend(e.ChannelID, "couldn't give you the alliance role.")
				}
				err = s.GuildMemberRoleAdd(e.GuildID, e.Author.ID, internal.LeaderRoleId)
				if err != nil {
					log.Println(err)
					_, _ = s.ChannelMessageSend(e.ChannelID, "couldn't give you the alliance leader role.")
				}
				if internal.HasRole(e.Member, internal.NameNotChanged) {
					err = s.GuildMemberRoleRemove(e.GuildID, e.Author.ID, internal.NameNotChanged)
					if err != nil {
						_, _ = s.ChannelMessageSend(e.ChannelID, "Could not remove redundant role.")
					}
				}
				_, _ = s.ChannelMessageSend(e.ChannelID, "You are the first person to join this guild. Welcome, admiral.")
			} else {
				_, _ = s.ChannelMessageSend(e.ChannelID, "asking your representative for permission")
				channel, err := s.UserChannelCreate(leader.User.ID)
				if err != nil {
					_, _ = s.ChannelMessageSend(e.ChannelID, "couldn't ask your rep. please contact an admin.")
					return
				}
				embed := &discordgo.MessageEmbed{
					Title:       "Someone wants to join your alliance!",
					Description: fmt.Sprintf("%s wants to join your alliance.", e.Author.Username),
				}

				w := dgwidgets.NewWidget(s, channel.ID, embed)
				_ = w.Handle("✅", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
					// adding new user to alliance
					_ = s.GuildMemberNickname(e.GuildID, e.Author.ID, "["+tag+"] "+e.Author.Username)

					_ = s.GuildMemberRoleAdd(e.GuildID, e.Author.ID, internal.MemberRoleId)

					if internal.HasRole(e.Member, internal.NameNotChanged) {
						err = s.GuildMemberRoleRemove(e.GuildID, e.Author.ID, internal.NameNotChanged)
						if err != nil {
							_, _ = s.ChannelMessageSend(e.ChannelID, "Could not remove role.")
						}
					}
				})

				_ = w.Handle("❌", func(widget *dgwidgets.Widget, reaction *discordgo.MessageReaction) {
					_ = s.ChannelMessageDelete(reaction.ChannelID, reaction.MessageID)
				})

				_ = w.Spawn()
			}
		},
	}
	cmd.AddCommand(setnickCmdFactory(s, e))
	cmd.AddCommand(leaveCmdFactory(s, e))
	cmd.AddCommand(promoteCmdFactory(s, e))
	cmd.AddCommand(demoteCmdFactory(s, e))
	cmd.AddCommand(kickCmdFactory(s, e))
	return cmd
}
