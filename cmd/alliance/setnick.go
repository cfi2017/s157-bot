package alliance

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/s157-bot/internal"
	"github.com/spf13/cobra"
)

func setnickCmdFactory(s *discordgo.Session, e *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setnick",
		Short: "Sets your nickname.",
		Long: `
Sets your nickname.
`,
		Args: cobra.ExactArgs(1),
	}
	user := cmd.Flags().StringP("user", "u", "", "user to set the nickname for. must be a mention.")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		var err error
		userId := e.Author.ID
		if *user != "" {
			if !internal.HasRole(e.Member, internal.StaffId) {
				_, _ = s.ChannelMessageSend(e.ChannelID, "You do not have permission to do this.")
				return
			}
			if len(e.Mentions) != 0 {
				_, _ = s.ChannelMessageSend(e.ChannelID, "Please mention exactly one user.")
				return
			}
			userId = e.Mentions[0].ID
		}
		if !internal.HasRole(e.Member, internal.MemberRoleId) {
			err = s.GuildMemberNickname(e.GuildID, userId, args[0])
		} else {
			tag := strings.Split(e.Member.Nick, " ")[0]
			err = s.GuildMemberNickname(e.GuildID, userId, tag+" "+args[0])
		}
		if err != nil {
			_, _ = s.ChannelMessageSend(e.ChannelID, "Could not set nickname. Unhandled exception.")
		}
	}

	return cmd
}
