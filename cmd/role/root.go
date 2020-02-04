package role

import (
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/cfi2017/s157-bot/internal"
	"github.com/spf13/cobra"
)

func RootCmdFactory(s *discordgo.Session, e *discordgo.MessageCreate) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Role related commands.",
		Long:  `Role related commands.`,
		Example: `
!role prompt create <roleId> <message> -- create a new role prompt in the current channel
`,
		Args: cobra.MinimumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if e.Author.ID != internal.BotOwnerId {
				return errors.New("in the current configuration, this can only be run by the bot administrator")
			}
			if e.ChannelID != internal.CrewAssignmentChannelId {
				_, _ = s.ChannelMessageSend(e.ChannelID, "Please run this command on a server.")
				return errors.New("is private channel")
			}
			return nil
		},
	}

	cmd.AddCommand(createPromptCmdFactory(s, e))

	return cmd
}
